package main

import (
	"burrchess/internal/chess"
	"burrchess/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type getChessMoveData struct {
	Fen   string
	Piece int
}

type getChessMoveDataJSON struct {
	Moves            []int `json:"moves"`
	Captures         []int `json:"captures"`
	TriggerPromotion bool  `json:"triggerPromotion"`
}

type joinQueueRequest struct {
	TimeFormatInMilliseconds int64  `json:"timeFormatInMilliseconds"`
	IncrementInMilliseconds  int64  `json:"incrementInMilliseconds"`
	Action                   string `json:"action"`
}

type getHighestEloMatchResponse struct {
	MatchID int64 `json:"matchID"`
}

type authData struct {
	Username string `json:"username"`
}

type updateEmailRequest struct {
	Email string `json:"email"`
}

type updatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func generateNewPlayerId() int64 {
	return rand.Int63()
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Strict-Transport-Security", "max-age=63072000")
	app.clientError(w, http.StatusNotFound)
}

func getChessMovesHandler(w http.ResponseWriter, r *http.Request) {
	var chessMoveData getChessMoveData

	err := json.NewDecoder(r.Body).Decode(&chessMoveData)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	app.infoLog.Printf("Received body: %+v\n", chessMoveData)

	var currentGameState = chess.BoardFromFEN(chessMoveData.Fen)
	var moves, captures, triggerPromotion, _ = chess.GetValidMovesForPiece(chessMoveData.Piece, currentGameState)

	app.writeJSON(w, getChessMoveDataJSON{Moves: moves, Captures: captures, TriggerPromotion: triggerPromotion})
}

func joinQueueHandler(w http.ResponseWriter, r *http.Request) {
	var joinQueue joinQueueRequest

	app.infoLog.Printf("%v\n", r.Body)

	err := json.NewDecoder(r.Body).Decode(&joinQueue)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	app.infoLog.Printf("Received body: %+v\n", joinQueue)

	// Generate new playerID if it doesnt exist, this is for logged out players
	if !app.sessionManager.Exists(r.Context(), "playerID") && joinQueue.Action == "join" {
		var playerID = generateNewPlayerId()
		app.sessionManager.Put(r.Context(), "playerID", playerID)
	} else if !app.sessionManager.Exists(r.Context(), "playerID") && joinQueue.Action == "leave" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var playerID = app.sessionManager.GetInt64(r.Context(), "playerID")

	isInMatch, err := app.liveMatches.IsPlayerInMatch(playerID)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	if isInMatch {
		app.errorLog.Printf("Already in match, playerID: %v\n", playerID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	app.infoLog.Printf("Player ID: %v\n", playerID)

	if joinQueue.Action == "join" {
		addPlayerToWaitingPool(playerID, joinQueue.TimeFormatInMilliseconds, joinQueue.IncrementInMilliseconds)
	} else {
		removePlayerFromWaitingPool(playerID, joinQueue.TimeFormatInMilliseconds, joinQueue.IncrementInMilliseconds)
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

	// Set appropriate headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	w.Header().Set("Access-Control-Allow-Origin", app.corsOrigin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Max-Age", "10")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	cookie, _ := r.Cookie("session")
	app.infoLog.Printf("SessionID from cookie: %s\n", cookie.Value)

	ctx, err := app.sessionManager.Load(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "Failed to load session", http.StatusInternalServerError)
		return
	}

	r = r.WithContext(ctx)

	if !app.sessionManager.Exists(r.Context(), "playerID") {
		app.serverError(w, errors.New("no playerID in session"), false)
		return
	}

	var playerID = app.sessionManager.GetInt64(r.Context(), "playerID")
	app.infoLog.Printf("playerID in session: %v", playerID)

	clients.mu.Lock()
	_, ok := clients.clients[playerID]
	if !ok {
		clients.clients[playerID] = &Client{id: playerID, channel: make(chan string)}
	}
	clientChannel := clients.clients[playerID].channel
	clients.mu.Unlock()

	defer func() {
		clients.mu.Lock()
		delete(clients.clients, playerID)
		clients.mu.Unlock()
		app.infoLog.Printf("Closed SSE for playerID: %v\n", playerID)
	}()

	defer app.liveMatches.EnQueueLogAll()

	flusher, ok := w.(http.Flusher)
	if !ok {
		app.infoLog.Println("Streaming not supported")
		app.serverError(w, errors.New("streaming unsupported"), false)
		return
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case message, ok := <-clientChannel:
			if !ok {
				app.infoLog.Printf("SSE: Client Channel Closed")
				return
			}
			app.infoLog.Printf("Sending: data: %s\n\n", message)

			// Send the message to the client in SSE format
			_, err := fmt.Fprintf(w, "data: %s\n\n", message)
			if err != nil {
				app.infoLog.Printf("SSE: Client disconnected unexpectedly: %s\n", err)
				return
			}
			flusher.Flush()

		case <-heartbeat.C:
			_, err := fmt.Fprintf(w, ": heartbeat\n\n")
			if err != nil {
				app.infoLog.Printf("SSE: Client disconnected during heartbeat: %s\n", err)
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			app.infoLog.Printf("SSE: Client disconnected: %s\n", r.Context().Err())
			return
		}
	}
}

func getHighestEloMatchHandler(w http.ResponseWriter, r *http.Request) {
	matchID, err := app.liveMatches.GetHighestEloMatch()
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			app.serverError(w, err, true)
		}
		return
	}

	app.writeJSON(w, getHighestEloMatchResponse{MatchID: matchID})
}

func registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var newUser models.NewUserInfo

	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	newUserOptions := models.CreateNewUserOptions(newUser)

	var registerUserValidationErrors models.NewUserInfo

	playerID, err := app.users.InsertNew(newUser.Username, newUser.Password, &newUserOptions)
	if err != nil {
		app.errorLog.Printf("DB Error: %s\n", err.Error())
		if err.Error() == "constraint failed: UNIQUE constraint failed: users.username (2067)" {
			registerUserValidationErrors.Username = "Username already exists."
		}
		jsonStr, jsonErr := json.Marshal(registerUserValidationErrors)
		if jsonErr != nil {
			app.errorLog.Printf("Error marshalling json: %s\n", jsonErr.Error())
			app.serverError(w, err, false)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(jsonStr)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	app.sessionManager.RememberMe(r.Context(), newUser.RememberMe)
	app.sessionManager.Put(r.Context(), "username", newUser.Username)
	app.sessionManager.Put(r.Context(), "playerID", playerID)
	w.WriteHeader(http.StatusCreated)
	app.writeJSON(w, authData{Username: newUser.Username})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var loginInfo models.UserLoginInfo

	err := json.NewDecoder(r.Body).Decode(&loginInfo)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	playerID, authorized := app.users.Authenticate(loginInfo.Username, loginInfo.Password)
	if !authorized {
		w.WriteHeader(http.StatusUnauthorized)
		var loginValidationErrors models.UserLoginInfo
		loginValidationErrors.Username = "Username or password invalid."
		jsonStr, jsonErr := json.Marshal(loginValidationErrors)
		if jsonErr == nil {
			w.Write(jsonStr)
		} else {
			app.errorLog.Println(jsonErr)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	app.sessionManager.RememberMe(r.Context(), loginInfo.RememberMe)
	app.sessionManager.Put(r.Context(), "username", loginInfo.Username)
	app.sessionManager.Put(r.Context(), "playerID", playerID)
	app.writeJSON(w, authData{Username: loginInfo.Username})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if !app.sessionManager.Exists(r.Context(), "username") {
		app.errorLog.Printf("Not logged in\n")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	app.sessionManager.Destroy(r.Context())
	w.WriteHeader(http.StatusOK)
}

func validateSessionHandler(w http.ResponseWriter, r *http.Request) {
	if !app.sessionManager.Exists(r.Context(), "username") {
		if !app.sessionManager.Exists(r.Context(), "playerID") {
			app.sessionManager.Put(r.Context(), "playerID", generateNewPlayerId())
		}
		w.WriteHeader(http.StatusUnauthorized)
	}

	app.writeJSON(w, authData{
		Username: app.sessionManager.GetString(r.Context(), "username"),
	})
}

func userSearchHandler(w http.ResponseWriter, r *http.Request) {
	searchString := r.URL.Query().Get("search")

	if searchString == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userList, err := app.users.SearchForUsers(searchString)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	app.writeJSON(w, userList)
}

func getTileInfoHandler(w http.ResponseWriter, r *http.Request) {
	searchString := r.URL.Query().Get("search")

	if searchString == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tileInfo, err := app.users.GetTileInfoFromUsername(searchString)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	app.writeJSON(w, tileInfo)
}

func getPastMatchesListHandler(w http.ResponseWriter, r *http.Request) {
	filters := models.PastMatchFilters{}

	queryParams := r.URL.Query()

	searchString := queryParams.Get("timeFormat")
	username := queryParams.Get("username")

	if username != "" {
		filters.Username = &username
	}

	app.infoLog.Printf("searchString: %s\n", searchString)

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
		app.serverError(w, err, false)
		return
	}
	app.infoLog.Printf("%v\n", matchList)

	app.writeJSON(w, matchList)
}

func getUserAccountSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if !app.sessionManager.Exists(r.Context(), "playerID") {
		app.clientError(w, http.StatusUnauthorized)
		return
	}
	playerID := app.sessionManager.GetInt64(r.Context(), "playerID")

	accountSettings, err := app.users.GetUserAccountSettings(playerID)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	app.writeJSON(w, accountSettings)
}

func updateEmailHandler(w http.ResponseWriter, r *http.Request) {
	if !app.sessionManager.Exists(r.Context(), "playerID") {
		app.clientError(w, http.StatusUnauthorized)
		return
	}
	playerID := app.sessionManager.GetInt64(r.Context(), "playerID")

	var updateEmailData updateEmailRequest

	app.infoLog.Printf("%v\n", r.Body)

	err := json.NewDecoder(r.Body).Decode(&updateEmailData)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	err = app.users.UpdateEmail(playerID, updateEmailData.Email)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func updatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	if !app.sessionManager.Exists(r.Context(), "playerID") || !app.sessionManager.Exists(r.Context(), "username") {
		app.clientError(w, http.StatusUnauthorized)
		return
	}
	playerID := app.sessionManager.GetInt64(r.Context(), "playerID")
	username := app.sessionManager.GetString(r.Context(), "username")

	var updatePasswordData updatePasswordRequest

	app.infoLog.Printf("%v\n", r.Body)

	err := json.NewDecoder(r.Body).Decode(&updatePasswordData)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	_, authorized := app.users.Authenticate(username, updatePasswordData.CurrentPassword)
	if !authorized {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	err = app.users.UpdatePassword(playerID, updatePasswordData.NewPassword)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	w.WriteHeader(http.StatusOK)
}
