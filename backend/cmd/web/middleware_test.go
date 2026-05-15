package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// CSRF / requireSameOrigin
// ---------------------------------------------------------------------------

func TestCSRF_NoSessionNoCsrfToken(t *testing.T) {
	// Without any session, a POST to a protected endpoint returns 403.
	ts := newTestServer(t, newTestApp(t).routes())

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/joinQueue", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	// No X-CSRF-Token header, no session cookie

	resp, err := ts.client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403 (missing CSRF token)", resp.StatusCode)
	}
}

func TestCSRF_WrongToken(t *testing.T) {
	// With a valid session but a wrong CSRF token, a POST returns 403.
	ts := newTestServer(t, newTestApp(t).routes())
	ts.validateSession(t) // establishes session with a real CSRF token

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/joinQueue", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "definitely-wrong-token")

	resp, err := ts.client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403 (wrong CSRF token)", resp.StatusCode)
	}
}

func TestCSRF_ValidToken(t *testing.T) {
	// With a valid session and correct CSRF token, the request reaches the handler.
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.validateSession(t)

	// joinQueue with a valid token — the handler itself may return 400 for bad
	// JSON, but it must NOT be rejected by the CSRF middleware (status != 403).
	resp := ts.postJSON(t, "/joinQueue", csrf, struct {
		TimeFormatInMilliseconds int64  `json:"timeFormatInMilliseconds"`
		IncrementInMilliseconds  int64  `json:"incrementInMilliseconds"`
		Action                   string `json:"action"`
	}{5 * 60_000, 0, "join"})
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		t.Error("status = 403, CSRF token should have been accepted")
	}
}

func TestCSRF_ValidateSessionSkipsCheck(t *testing.T) {
	// POST /validateSession must succeed even with no CSRF token because it is
	// the endpoint that issues the token in the first place.
	ts := newTestServer(t, newTestApp(t).routes())

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/validateSession", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	// Deliberately no X-CSRF-Token and no session cookie.

	resp, err := ts.client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		t.Error("/validateSession must not require a CSRF token")
	}
}

func TestCSRF_WrongOriginRejected(t *testing.T) {
	// A POST with an Origin that does not match allowedOrigin is rejected.
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.validateSession(t)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/joinQueue", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Origin", "http://evil.example.com") // wrong origin

	resp, err := ts.client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403 for wrong Origin header", resp.StatusCode)
	}
}

func TestCSRF_GetRequestSkipsCheck(t *testing.T) {
	// GET requests are never subject to CSRF checks.
	ts := newTestServer(t, newTestApp(t).routes())

	resp := ts.get(t, "/userSearch?search=x")
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		t.Error("GET request must not be blocked by CSRF middleware")
	}
}

// ---------------------------------------------------------------------------
// secureHeaders
// ---------------------------------------------------------------------------

func TestSecureHeaders(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())

	resp := ts.get(t, "/userSearch?search=x")
	defer resp.Body.Close()

	headers := map[string]string{
		"X-Frame-Options":        "deny",
		"X-Content-Type-Options": "nosniff",
	}
	for header, want := range headers {
		if got := resp.Header.Get(header); got != want {
			t.Errorf("header %s = %q, want %q", header, got, want)
		}
	}
	if resp.Header.Get("Strict-Transport-Security") == "" {
		t.Error("Strict-Transport-Security header missing")
	}
}

// ---------------------------------------------------------------------------
// requireLocalhost
// ---------------------------------------------------------------------------

func TestRequireLocalhost_RemoteAddressRejected(t *testing.T) {
	// The health endpoint is restricted to localhost; a remote address must
	// receive 403. We test this directly against the handler wrapped in the
	// middleware, using a fake remote address.
	app := newTestApp(t)

	handler := requireLocalhost(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Simulate a non-localhost remote address
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "203.0.113.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 for non-localhost address", rr.Code)
	}

	_ = app // ensure app compiles
}

func TestRequireLocalhost_LocalhostIPv4Allowed(t *testing.T) {
	handler := requireLocalhost(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for 127.0.0.1", rr.Code)
	}
}

func TestRequireLocalhost_LocalhostIPv6Allowed(t *testing.T) {
	handler := requireLocalhost(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "[::1]:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for ::1", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// rateLimit — 429 after burst exhausted
// ---------------------------------------------------------------------------

func TestRateLimit_Returns429AfterBurst(t *testing.T) {
	app := newTestApp(t)

	// Wrap a trivial handler with the HTTP rate limiter.
	handler := app.rateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// The limiter allows a burst of 20 (see middleware.go).  Send 21 requests
	// from the same RemoteAddr — the last one must be rejected with 429.
	const burst = 20
	var lastStatus int
	for i := 0; i <= burst; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		lastStatus = rr.Code
	}

	if lastStatus != http.StatusTooManyRequests {
		t.Errorf("status after burst = %d, want 429", lastStatus)
	}
}

func TestAuthRateLimit_Returns429AfterBurst(t *testing.T) {
	app := newTestApp(t)

	handler := app.authRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Auth limiter burst is 5; the 6th request must be rejected.
	const burst = 5
	var lastStatus int
	for i := 0; i <= burst; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		lastStatus = rr.Code
	}

	if lastStatus != http.StatusTooManyRequests {
		t.Errorf("status after auth burst = %d, want 429", lastStatus)
	}
}

// ---------------------------------------------------------------------------
// recoverPanic
// ---------------------------------------------------------------------------

func TestRecoverPanic(t *testing.T) {
	app := newTestApp(t)

	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := app.recoverPanic(panicking)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 after panic", rr.Code)
	}
}
