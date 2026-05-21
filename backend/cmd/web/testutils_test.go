package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"burrchess/internal/models"

	"github.com/alexedwards/scs/v2"
	"golang.org/x/crypto/bcrypt"
)


// openTestDB opens an in-memory SQLite database with the schema applied.
// Working directory for tests in cmd/web/ is the package directory itself.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("openTestDB: open: %v", err)
	}
	schema, err := os.ReadFile("../../internal/models/schema.sql")
	if err != nil {
		t.Fatalf("openTestDB: read schema.sql: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("openTestDB: exec schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// newTestApp creates a fully wired application backed by an in-memory SQLite
// database and an in-memory session store. Logs are discarded.
func newTestApp(t *testing.T) *application {
	t.Helper()
	db := openTestDB(t)

	sm := scs.New() // default: in-memory session store
	sm.Cookie.Secure = false // allow plain HTTP in tests

	return &application{
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		secretKey:      []byte("aaaabbbbccccddddeeeeffffgggghhhh"),
		liveMatches:    &models.LiveMatchModel{DB: db},
		pastMatches:    &models.PastMatchModel{DB: db},
		users:          &models.UserModel{DB: db, BcryptCost: bcrypt.MinCost},
		userRatings:    &models.UserRatingsModel{DB: db},
		dbTaskQueue:    models.DBTaskQueue,
		sessionManager: sm,
		allowedOrigin:  "http://localhost",
	}
}

// testServer wraps httptest.Server with a cookie-aware HTTP client.
type testServer struct {
	*httptest.Server
	client *http.Client
}

// newTestServer starts a test HTTP server and returns a testServer whose
// client automatically stores and sends cookies across requests.
func newTestServer(t *testing.T, h http.Handler) *testServer {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	jar, _ := cookiejar.New(nil)
	return &testServer{
		Server: ts,
		client: &http.Client{
			Jar: jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// get sends a GET request and returns the response. Caller must close the body.
func (ts *testServer) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := ts.client.Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

// postJSON sends a POST request with a JSON-marshalled body and an optional
// CSRF token header. Caller must close the response body.
func (ts *testServer) postJSON(t *testing.T, path, csrfToken string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("postJSON marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(string(b)))
	if err != nil {
		t.Fatalf("postJSON new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if csrfToken != "" {
		req.Header.Set("X-CSRF-Token", csrfToken)
	}
	resp, err := ts.client.Do(req)
	if err != nil {
		t.Fatalf("postJSON do %s: %v", path, err)
	}
	return resp
}

// validateSession calls POST /validateSession (CSRF-exempt) to initialise a
// session and returns the CSRF token for use in subsequent requests.
func (ts *testServer) validateSession(t *testing.T) string {
	t.Helper()
	resp := ts.postJSON(t, "/validateSession", "", struct{}{})
	defer resp.Body.Close()

	var data authData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("validateSession decode: %v", err)
	}
	if data.CsrfToken == "" {
		t.Fatal("validateSession: empty CSRF token in response")
	}
	return data.CsrfToken
}

// registerAndLogin registers a fresh user, then logs in (renews token), and
// returns the CSRF token from the login response. Suitable for tests that
// need a fully authenticated session.
//
// Note: each call invokes bcrypt twice (~1.5s total at cost 14).
func (ts *testServer) registerAndLogin(t *testing.T, username, password string) string {
	t.Helper()

	csrf := ts.validateSession(t)

	// Register
	resp := ts.postJSON(t, "/register", csrf, models.NewUserInfo{
		Username: username,
		Password: password,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("register %q: got %d, body: %s", username, resp.StatusCode, body)
	}
	var regData authData
	json.NewDecoder(resp.Body).Decode(&regData)
	return regData.CsrfToken
}
