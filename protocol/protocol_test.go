package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewPacketSetsFields(t *testing.T) {
	before := time.Now().Unix()
	p := NewPacket(TypeMessage, "1111111111", "2222222222", "cGF5bG9hZA==")
	after := time.Now().Unix()

	if p.Type != TypeMessage {
		t.Errorf("Type = %q, want %q", p.Type, TypeMessage)
	}
	if p.From != "1111111111" {
		t.Errorf("From = %q, want %q", p.From, "1111111111")
	}
	if p.To != "2222222222" {
		t.Errorf("To = %q, want %q", p.To, "2222222222")
	}
	if p.Payload != "cGF5bG9hZA==" {
		t.Errorf("Payload = %q, want %q", p.Payload, "cGF5bG9hZA==")
	}
	if p.Timestamp < before || p.Timestamp > after {
		t.Errorf("Timestamp = %d, want between %d and %d", p.Timestamp, before, after)
	}
}

func TestPacketJSONRoundTrip(t *testing.T) {
	original := Packet{
		Type:      TypeAESKey,
		From:      "1111111111",
		To:        "2222222222",
		Payload:   "c29tZS1jaXBoZXJ0ZXh0",
		Timestamp: 1234567890,
		MessageID: "msg-1",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded Packet
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded != original {
		t.Errorf("round-tripped packet = %+v, want %+v", decoded, original)
	}
}

func TestPacketOmitsEmptyOptionalFields(t *testing.T) {
	p := Packet{Type: TypeRegister, From: "1111111111"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	for _, field := range []string{"payload", "timestamp", "message_id"} {
		if _, present := raw[field]; present {
			t.Errorf("field %q present in JSON for zero value, want omitted: %s", field, data)
		}
	}
	if _, present := raw["to"]; !present {
		t.Errorf("field %q missing from JSON, want present (no omitempty tag): %s", "to", data)
	}
}

func TestPacketTypeConstantsAreDistinct(t *testing.T) {
	types := []string{
		TypeRegister, TypeConnectRequest, TypeConnectAccept, TypeConnectReject,
		TypePublicKey, TypeAESKey, TypeMessage, TypeACK, TypeHeartbeat,
		TypeDisconnect, TypeError,
	}
	seen := make(map[string]bool, len(types))
	for _, ty := range types {
		if seen[ty] {
			t.Errorf("duplicate packet type constant value: %q", ty)
		}
		seen[ty] = true
	}
}
