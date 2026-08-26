package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/XplnHUB/Les-Go/protocol"
	"github.com/gorilla/websocket"
)

// resetGlobalState clears the package-level session maps between tests,
// since they're shared global state (sync.Map, not per-server).
func resetGlobalState(t *testing.T) {
	t.Helper()
	onlineUsers.Range(func(key, _ interface{}) bool {
		onlineUsers.Delete(key)
		return true
	})
	activeChats.Range(func(key, _ interface{}) bool {
		activeChats.Delete(key)
		return true
	})
}

func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	resetGlobalState(t)
	srv := httptest.NewServer(newServeMux())
	t.Cleanup(srv.Close)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	return srv, wsURL
}

func dialAndRegister(t *testing.T, wsURL, id string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	if err := conn.WriteJSON(protocol.NewPacket(protocol.TypeRegister, id, "", "")); err != nil {
		t.Fatalf("register WriteJSON() error = %v", err)
	}
	waitForRegistration(t, id)
	return conn
}

// waitForRegistration blocks until the server has actually processed id's
// register packet (registration happens asynchronously over the socket, so
// a test that writes it and immediately acts as a peer would otherwise race
// the server's read loop).
func waitForRegistration(t *testing.T, id string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := onlineUsers.Load(id); ok {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to register", id)
}

func readPacket(t *testing.T, conn *websocket.Conn, timeout time.Duration) (protocol.Packet, error) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, data, err := conn.ReadMessage()
	if err != nil {
		return protocol.Packet{}, err
	}
	var p protocol.Packet
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	return p, nil
}

func TestConnectRequestIsForwardedToTarget(t *testing.T) {
	_, wsURL := newTestServer(t)

	connA := dialAndRegister(t, wsURL, "1111111111")
	connB := dialAndRegister(t, wsURL, "2222222222")

	if err := connA.WriteJSON(protocol.NewPacket(protocol.TypeConnectRequest, "1111111111", "2222222222", "")); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	p, err := readPacket(t, connB, 2*time.Second)
	if err != nil {
		t.Fatalf("expected B to receive connect_request, got error: %v", err)
	}
	if p.Type != protocol.TypeConnectRequest || p.From != "1111111111" || p.To != "2222222222" {
		t.Errorf("forwarded packet = %+v, want connect_request from 1111111111 to 2222222222", p)
	}
}

func TestSpoofedSenderIsRejected(t *testing.T) {
	_, wsURL := newTestServer(t)

	connA := dialAndRegister(t, wsURL, "1111111111")
	connB := dialAndRegister(t, wsURL, "2222222222")
	dialAndRegister(t, wsURL, "3333333333") // the identity A will try to impersonate

	// A claims to be 3333333333 while actually registered as 1111111111.
	forged := protocol.NewPacket(protocol.TypeConnectRequest, "3333333333", "2222222222", "")
	if err := connA.WriteJSON(forged); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	// B should never receive the forged request. Prove it by sending a
	// legitimate, distinguishable packet from A right after, and confirming
	// that's the first (and only) thing B sees.
	if err := connA.WriteJSON(protocol.NewPacket(protocol.TypeConnectRequest, "1111111111", "2222222222", "")); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	p, err := readPacket(t, connB, 2*time.Second)
	if err != nil {
		t.Fatalf("expected the legitimate request to arrive, got error: %v", err)
	}
	if p.From != "1111111111" {
		t.Errorf("first packet B received had From = %q, want %q (the forged packet should have been dropped)", p.From, "1111111111")
	}
}

func TestOfflineTargetGetsErrorPacket(t *testing.T) {
	_, wsURL := newTestServer(t)

	connA := dialAndRegister(t, wsURL, "1111111111")

	if err := connA.WriteJSON(protocol.NewPacket(protocol.TypeConnectRequest, "1111111111", "9999999999", "")); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	p, err := readPacket(t, connA, 2*time.Second)
	if err != nil {
		t.Fatalf("expected an error packet back, got error: %v", err)
	}
	if p.Type != protocol.TypeError || p.Payload != "target_offline" {
		t.Errorf("packet = %+v, want type=%q payload=%q", p, protocol.TypeError, "target_offline")
	}
}

func TestRateLimitExceededGetsErrorPacket(t *testing.T) {
	origMax, origWindow := rateLimitMax, rateLimitWindow
	rateLimitMax = 2
	rateLimitWindow = time.Minute
	t.Cleanup(func() { rateLimitMax, rateLimitWindow = origMax, origWindow })

	_, wsURL := newTestServer(t)
	connA := dialAndRegister(t, wsURL, "1111111111")
	dialAndRegister(t, wsURL, "2222222222")

	// First 2 requests are within the limit; the 3rd should be rejected with
	// a rate_limited error instead of being forwarded/dropped silently.
	for i := 0; i < 2; i++ {
		if err := connA.WriteJSON(protocol.NewPacket(protocol.TypeMessage, "1111111111", "2222222222", "hi")); err != nil {
			t.Fatalf("WriteJSON() error = %v", err)
		}
	}
	if err := connA.WriteJSON(protocol.NewPacket(protocol.TypeMessage, "1111111111", "2222222222", "hi")); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	p, err := readPacket(t, connA, 2*time.Second)
	if err != nil {
		t.Fatalf("expected a rate_limited error back, got error: %v", err)
	}
	if p.Type != protocol.TypeError || p.Payload != "rate_limited" {
		t.Errorf("packet = %+v, want type=%q payload=%q", p, protocol.TypeError, "rate_limited")
	}
}

func TestDeadSessionCleanupNotifiesPeer(t *testing.T) {
	origTimeout, origInterval := heartbeatTimeout, cleanupInterval
	heartbeatTimeout = 100 * time.Millisecond
	cleanupInterval = 50 * time.Millisecond
	t.Cleanup(func() { heartbeatTimeout, cleanupInterval = origTimeout, origInterval })

	_, wsURL := newTestServer(t)
	connA := dialAndRegister(t, wsURL, "1111111111")
	connB := dialAndRegister(t, wsURL, "2222222222")

	// Pair them up as an active chat, as connect_accept would.
	if err := connA.WriteJSON(protocol.NewPacket(protocol.TypeConnectAccept, "1111111111", "2222222222", "")); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	if _, err := readPacket(t, connB, 2*time.Second); err != nil {
		t.Fatalf("expected B to receive connect_accept, got error: %v", err)
	}

	stopCleanup := make(chan struct{})
	t.Cleanup(func() { close(stopCleanup) })
	go startCleanupLoop(stopCleanup)

	// Keep B alive with real heartbeats so only A (silent from here on) gets
	// reaped by the cleanup loop.
	stopHeartbeat := make(chan struct{})
	t.Cleanup(func() { close(stopHeartbeat) })
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				connB.WriteJSON(protocol.NewPacket(protocol.TypeHeartbeat, "2222222222", "", ""))
			case <-stopHeartbeat:
				return
			}
		}
	}()

	// A goes silent (no more heartbeats). B should eventually get a
	// disconnect notification once the cleanup loop evicts A.
	p, err := readPacket(t, connB, 3*time.Second)
	if err != nil {
		t.Fatalf("expected B to receive a disconnect notification, got error: %v", err)
	}
	if p.Type != protocol.TypeDisconnect || p.From != "1111111111" {
		t.Errorf("packet = %+v, want type=%q from=%q", p, protocol.TypeDisconnect, "1111111111")
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := srv.Client().Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %q, want %q", body["status"], "ok")
	}
}
