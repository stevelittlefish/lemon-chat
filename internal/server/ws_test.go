package server

import (
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// readFrame reads one unmasked server text frame (payload ≤ 65535 bytes).
func readFrame(t *testing.T, conn net.Conn) (string, bool) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond)) //nolint:errcheck
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return "", false
	}
	n := int(hdr[1] & 0x7F)
	if n == 126 {
		ext := make([]byte, 2)
		if _, err := io.ReadFull(conn, ext); err != nil {
			t.Fatalf("read ext length: %v", err)
		}
		n = int(ext[0])<<8 | int(ext[1])
	}
	payload := make([]byte, n)
	if _, err := io.ReadFull(conn, payload); err != nil {
		t.Fatalf("read payload: %v", err)
	}
	return string(payload), true
}

func TestHubRoutesEventsToOwner(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	alice, err := st.CreateUser("alice", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	bob, err := st.CreateUser("bob", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	model := "m"
	conv, err := st.CreateConversation(alice.ID, nil, &model, nil)
	if err != nil {
		t.Fatal(err)
	}

	hub := NewHub(st)
	aliceServer, aliceClient := net.Pipe()
	bobServer, bobClient := net.Pipe()
	a := hub.add(aliceServer, alice.ID)
	b := hub.add(bobServer, bob.ID)
	defer hub.remove(a)
	defer hub.remove(b)

	hub.BroadcastTitleUpdate(conv.ID, "secret plans")

	got, ok := readFrame(t, aliceClient)
	if !ok || !strings.Contains(got, "secret plans") {
		t.Fatalf("owner did not receive title event, got %q ok=%v", got, ok)
	}
	if got, ok := readFrame(t, bobClient); ok {
		t.Fatalf("non-owner received event: %q", got)
	}

	hub.BroadcastConversationListChangedAll()
	for name, c := range map[string]net.Conn{"alice": aliceClient, "bob": bobClient} {
		if got, ok := readFrame(t, c); !ok || !strings.Contains(got, "conversations_changed") {
			t.Fatalf("%s did not receive list-changed event, got %q", name, got)
		}
	}
}

func TestHubDropsStalledClient(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	hub := NewHub(st)
	server, client := net.Pipe()
	defer client.Close()
	c := hub.add(server, 1)

	// Nobody reads from client, so the writer blocks on the first frame and
	// the buffer fills. Broadcasting must never block the caller.
	done := make(chan struct{})
	go func() {
		for i := 0; i < wsSendBuffer+10; i++ {
			hub.BroadcastConversationListChanged(1)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("broadcast blocked on a stalled client")
	}

	hub.mu.Lock()
	_, still := hub.clients[c]
	hub.mu.Unlock()
	if still {
		t.Fatal("stalled client was not removed")
	}
}
