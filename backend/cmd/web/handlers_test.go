package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"burrchess/internal/models"
)

// ---------------------------------------------------------------------------
// validateSession
// ---------------------------------------------------------------------------

func TestValidateSession_NotLoggedIn(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())

	resp := ts.postJSON(t, "/validateSession", "", struct{}{})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}

	var data authData
	json.NewDecoder(resp.Body).Decode(&data)
	if data.CsrfToken == "" {
		t.Error("expected a CSRF token even for unauthenticated users")
	}
	if data.Username != "" {
		t.Errorf("expected empty username, got %q", data.Username)
	}
}

func TestValidateSession_LoggedIn(t *testing.T) {
	app := newTestApp(t)
	ts := newTestServer(t, app.routes())

	csrf := ts.registerAndLogin(t, "alice", "password123")

	resp := ts.postJSON(t, "/validateSession", csrf, struct{}{})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var data authData
	json.NewDecoder(resp.Body).Decode(&data)
	if data.Username != "alice" {
		t.Errorf("username = %q, want alice", data.Username)
	}
	if data.CsrfToken == "" {
		t.Error("expected non-empty CSRF token")
	}
}

// ---------------------------------------------------------------------------
// register
// ---------------------------------------------------------------------------

func TestRegister_Valid(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.validateSession(t)

	resp := ts.postJSON(t, "/register", csrf, models.NewUserInfo{
		Username: "bob",
		Password: "password123",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 201; body: %s", resp.StatusCode, body)
	}

	var data authData
	json.NewDecoder(resp.Body).Decode(&data)
	if data.Username != "bob" {
		t.Errorf("username = %q, want bob", data.Username)
	}
	if data.CsrfToken == "" {
		t.Error("expected non-empty CSRF token after registration")
	}
}

func TestRegister_UsernameTooShort(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.validateSession(t)

	resp := ts.postJSON(t, "/register", csrf, models.NewUserInfo{
		Username: "ab", // 2 chars, below minimum of 3
		Password: "password123",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	var errs registerValidationErrors
	json.NewDecoder(resp.Body).Decode(&errs)
	if errs.Username == "" {
		t.Error("expected Username validation error, got empty string")
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())

	csrf := ts.registerAndLogin(t, "carol", "password123")

	// Re-initialise session for a "second user" attempt on the same server
	csrf2 := ts.validateSession(t)
	resp := ts.postJSON(t, "/register", csrf2, models.NewUserInfo{
		Username: "carol",
		Password: "different",
	})
	defer resp.Body.Close()
	_ = csrf

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	var errs registerValidationErrors
	json.NewDecoder(resp.Body).Decode(&errs)
	if errs.Username == "" {
		t.Error("expected Username error for duplicate, got empty")
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.validateSession(t)

	resp := ts.postJSON(t, "/register", csrf, models.NewUserInfo{
		Username: "dave",
		Password: "password123",
		Email:    "not-an-email",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	var errs registerValidationErrors
	json.NewDecoder(resp.Body).Decode(&errs)
	if errs.Email == "" {
		t.Error("expected Email validation error")
	}
}

// ---------------------------------------------------------------------------
// login
// ---------------------------------------------------------------------------

func TestLogin_Valid(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	ts.registerAndLogin(t, "eve", "password123")

	// Now test explicit login on a fresh session
	csrf2 := ts.validateSession(t)
	resp := ts.postJSON(t, "/login", csrf2, models.UserLoginInfo{
		Username: "eve",
		Password: "password123",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var data authData
	json.NewDecoder(resp.Body).Decode(&data)
	if data.Username != "eve" {
		t.Errorf("username = %q, want eve", data.Username)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	ts.registerAndLogin(t, "frank", "correctpass")

	csrf := ts.validateSession(t)
	resp := ts.postJSON(t, "/login", csrf, models.UserLoginInfo{
		Username: "frank",
		Password: "wrongpass",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
	var errs loginValidationErrors
	json.NewDecoder(resp.Body).Decode(&errs)
	if errs.Username == "" {
		t.Error("expected Username error for wrong password")
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.validateSession(t)

	resp := ts.postJSON(t, "/login", csrf, models.UserLoginInfo{
		Username: "nobody",
		Password: "password",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// logout
// ---------------------------------------------------------------------------

func TestLogout_Valid(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.registerAndLogin(t, "grace", "password123")

	resp := ts.postJSON(t, "/logout", csrf, struct{}{})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	// After logout, validateSession should return 401 (no username in session)
	resp2 := ts.postJSON(t, "/validateSession", "", struct{}{})
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("after logout, validateSession status = %d, want 401", resp2.StatusCode)
	}
}

func TestLogout_NotLoggedIn(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.validateSession(t)

	resp := ts.postJSON(t, "/logout", csrf, struct{}{})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 when logging out without a session", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// account settings
// ---------------------------------------------------------------------------

func TestGetAccountSettings_Unauthenticated(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())

	resp := ts.get(t, "/getAccountSettings")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestGetAccountSettings_Authenticated(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	ts.registerAndLogin(t, "henry", "password123")

	resp := ts.get(t, "/getAccountSettings")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var settings models.AccountSettings
	json.NewDecoder(resp.Body).Decode(&settings)
	// Newly registered user has no email set
	if settings.Email != nil {
		t.Errorf("expected nil email for new user, got %q", *settings.Email)
	}
}

// ---------------------------------------------------------------------------
// email change
// ---------------------------------------------------------------------------

func TestUpdateEmail_Unauthenticated(t *testing.T) {
	// updateEmailHandler only requires playerID in session, which validateSession
	// assigns even to anonymous users. Test with no session at all (no cookies)
	// to get a genuine 401.
	ts := newTestServer(t, newTestApp(t).routes())

	// Use a fresh client with no cookies — no session, no playerID.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/emailChange", strings.NewReader(`{"email":"x@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	// No X-CSRF-Token -> will be caught by CSRF middleware first (403), unless
	// we send it. Since there is no session, the session CSRF token is "" and
	// the condition `sessionToken == "" || token != sessionToken` triggers 403.
	// Either 401 or 403 indicates the request was rejected before doing work.
	resp, err := ts.client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 or 403 for unauthenticated email change", resp.StatusCode)
	}
}

func TestUpdateEmail_Valid(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.registerAndLogin(t, "iris", "password123")

	resp := ts.postJSON(t, "/emailChange", csrf, updateEmailRequest{Email: "iris@example.com"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body: %s", resp.StatusCode, body)
	}
}

func TestUpdateEmail_InvalidEmail(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.registerAndLogin(t, "jack", "password123")

	resp := ts.postJSON(t, "/emailChange", csrf, updateEmailRequest{Email: "not-an-email"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	var errs updateEmailValidationErrors
	json.NewDecoder(resp.Body).Decode(&errs)
	if errs.Email == "" {
		t.Error("expected Email validation error")
	}
}

// ---------------------------------------------------------------------------
// password change
// ---------------------------------------------------------------------------

func TestUpdatePassword_Unauthenticated(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.validateSession(t)

	resp := ts.postJSON(t, "/passwordChange", csrf, updatePasswordRequest{
		CurrentPassword: "old",
		NewPassword:     "new",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestUpdatePassword_WrongCurrentPassword(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.registerAndLogin(t, "kate", "correctpass")

	resp := ts.postJSON(t, "/passwordChange", csrf, updatePasswordRequest{
		CurrentPassword: "wrongpass",
		NewPassword:     "newpass123",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	var errs updatePasswordValidationErrors
	json.NewDecoder(resp.Body).Decode(&errs)
	if errs.CurrentPassword == "" {
		t.Error("expected CurrentPassword error for wrong password")
	}
}

func TestUpdatePassword_Valid(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())
	csrf := ts.registerAndLogin(t, "leo", "oldpass123")

	resp := ts.postJSON(t, "/passwordChange", csrf, updatePasswordRequest{
		CurrentPassword: "oldpass123",
		NewPassword:     "newpass456",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body: %s", resp.StatusCode, body)
	}

	// Old password should no longer work
	csrf2 := ts.validateSession(t)
	resp2 := ts.postJSON(t, "/login", csrf2, models.UserLoginInfo{
		Username: "leo",
		Password: "oldpass123",
	})
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("old password still works after change; login status = %d", resp2.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// user search
// ---------------------------------------------------------------------------

func TestUserSearch_EmptyQuery(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())

	resp := ts.get(t, "/userSearch")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestUserSearch_QueryTooLong(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())

	// 33 chars — one over the MaxUsernameLength of 32
	resp := ts.get(t, "/userSearch?search=aaaaabbbbbcccccdddddeeeeefffff123")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestUserSearch_ValidQuery(t *testing.T) {
	app := newTestApp(t)
	ts := newTestServer(t, app.routes())

	// Seed a user directly via the model
	_, err := app.users.InsertNew("searchme", "pass", &models.NewUserOptions{})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	resp := ts.get(t, "/userSearch?search=search")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var results []models.UserClientSide
	json.NewDecoder(resp.Body).Decode(&results)
	if len(results) != 1 || results[0].Username != "searchme" {
		t.Errorf("got results %+v, want [{Username:searchme}]", results)
	}
}

// ---------------------------------------------------------------------------
// tile info
// ---------------------------------------------------------------------------

func TestGetTileInfo_EmptyQuery(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())

	resp := ts.get(t, "/getTileInfo")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestGetTileInfo_KnownUser(t *testing.T) {
	app := newTestApp(t)
	ts := newTestServer(t, app.routes())

	_, err := app.users.InsertNew("mia", "pass", &models.NewUserOptions{})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	resp := ts.get(t, "/getTileInfo?search=mia")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var info models.UserTileInfo
	json.NewDecoder(resp.Body).Decode(&info)
	if info.Username != "mia" {
		t.Errorf("username = %q, want mia", info.Username)
	}
}

// ---------------------------------------------------------------------------
// getHighestEloMatch
// ---------------------------------------------------------------------------

func TestGetHighestEloMatch_NoMatches(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())

	resp := ts.get(t, "/getHighestEloMatch")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want 204 when no matches exist", resp.StatusCode)
	}
}

func TestGetHighestEloMatch_WithMatches(t *testing.T) {
	app := newTestApp(t)
	ts := newTestServer(t, app.routes())

	matchID, err := app.liveMatches.InsertNew(models.InsertNewParams{
		PlayerOneID:              100,
		PlayerTwoID:              200,
		PlayerOneIsWhite:         true,
		TimeFormatInMilliseconds: 5 * 60_000,
		GameHistory:              []byte("[]"),
		AverageElo:               1500,
		WhitePlayerElo:           1500,
		BlackPlayerElo:           1500,
	})
	if err != nil {
		t.Fatalf("seed match: %v", err)
	}

	resp := ts.get(t, "/getHighestEloMatch")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var result getHighestEloMatchResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.MatchID != matchID {
		t.Errorf("matchID = %d, want %d", result.MatchID, matchID)
	}
}

// ---------------------------------------------------------------------------
// getPastMatches
// ---------------------------------------------------------------------------

func TestGetPastMatches_NoFilter(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())

	resp := ts.get(t, "/getPastMatches")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	// GetPastMatchesWithFormat returns nil (not []) when there are no rows,
	// so the JSON response is "null" — that is valid; just confirm 200 OK.
	_ = models.PastMatchSummary{} // keep import used
}

func TestGetPastMatches_TimeFormatFilter(t *testing.T) {
	ts := newTestServer(t, newTestApp(t).routes())

	resp := ts.get(t, "/getPastMatches?timeFormat=blitz")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
