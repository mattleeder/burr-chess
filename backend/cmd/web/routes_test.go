package main

import (
	"net/http"
	"testing"
)

// TestRoutes_RootReturns404 verifies that undefined paths fall through to the
// root handler which returns 404.
func TestRoutes_RootReturns404(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())

	resp := ts.get(t, "/nonexistent-path-xyz")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for unknown path", resp.StatusCode)
	}
}
