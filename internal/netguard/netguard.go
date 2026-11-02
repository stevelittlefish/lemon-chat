// Package netguard provides an HTTP client for fetching model-chosen URLs that
// refuses to connect to loopback, private, link-local and other non-public
// addresses. A prompt-injected page could otherwise make the model read
// services on the LAN (routers, home automation, cameras) or cloud metadata
// endpoints, then leak the result through a second fetch.
package netguard

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

// ErrBlocked is returned (wrapped) when a connection to a non-public address
// is refused.
var ErrBlocked = errors.New("address is not public")

// cgnat is the shared address space (RFC 6598), not covered by IsPrivate.
var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// IsPublic reports whether ip is a globally routable unicast address.
func IsPublic(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsValid() &&
		ip.IsGlobalUnicast() &&
		!ip.IsPrivate() &&
		!ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() &&
		!cgnat.Contains(ip)
}

// control runs after DNS resolution with the concrete address being dialled,
// so it also covers redirects and DNS rebinding.
func control(network, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("netguard: %w: %q", ErrBlocked, address)
	}
	if !IsPublic(ap.Addr()) {
		return fmt.Errorf("netguard: %w: %s", ErrBlocked, ap.Addr())
	}
	return nil
}

// NewClient returns an HTTP client for fetching untrusted URLs. Unless
// allowPrivate is set, it refuses connections to non-public addresses. It
// never uses an environment proxy, since the proxy address would itself be
// checked (and the proxy could reach anything).
func NewClient(allowPrivate bool) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if !allowPrivate {
		dialer.Control = control
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			MaxIdleConns:          20,
			IdleConnTimeout:       90 * time.Second,
		},
	}
}
