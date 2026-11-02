package netguard

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestIsPublic(t *testing.T) {
	for _, tt := range []struct {
		addr string
		want bool
	}{
		{"8.8.8.8", true},
		{"2606:4700:4700::1111", true},
		{"127.0.0.1", false},
		{"::1", false},
		{"10.1.2.3", false},
		{"172.16.0.1", false},
		{"192.168.1.10", false},
		{"169.254.169.254", false}, // cloud metadata
		{"100.64.0.1", false},      // CGNAT / Tailscale
		{"0.0.0.0", false},
		{"fe80::1", false},
		{"fd00::1", false},
		{"::ffff:192.168.1.1", false}, // IPv4-mapped private
		{"224.0.0.1", false},
	} {
		if got := IsPublic(netip.MustParseAddr(tt.addr)); got != tt.want {
			t.Errorf("IsPublic(%s) = %v, want %v", tt.addr, got, tt.want)
		}
	}
}

func TestClientBlocksLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	_, err := NewClient(false).Get(srv.URL)
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("err = %v, want ErrBlocked", err)
	}

	resp, err := NewClient(true).Get(srv.URL)
	if err != nil {
		t.Fatalf("allowPrivate client: %v", err)
	}
	resp.Body.Close()
}
