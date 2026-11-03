package server

import (
	"net/http/httptest"
	"testing"
)

func TestRequestIsHTTPS(t *testing.T) {
	plain := httptest.NewRequest("POST", "http://lemon/api/login", nil)
	if requestIsHTTPS(plain) {
		t.Error("plain HTTP request reported as HTTPS")
	}
	proxied := httptest.NewRequest("POST", "http://localhost:8069/api/login", nil)
	proxied.Header.Set("X-Forwarded-Proto", "https")
	if !requestIsHTTPS(proxied) {
		t.Error("X-Forwarded-Proto: https not recognised")
	}
	proxiedHTTP := httptest.NewRequest("POST", "http://localhost:8069/api/login", nil)
	proxiedHTTP.Header.Set("X-Forwarded-Proto", "http")
	if requestIsHTTPS(proxiedHTTP) {
		t.Error("X-Forwarded-Proto: http reported as HTTPS")
	}
	direct := httptest.NewRequest("POST", "https://lemon/api/login", nil)
	if !requestIsHTTPS(direct) {
		t.Error("direct TLS request not recognised")
	}
}
