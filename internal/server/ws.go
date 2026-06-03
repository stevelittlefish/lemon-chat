package server

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type Hub struct {
	mu      sync.Mutex
	clients map[net.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[net.Conn]struct{})}
}

func (h *Hub) add(conn net.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) remove(conn net.Conn) {
	h.mu.Lock()
	_, ok := h.clients[conn]
	if ok {
		delete(h.clients, conn)
	}
	h.mu.Unlock()
	if ok {
		conn.Close()
	}
}

func (h *Hub) broadcast(msg []byte) {
	h.mu.Lock()
	conns := make([]net.Conn, 0, len(h.clients))
	for c := range h.clients {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		if err := sendTextFrame(c, msg); err != nil {
			h.remove(c)
		}
	}
}

func (h *Hub) BroadcastTitleUpdate(convID int64, title string) {
	data, _ := json.Marshal(map[string]any{
		"type":  "conversation_titled",
		"id":    convID,
		"title": title,
	})
	h.broadcast(data)
}

func (h *Hub) BroadcastCompletionTitleUpdate(compID int64, title string) {
	data, _ := json.Marshal(map[string]any{
		"type":  "completion_titled",
		"id":    compID,
		"title": title,
	})
	h.broadcast(data)
}

func (h *Hub) BroadcastConversationListChanged() {
	data, _ := json.Marshal(map[string]any{"type": "conversations_changed"})
	h.broadcast(data)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := s.store.SessionUserID(cookie.Value); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Header.Get("Upgrade") != "websocket" {
		http.Error(w, "expected websocket upgrade", http.StatusBadRequest)
		return
	}
	key := r.Header.Get("Sec-Websocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		internalError(w, fmt.Errorf("websocket hijacker not supported"))
		return
	}
	conn, buf, err := hijacker.Hijack()
	if err != nil {
		internalError(w, fmt.Errorf("hijack: %w", err))
		return
	}

	accept := wsAcceptKey(key)
	_, err = buf.WriteString("HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n")
	if err != nil {
		conn.Close()
		return
	}
	if err := buf.Flush(); err != nil {
		conn.Close()
		return
	}

	s.hub.add(conn)
	go func() {
		defer s.hub.remove(conn)
		drainFrames(conn)
	}()
}

func wsAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func sendTextFrame(conn net.Conn, payload []byte) error {
	n := len(payload)
	var header []byte
	switch {
	case n <= 125:
		header = []byte{0x81, byte(n)}
	case n <= 65535:
		header = []byte{0x81, 126, byte(n >> 8), byte(n)}
	default:
		header = []byte{0x81, 127, 0, 0, 0, 0,
			byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
	}
	if _, err := conn.Write(header); err != nil {
		return err
	}
	_, err := conn.Write(payload)
	return err
}

// drainFrames reads incoming WebSocket frames and handles control frames until
// the connection closes or errors. Clients must mask their frames per RFC 6455.
func drainFrames(conn net.Conn) {
	for {
		hdr := make([]byte, 2)
		if _, err := io.ReadFull(conn, hdr); err != nil {
			return
		}
		opcode := hdr[0] & 0x0F
		masked := (hdr[1] & 0x80) != 0
		length := int(hdr[1] & 0x7F)

		if length == 126 {
			ext := make([]byte, 2)
			if _, err := io.ReadFull(conn, ext); err != nil {
				return
			}
			length = int(ext[0])<<8 | int(ext[1])
		} else if length == 127 {
			ext := make([]byte, 8)
			if _, err := io.ReadFull(conn, ext); err != nil {
				return
			}
			length = int(ext[4])<<24 | int(ext[5])<<16 | int(ext[6])<<8 | int(ext[7])
		}

		var maskKey [4]byte
		if masked {
			if _, err := io.ReadFull(conn, maskKey[:]); err != nil {
				return
			}
		}

		if length > 4096 {
			return
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return
		}
		if masked {
			for i := range payload {
				payload[i] ^= maskKey[i%4]
			}
		}

		switch opcode {
		case 0x8: // close
			return
		case 0x9: // ping — respond with pong
			if length > 125 {
				return
			}
			pong := append([]byte{0x8A, byte(length)}, payload...)
			conn.Write(pong) //nolint:errcheck
		}
	}
}
