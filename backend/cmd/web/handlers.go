package main

import (
	"burrchess/internal/chess"
	"burrchess/internal/models"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

var sseHeartbeatTimer = 15 * time.Second

type joinQueueRequest struct {
	TimeFormatInMilliseconds int64  `json:"timeFormatInMilliseconds"`
	IncrementInMilliseconds  int64  `json:"incrementInMilliseconds"`
	Action                   string `json:"action"`
}

type getHighestEloMatchResponse struct {
	MatchID int64 `json:"matchID"`
}

type authData struct {
	Username  string `json:"username"`
	CsrfToken string `json:"csrfToken"`
}

type updateEmailRequest struct {
	Email string `json:"email"`
}

type registerValidationErrors struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
}

type loginValidationErrors struct {
	Username string `json:"username,omitempty"`
}

type updateEmailValidationErrors struct {
	Email string `json:"email,omitempty"`
}

type updatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type updatePasswordValidationErrors struct {
	CurrentPassword string `json:"currentPassword,omitempty"`
	NewPassword     string `json:"newPassword,omitempty"`
}

func (app *application) rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Strict-Transport-Security", "max-age=63072000")
	app.clientError(w, http.StatusNotFound)
}

func (app *application) joinQueueHandler(w http.ResponseWriter, r *http.Request) {
	var joinQueue joinQueueRequest

	err := json.NewDecoder(r.Body).Decode(&joinQueue)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Generate new playerID if it doesnt exist, this is for logged out players
	if !app.sessionManager.Exists(r.Context(), "playerID") && joinQueue.Action == "join" {
		var playerID = models.GenerateNewPlayerId()
		app.sessionManager.Put(r.Context(), "playerID", playerID)
	} else if !app.sessionManager.Exists(r.Context(), "playerID") && joinQueue.Action == "leave" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var playerID = app.sessionManager.GetInt64(r.Context(), "playerID")

	if joinQueue.Action == "join" {
		isInMatch, err := app.liveMatches.IsPlayerInMatch(playerID)
		if err != nil {
			app.serverError(w, err)
			return
		}
		if isInMatch {
			app.logger.Warn("join queue rejected: player already in match", "playerID", playerID)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	if joinQueue.Action == "join" {
		addPlayerToWaitingPool(playerID, joinQueue.TimeFormatInMilliseconds, joinQueue.IncrementInMilliseconds)
		app.logger.Info("player joined queue", "playerID", playerID)
	} else {
		removePlayerFromWaitingPool(playerID, joinQueue.TimeFormatInMilliseconds, joinQueue.IncrementInMilliseconds)
		app.logger.Info("player left queue", "playerID", playerID)
	}
}

type Client struct {
	id      int64
	channel chan string
}

type Clients struct {
	mu      sync.Mutex
	clients map[int64]*Client
}

var clients = Clients{
	clients: make(map[int64]*Client),
}

func (app *application) matchFoundSSEHandler(w http.ResponseWriter, r *http.Request) {

	// Disable the server's WriteTimeout for this long-lived SSE connection.
	// The heartbeat ticker detects dead clients instead.
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		app.serverError(w, err)
		return
	}

	// Set appropriate headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	playerID, ok := app.sessionPlayerID(w, r)
	if !ok {
		return
	}

	clients.mu.Lock()
	client := &Client{id: playerID, channel: make(chan string, 1)}
	if existing, exists := clients.clients[playerID]; exists {
		// Forward any buffered notification so we don't lose a match-found event
		// that arrived between joinQueue returning and this SSE connection opening.
		select {
		case msg := <-existing.channel:
			client.channel <- msg
		default:
		}
		close(existing.channel)
	}
	clients.clients[playerID] = client
	clientChannel := client.channel
	clients.mu.Unlock()

	defer func() {
		clients.mu.Lock()
		if clients.clients[playerID] == client {
			delete(clients.clients, playerID)
		}
		clients.mu.Unlock()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		app.logger.Error("match found sse handler: streaming not supported")
		app.serverError(w, errors.New("streaming unsupported"))
		return
	}

	// Send an initial SSE comment to flush headers through any proxy buffer and
	// trigger the client's EventSource onopen event immediately.
	// Without body content, some proxies (including Vite's dev proxy) buffer
	// the headers and delay onopen until real data arrives.
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(sseHeartbeatTimer)
	defer heartbeat.Stop()

	for {
		select {
		case message, ok := <-clientChannel:
			if !ok {
				return
			}

			// Send the message to the client in SSE format
			_, err := fmt.Fprintf(w, "data: %s\n\n", message)
			if err != nil {
				app.logger.Warn("client disconnected from sse unexpectedly", "playerID", playerID, "err", err)
				return
			}
			flusher.Flush()

		case <-heartbeat.C:
			_, err := fmt.Fprintf(w, ": heartbeat\n\n")
			if err != nil {
				app.logger.Warn("client disconnected from sse during heartbeat", "playerID", playerID, "err", err)
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

func (app *application) getHighestEloMatchHandler(w http.ResponseWriter, r *http.Request) {
	matchID, err := app.liveMatches.GetHighestEloMatch()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.writeJSON(w, getHighestEloMatchResponse{MatchID: matchID})
}

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var newUser models.NewUserInfo

	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	newUserOptions := models.CreateNewUserOptions(newUser)

	var registerUserValidationErrors registerValidationErrors
	hasErrors := false

	if utf8.RuneCountInString(newUser.Username) < models.MinUsernameLength || utf8.RuneCountInString(newUser.Username) > models.MaxUsernameLength {
		registerUserValidationErrors.Username = fmt.Sprintf("Username must be between %d and %d characters.", models.MinUsernameLength, models.MaxUsernameLength)
		hasErrors = true
	}

	// Use byte count as bcrypt truncates at 72 bytes
	if len(newUser.Password) < models.MinPasswordLength || len(newUser.Password) > models.MaxPasswordLength {
		registerUserValidationErrors.Password = fmt.Sprintf("Password must be between %d and %d characters.", models.MinPasswordLength, models.MaxPasswordLength)
		hasErrors = true
	}

	if !models.IsValidEmail(newUser.Email) {
		registerUserValidationErrors.Email = "invalid email address"
		hasErrors = true
	}

	playerID, err := app.users.InsertNew(newUser.Username, newUser.Password, &newUserOptions)
	if err != nil {

		hasErrors = true

		if strings.Contains(err.Error(), models.SqliteUniqueErrSubstr) {
			registerUserValidationErrors.Username = "Username already exists."
		} else {
			app.logger.Error("registration failed", "err", err)
		}

	}

	if hasErrors {
		w.WriteHeader(http.StatusBadRequest)
		app.writeJSON(w, registerUserValidationErrors)
		return
	}

	app.logger.Info("user registered", "playerID", playerID, "username", newUser.Username)

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.logger.Error("failed to renew session token", "playerID", playerID)
		app.serverError(w, err)
		return
	}

	csrfToken, err := generateCSRFToken()
	if err != nil {
		app.logger.Error("failed to generate CSRF token during registration", "err", err)
		app.serverError(w, err)
		return
	}
	app.sessionManager.Put(r.Context(), "csrfToken", csrfToken)
	app.sessionManager.RememberMe(r.Context(), newUser.RememberMe)
	app.sessionManager.Put(r.Context(), "username", newUser.Username)
	app.sessionManager.Put(r.Context(), "playerID", playerID)
	w.WriteHeader(http.StatusCreated)
	app.writeJSON(w, authData{Username: newUser.Username, CsrfToken: csrfToken})
}

func (app *application) loginHandler(w http.ResponseWriter, r *http.Request) {
	var loginInfo models.UserLoginInfo

	err := json.NewDecoder(r.Body).Decode(&loginInfo)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	playerID, authorized := app.users.Authenticate(loginInfo.Username, loginInfo.Password)
	if !authorized {
		app.logger.Info("login failed", "username", loginInfo.Username)
		w.WriteHeader(http.StatusUnauthorized)
		app.writeJSON(w, loginValidationErrors{Username: "Username or password invalid."})
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.logger.Error("failed to renew session token", "playerID", playerID)
		app.serverError(w, err)
		return
	}

	csrfToken, err := generateCSRFToken()
	if err != nil {
		app.logger.Error("failed to generate CSRF token during login", "err", err)
		app.serverError(w, err)
		return
	}
	app.sessionManager.Put(r.Context(), "csrfToken", csrfToken)
	app.sessionManager.RememberMe(r.Context(), loginInfo.RememberMe)
	app.sessionManager.Put(r.Context(), "username", loginInfo.Username)
	app.sessionManager.Put(r.Context(), "playerID", playerID)
	app.logger.Info("login succeeded", "playerID", playerID, "username", loginInfo.Username)
	app.writeJSON(w, authData{Username: loginInfo.Username, CsrfToken: csrfToken})
}

func (app *application) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if !app.sessionManager.Exists(r.Context(), "username") {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.logger.Error("failed to renew session token on logout", "err", err)
		app.serverError(w, err)
		return
	}

	app.sessionManager.Destroy(r.Context())
	w.WriteHeader(http.StatusOK)
}

func (app *application) validateSessionHandler(w http.ResponseWriter, r *http.Request) {
	if !app.sessionManager.Exists(r.Context(), "csrfToken") {
		csrfToken, err := generateCSRFToken()
		if err != nil {
			app.logger.Error("failed to generate CSRF token during session validation", "err", err)
			app.serverError(w, err)
			return
		}
		app.sessionManager.Put(r.Context(), "csrfToken", csrfToken)
	}
	csrfToken := app.sessionManager.GetString(r.Context(), "csrfToken")

	if !app.sessionManager.Exists(r.Context(), "username") {
		if !app.sessionManager.Exists(r.Context(), "playerID") {
			app.sessionManager.Put(r.Context(), "playerID", models.GenerateNewPlayerId())
		}
		// Return the CSRF token even for logged-out users so they can make
		// CSRF-protected requests (e.g. joinQueue) without logging in first.
		jsonStr, err := json.Marshal(authData{CsrfToken: csrfToken})
		if err != nil {
			app.serverError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)

		_, err = w.Write(jsonStr)
		if err != nil {
			app.serverError(w, err)
			return
		}

		return
	}

	app.writeJSON(w, authData{
		Username:  app.sessionManager.GetString(r.Context(), "username"),
		CsrfToken: csrfToken,
	})
}

func (app *application) userSearchHandler(w http.ResponseWriter, r *http.Request) {
	searchString := r.URL.Query().Get("search")

	if searchString == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if utf8.RuneCountInString(searchString) > models.MaxUsernameLength {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userList, err := app.users.SearchForUsers(searchString)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.writeJSON(w, userList)
}

func (app *application) getTileInfoHandler(w http.ResponseWriter, r *http.Request) {
	searchString := r.URL.Query().Get("search")

	if searchString == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tileInfo, err := app.users.GetTileInfoFromUsername(searchString)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.writeJSON(w, tileInfo)
}

func (app *application) getPastMatchesListHandler(w http.ResponseWriter, r *http.Request) {
	filters := models.PastMatchFilters{}

	queryParams := r.URL.Query()

	searchString := queryParams.Get("timeFormat")
	username := queryParams.Get("username")

	if username != "" {
		filters.Username = &username
	}

	switch searchString {
	case "bullet":
		filters.TimeFormatLower, filters.TimeFormatUpper = &chess.Bullet[0], &chess.Bullet[1]
	case "blitz":
		filters.TimeFormatLower, filters.TimeFormatUpper = &chess.Blitz[0], &chess.Blitz[1]
	case "rapid":
		filters.TimeFormatLower, filters.TimeFormatUpper = &chess.Rapid[0], &chess.Rapid[1]
	case "classical":
		filters.TimeFormatLower, filters.TimeFormatUpper = &chess.Classical[0], &chess.Classical[1]
	}

	matchList, err := app.pastMatches.GetPastMatchesWithFormat(filters)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.writeJSON(w, matchList)
}

func (app *application) getUserAccountSettingsHandler(w http.ResponseWriter, r *http.Request) {
	playerID, ok := app.sessionPlayerID(w, r)
	if !ok {
		return
	}

	accountSettings, err := app.users.GetUserAccountSettings(playerID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.writeJSON(w, accountSettings)
}

func (app *application) updateEmailHandler(w http.ResponseWriter, r *http.Request) {
	playerID, _, ok := app.sessionPlayer(w, r)
	if !ok {
		return
	}

	var updateEmailData updateEmailRequest
	var updateEmailValidationErrors updateEmailValidationErrors
	hasErrors := false

	err := json.NewDecoder(r.Body).Decode(&updateEmailData)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if !models.IsValidEmail(updateEmailData.Email) {
		updateEmailValidationErrors.Email = "invalid email address"
		hasErrors = true
	}

	if hasErrors {
		w.WriteHeader(http.StatusBadRequest)
		app.writeJSON(w, updateEmailValidationErrors)
		return
	}

	err = app.users.UpdateEmail(playerID, updateEmailData.Email)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (app *application) updatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	playerID, username, ok := app.sessionPlayer(w, r)
	if !ok {
		return
	}

	var updatePasswordData updatePasswordRequest

	err := json.NewDecoder(r.Body).Decode(&updatePasswordData)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	_, authorized := app.users.Authenticate(username, updatePasswordData.CurrentPassword)
	if !authorized {
		w.WriteHeader(http.StatusBadRequest)
		app.writeJSON(w, updatePasswordValidationErrors{CurrentPassword: "Current password is incorrect."})
		return
	}

	if len(updatePasswordData.NewPassword) < models.MinPasswordLength || len(updatePasswordData.NewPassword) > models.MaxPasswordLength {
		w.WriteHeader(http.StatusBadRequest)
		app.writeJSON(w, updatePasswordValidationErrors{NewPassword: fmt.Sprintf("Password must be between %d and %d characters.", models.MinPasswordLength, models.MaxPasswordLength)})
		return
	}

	err = app.users.UpdatePassword(playerID, updatePasswordData.NewPassword)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
