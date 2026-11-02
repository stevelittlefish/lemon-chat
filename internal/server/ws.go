package server

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// wsSendBuffer is how many frames may queue for one client before it is
// considered stalled and dropped.
const wsSendBuffer = 32

// wsClient is one WebSocket connection. All writes go through send and are
// performed by a single writer goroutine, so frames never interleave and a
// slow client never blocks the goroutine that triggered the event.
type wsClient struct {
	conn   net.Conn
	userID int64
	send   chan []byte   // complete, already-framed messages
	done   chan struct{} // closed when the client is removed
}

type Hub struct {
	mu      sync.Mutex
	clients map[*wsClient]struct{}
	store   *store.Store // resolves conversation/completion owners
}

func NewHub(st *store.Store) *Hub {
	return &Hub{clients: make(map[*wsClient]struct{}), store: st}
}

func (h *Hub) add(conn net.Conn, userID int64) *wsClient {
	c := &wsClient{conn: conn, userID: userID, send: make(chan []byte, wsSendBuffer), done: make(chan struct{})}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	go c.writeLoop(h)
	return c
}

func (h *Hub) remove(c *wsClient) {
	h.mu.Lock()
	_, ok := h.clients[c]
	if ok {
		delete(h.clients, c)
	}
	h.mu.Unlock()
	if ok {
		close(c.done)
		c.conn.Close()
	}
}

func (c *wsClient) writeLoop(h *Hub) {
	for {
		select {
		case frame := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)) //nolint:errcheck
			if _, err := c.conn.Write(frame); err != nil {
				h.remove(c)
				return
			}
		case <-c.done:
			return
		}
	}
}

// enqueue queues a frame without blocking; a client whose buffer is full is
// dropped (the frontend reconnects and refetches state).
func (h *Hub) enqueue(c *wsClient, frame []byte) {
	select {
	case c.send <- frame:
	case <-c.done:
	default:
		h.remove(c)
	}
}

// sendToUser delivers msg to every connection belonging to userID.
func (h *Hub) sendToUser(userID int64, msg []byte) {
	frame := textFrame(msg)
	h.mu.Lock()
	var targets []*wsClient
	for c := range h.clients {
		if c.userID == userID {
			targets = append(targets, c)
		}
	}
	h.mu.Unlock()
	for _, c := range targets {
		h.enqueue(c, frame)
	}
}

// sendToAll delivers msg to every connection. Only use it for events that
// carry no user data.
func (h *Hub) sendToAll(msg []byte) {
	frame := textFrame(msg)
	h.mu.Lock()
	targets := make([]*wsClient, 0, len(h.clients))
	for c := range h.clients {
		targets = append(targets, c)
	}
	h.mu.Unlock()
	for _, c := range targets {
		h.enqueue(c, frame)
	}
}

func (h *Hub) sendToConversationOwner(convID int64, msg []byte) {
	userID, err := h.store.ConversationOwner(convID)
	if err != nil {
		log.Printf("ws: resolving owner of conversation id=%d: %v", convID, err)
		return
	}
	h.sendToUser(userID, msg)
}

func (h *Hub) BroadcastTitleUpdate(convID int64, title string) {
	data, _ := json.Marshal(map[string]any{
		"type":  "conversation_titled",
		"id":    convID,
		"title": title,
	})
	h.sendToConversationOwner(convID, data)
}

func (h *Hub) BroadcastCompletionTitleUpdate(compID int64, title string) {
	data, _ := json.Marshal(map[string]any{
		"type":  "completion_titled",
		"id":    compID,
		"title": title,
	})
	userID, err := h.store.CompletionOwner(compID)
	if err != nil {
		log.Printf("ws: resolving owner of completion id=%d: %v", compID, err)
		return
	}
	h.sendToUser(userID, data)
}

func (h *Hub) BroadcastConversationBackground(convID, attID int64) {
	data, _ := json.Marshal(map[string]any{
		"type":          "conversation_background",
		"id":            convID,
		"attachment_id": attID,
	})
	h.sendToConversationOwner(convID, data)
}

// BroadcastConversationListChanged tells userID's clients to refetch their
// conversation list.
func (h *Hub) BroadcastConversationListChanged(userID int64) {
	data, _ := json.Marshal(map[string]any{"type": "conversations_changed"})
	h.sendToUser(userID, data)
}

// BroadcastConversationListChangedAll tells every client to refetch its
// conversation list. The event carries no data, so it is safe to send to all
// users when the affected owners aren't tracked (e.g. the cleanup worker).
func (h *Hub) BroadcastConversationListChangedAll() {
	data, _ := json.Marshal(map[string]any{"type": "conversations_changed"})
	h.sendToAll(data)
}

func (h *Hub) BroadcastAttachmentReady(att *store.Attachment, background bool) {
	data, _ := json.Marshal(map[string]any{
		"type":            "attachment_ready",
		"id":              att.ID,
		"conversation_id": att.ConversationID,
		"title":           att.Title,
		"filename":        att.Filename,
		"mime_type":       att.MimeType,
		"background":      background,
	})
	h.sendToConversationOwner(att.ConversationID, data)
}

func (h *Hub) BroadcastAttachmentError(convID, attID int64, errMsg string) {
	data, _ := json.Marshal(map[string]any{
		"type":            "attachment_error",
		"id":              attID,
		"conversation_id": convID,
		"error":           errMsg,
	})
	h.sendToConversationOwner(convID, data)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := s.store.SessionUserID(cookie.Value)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if origin := r.Header.Get("Origin"); origin != "" {
		originURL, err := url.Parse(origin)
		if err != nil || !strings.EqualFold(originURL.Host, r.Host) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
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

	client := s.hub.add(conn, userID)
	go func() {
		defer s.hub.remove(client)
		drainFrames(client, s.hub)
	}()
}

func wsAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// textFrame builds an unmasked, unfragmented WebSocket text frame.
func textFrame(payload []byte) []byte {
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
	return append(header, payload...)
}

// drainFrames reads incoming WebSocket frames and handles control frames until
// the connection closes or errors. Clients must mask their frames per RFC 6455.
func drainFrames(c *wsClient, h *Hub) {
	conn := c.conn
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
			h.enqueue(c, pong)
		}
	}
}
