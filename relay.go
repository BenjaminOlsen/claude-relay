package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"nhooyr.io/websocket"
)

type clientMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("websocket accept: %v", err)
		return
	}
	defer conn.CloseNow()

	session := NewSession(s.dir)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	log.Printf("client connected (dir: %s)", s.dir)

	for {
		_, raw, err := conn.Read(ctx)
		if err != nil {
			log.Printf("client disconnected: %v", err)
			return
		}

		var msg clientMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("bad message: %v", err)
			continue
		}

		if msg.Content == "" {
			continue
		}

		// Run claude and stream response back to client
		err = session.Send(msg.Content, func(line []byte) {
			if wErr := conn.Write(ctx, websocket.MessageText, line); wErr != nil {
				log.Printf("ws write: %v", wErr)
			}
		})
		if err != nil {
			log.Printf("session error: %v", err)
			errMsg, _ := json.Marshal(map[string]string{
				"type":  "error",
				"error": err.Error(),
			})
			conn.Write(ctx, websocket.MessageText, errMsg)
		}
	}
}
