package embedtest

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Host is a fake model host: it serves the two files of the tiny model over
// TLS, like the hub does for the real one. Its Transport trusts only this
// server and dials only the addresses it is told to, so a test can prove that
// nothing else is contacted.
type Host struct {
	*httptest.Server
	Weights, Tokenizer []byte
	// Handler, when set, sees each request first and returns true when it
	// answered it.
	Handler func(w http.ResponseWriter, r *http.Request) bool

	mu       sync.Mutex
	allowed  map[string]bool
	requests int
	stray    []string
}

// NewHost starts a host serving the tiny model of seed.
func NewHost(t testing.TB, seed int64) *Host {
	t.Helper()
	names, shapes := Tensors(Tiny)
	h := &Host{Weights: Safetensors(names, shapes, seed, nil), Tokenizer: Tokenizer(nil), allowed: map[string]bool{}}
	h.Server = httptest.NewTLSServer(http.HandlerFunc(h.serve))
	t.Cleanup(h.Close)
	h.Allow(h.Listener.Addr().String())
	return h
}

func (h *Host) serve(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.requests++
	h.mu.Unlock()
	if h.Handler != nil && h.Handler(w, r) {
		return
	}
	switch r.URL.Path {
	case "/model.safetensors":
		_, _ = w.Write(h.Weights)
	case "/tokenizer.json":
		_, _ = w.Write(h.Tokenizer)
	default:
		http.NotFound(w, r)
	}
}

// Requests is how many requests the host has answered.
func (h *Host) Requests() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.requests
}

// Allow lets the transport dial addr, for a second host a redirect leads to.
func (h *Host) Allow(addr string) {
	h.mu.Lock()
	h.allowed[addr] = true
	h.mu.Unlock()
}

// Stray lists the addresses the transport was asked to dial and refused.
func (h *Host) Stray() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.stray...)
}

// Transport trusts the certificates of the test servers and dials only the
// allowed addresses.
func (h *Host) Transport() http.RoundTripper {
	var d net.Dialer
	base := h.Client().Transport.(*http.Transport).Clone()
	base.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // test servers only
	base.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		h.mu.Lock()
		ok := h.allowed[addr]
		if !ok {
			h.stray = append(h.stray, addr)
		}
		h.mu.Unlock()
		if !ok {
			return nil, fmt.Errorf("embedtest: %s is not an allowed address", addr)
		}
		return d.DialContext(ctx, network, addr)
	}
	return base
}

// Hash is the SHA-256 of data, as hex.
func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
