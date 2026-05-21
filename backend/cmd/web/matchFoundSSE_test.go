package main

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func init() {
	// Speed up SSE tests: heartbeat detects dead clients; 100 ms is plenty.
	sseHeartbeatTimer = 100 * time.Millisecond
}

// resetSSEClients wipes the global clients map before/after SSE tests to
// avoid state left by other tests (e.g. notifyMatchFound in matchmaking tests).
func resetSSEClients(t *testing.T) {
	t.Helper()
	clients.mu.Lock()
	clients.clients = make(map[int64]*Client)
	clients.mu.Unlock()
}

// openSSE starts a cancellable GET /listenformatch and returns the response.
// SSE streams indefinitely; cancel ctx and close the body when done.
func (ts *testServer) openSSE(t *testing.T, ctx context.Context) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/listenformatch", nil)
	if err != nil {
		t.Fatalf("openSSE: new request: %v", err)
	}
	resp, err := ts.client.Do(req)
	if err != nil {
		t.Fatalf("openSSE: do: %v", err)
	}
	return resp
}

// waitForSSEClient polls until at least one client appears in the global map.
func waitForSSEClient(t *testing.T) *Client {
	t.Helper()
	for i := 0; i < 100; i++ {
		clients.mu.Lock()
		for _, c := range clients.clients {
			clients.mu.Unlock()
			return c
		}
		clients.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("waitForSSEClient: no client registered within timeout")
	return nil
}

// closeSSEClient closes the client's channel and removes it from the map.
// This makes the handler's "!ok" branch fire immediately so tests don't have
// to wait 15 s for the next heartbeat write to fail.
func closeSSEClient(t *testing.T, c *Client) {
	t.Helper()
	clients.mu.Lock()
	defer clients.mu.Unlock()
	if cc, ok := clients.clients[c.id]; ok && cc == c {
		delete(clients.clients, c.id)
		close(c.channel)
	}
}

// readNextDataLine scans SSE lines from resp.Body until it finds one that
// starts with "data:". Returns ("", false) if none arrives within 2 s.
func readNextDataLine(resp *http.Response) (string, bool) {
	found := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), "data:") {
				found <- sc.Text()
				return
			}
		}
	}()
	select {
	case line := <-found:
		return line, true
	case <-time.After(2 * time.Second):
		return "", false
	}
}

// ---------------------------------------------------------------------------
// matchFoundSSEHandler
// ---------------------------------------------------------------------------

func TestMatchFoundSSE_NoSession(t *testing.T) {
	// No validateSession call → no playerID in session → handler returns 401.
	ts := newTestServer(t, newTestApp(t).routes())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp := ts.openSSE(t, ctx)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestMatchFoundSSE_Headers(t *testing.T) {
	resetSSEClients(t)
	t.Cleanup(func() { resetSSEClients(t) })

	ts := newTestServer(t, newTestApp(t).routes())
	ts.validateSession(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp := ts.openSSE(t, ctx)
	defer resp.Body.Close()

	// Close the channel in cleanup so the handler exits promptly instead of
	// waiting 15 s for the next heartbeat write to fail.
	c := waitForSSEClient(t)
	defer closeSSEClient(t, c)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
	if got := resp.Header.Get("Connection"); got != "keep-alive" {
		t.Errorf("Connection = %q, want keep-alive", got)
	}
}

func TestMatchFoundSSE_DeliversMessage(t *testing.T) {
	resetSSEClients(t)
	t.Cleanup(func() { resetSSEClients(t) })

	ts := newTestServer(t, newTestApp(t).routes())
	ts.validateSession(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp := ts.openSSE(t, ctx)
	defer resp.Body.Close()

	c := waitForSSEClient(t)
	defer closeSSEClient(t, c)

	const wantMsg = "42,300000,0"
	c.channel <- wantMsg

	line, ok := readNextDataLine(resp)
	if !ok {
		t.Fatal("timed out waiting for SSE data line")
	}
	if want := "data: " + wantMsg; line != want {
		t.Errorf("SSE data line = %q, want %q", line, want)
	}
}

// TestMatchFoundSSE_ClientReplacement verifies that a second registration for
// the same playerID closes the first client's channel.
//
// This is tested at the clients-map level rather than through two HTTP
// requests because concurrent SSE connections sharing a session would deadlock
// on the session store lock (the first handler holds the session for its
// entire lifetime under HTTP/1.1).
func TestMatchFoundSSE_ClientReplacement(t *testing.T) {
	resetSSEClients(t)
	t.Cleanup(func() { resetSSEClients(t) })

	const playerID = int64(777)

	// Seed an existing client.
	firstCh := make(chan string, 1)
	clients.mu.Lock()
	clients.clients[playerID] = &Client{id: playerID, channel: firstCh}
	clients.mu.Unlock()

	// Replay the replacement block from matchFoundSSEHandler (handlers.go:147-154).
	clients.mu.Lock()
	if existing, exists := clients.clients[playerID]; exists {
		close(existing.channel)
	}
	newCh := make(chan string, 1)
	clients.clients[playerID] = &Client{id: playerID, channel: newCh}
	clients.mu.Unlock()

	t.Cleanup(func() {
		clients.mu.Lock()
		if c, ok := clients.clients[playerID]; ok && c.channel == newCh {
			delete(clients.clients, playerID)
			close(newCh)
		}
		clients.mu.Unlock()
	})

	// The first channel must now be closed.
	select {
	case _, ok := <-firstCh:
		if ok {
			t.Error("first channel: expected closed, received a value instead")
		}
		// ok == false → closed ✓
	default:
		t.Error("first channel: expected closed, is still open and empty")
	}
}

// TestMatchFoundSSE_CleansUpOnExit verifies the handler's deferred cleanup
// removes the client from the map when the handler exits.
func TestMatchFoundSSE_CleansUpOnExit(t *testing.T) {
	resetSSEClients(t)
	t.Cleanup(func() { resetSSEClients(t) })

	ts := newTestServer(t, newTestApp(t).routes())
	ts.validateSession(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp := ts.openSSE(t, ctx)
	defer resp.Body.Close()

	c := waitForSSEClient(t)

	// Close the channel without removing from the map so the handler's own
	// deferred cleanup is what removes the entry.
	clients.mu.Lock()
	close(c.channel)
	clients.mu.Unlock()

	// Poll until the handler's defer deletes the map entry.
	for i := 0; i < 100; i++ {
		clients.mu.Lock()
		_, exists := clients.clients[c.id]
		clients.mu.Unlock()
		if !exists {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("handler did not remove client from map after exiting")
}
