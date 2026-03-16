package main

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"github.com/XplnHUB/Les-Go/protocol"
)

// StartHeartbeat begins sending heartbeat packets at the specified interval.
// It stops when the context is cancelled.
func StartHeartbeat(ctx context.Context, conn *websocket.Conn, myID string) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			heartbeat := protocol.NewPacket(protocol.TypeHeartbeat, myID, "", "")
			err := conn.WriteJSON(heartbeat)
			if err != nil {
				return // Connection likely closed
			}
		case <-ctx.Done():
			return
		}
	}
}
