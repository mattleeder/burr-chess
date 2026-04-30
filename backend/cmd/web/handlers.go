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

var (
	ErrValueTooLong = errors.New("cookie value too long")
	ErrInvalidValue = errors.New("invalid cookie value")
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

type userSearchData struct {
	SearchString string `json:"searchString"`
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
	start := time.Now()
	defer func() { app.perfLog.Printf("getChessMovesHandler took: %s\n", time.Since(start)) }()

	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	w.Header().Set("Content-Type", "application/json")

	var chessMoveData getChessMoveData

	err := json.NewDecoder(r.Body).Decode(&chessMoveData)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	app.infoLog.Printf("Received body: %+v\n", chessMoveData)

	var currentGameState = chess.BoardFromFEN(chessMoveData.Fen)
	var moves, captures, triggerPromotion, _ = chess.GetValidMovesForPiece(chessMoveData.Piece, currentGameState)

	var data = getChessMoveDataJSON{Moves: moves, Captures: captures, TriggerPromotion: triggerPromotion}

	jsonStr, err := json.Marshal(data)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	w.Write(jsonStr)
}

func joinQueueHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() { app.perfLog.Printf("joinQueueHandler took: %s\n", time.Since(start)) }()

	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		app.clientError(w, http.StatusMethodNotAllowed)
	}

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
		// err = removePlayerFromQueue(playerIDasInt, joinQueue.Time, joinQueue.Increment)
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

func matchFoundSSEHandler(w http.ResponseWriter, r *http.Request) {

	// Set appropriate headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
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
	}

	var playerID = app.sessionManager.GetInt64(r.Context(), "playerID")
	app.infoLog.Printf("playerID in session: %v", playerID)

	// Do the channels get properly closed on a leave queue?
	// Do we send the proper message on a leave queue?
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
	start := time.Now()
	defer func() { app.perfLog.Printf("getChessMovesHandler took: %s\n", time.Since(start)) }()

	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	matchID, err := app.liveMatches.GetHighestEloMatch()
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			app.serverError(w, err, true)
		}
		return
	}

	data := getHighestEloMatchResponse{MatchID: matchID}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonStr)
}

func registerUserHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() { app.perfLog.Printf("registerUserHandler took: %s\n", time.Since(start)) }()

	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

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

	var authData = authData{
		Username: newUser.Username,
	}

	jsonStr, err := json.Marshal(authData)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	app.sessionManager.RememberMe(r.Context(), newUser.RememberMe)
	app.sessionManager.Put(r.Context(), "username", newUser.Username)
	app.sessionManager.Put(r.Context(), "playerID", playerID)
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonStr)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() { app.perfLog.Printf("loginHandler took: %s\n", time.Since(start)) }()

	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	var loginInfo models.UserLoginInfo

	var loginValidationErrors models.UserLoginInfo

	err := json.NewDecoder(r.Body).Decode(&loginInfo)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	playerID, authorized := app.users.Authenticate(loginInfo.Username, loginInfo.Password)
	if !authorized {
		w.WriteHeader(http.StatusUnauthorized)
		loginValidationErrors.Username = "Username or password invalid."
		jsonStr, jsonErr := json.Marshal(loginValidationErrors)
		if jsonErr == nil {
			w.Write(jsonStr)
		} else {
			app.errorLog.Println(jsonErr)
		}
		return
	}

	var authData = authData{
		Username: loginInfo.Username,
	}

	jsonStr, err := json.Marshal(authData)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	app.sessionManager.RememberMe(r.Context(), loginInfo.RememberMe)
	app.sessionManager.Put(r.Context(), "username", loginInfo.Username)
	app.sessionManager.Put(r.Context(), "playerID", playerID)
	w.Write(jsonStr)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() { app.perfLog.Printf("logoutHandler took: %s\n", time.Since(start)) }()

	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	if !app.sessionManager.Exists(r.Context(), "username") {
		app.errorLog.Printf("Not logged in\n")
		w.WriteHeader(http.StatusBadRequest)
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
	start := time.Now()
	defer func() { app.perfLog.Printf("validateSessionHandler took: %s\n", time.Since(start)) }()

	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	if !app.sessionManager.Exists(r.Context(), "username") {
		if !app.sessionManager.Exists(r.Context(), "playerID") {
			app.sessionManager.Put(r.Context(), "playerID", generateNewPlayerId())
		}
		w.WriteHeader(http.StatusUnauthorized)
	}

	var authData = authData{
		Username: app.sessionManager.GetString(r.Context(), "username"),
	}

	jsonStr, err := json.Marshal(authData)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonStr)
}

func userSearchHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() { app.perfLog.Printf("userSearchHandler took: %s\n", time.Since(start)) }()

	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	queryParams := r.URL.Query()

	searchString := queryParams.Get("search")

	if searchString == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userList, err := app.users.SearchForUsers(searchString)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	jsonStr, err := json.Marshal(userList)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonStr)
}

func getTileInfoHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() { app.perfLog.Printf("getTileInfoHandler took: %s\n", time.Since(start)) }()

	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	queryParams := r.URL.Query()

	searchString := queryParams.Get("search")

	if searchString == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tileInfo, err := app.users.GetTileInfoFromUsername(searchString)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	jsonStr, err := json.Marshal(tileInfo)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonStr)
}

func getPastMatchesListHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() { app.perfLog.Printf("getPastMatchesListHandler took: %s\n", time.Since(start)) }()

	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

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

	jsonStr, err := json.Marshal(matchList)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonStr)
}

func getUserAccountSettingsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() { app.perfLog.Printf("getUserAccountSettingsHandler took: %s\n", time.Since(start)) }()

	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	if !app.sessionManager.Exists(r.Context(), "playerID") {
		app.clientError(w, http.StatusUnauthorized)
	}
	playerID := app.sessionManager.GetInt64(r.Context(), "playerID")

	accountSettings, err := app.users.GetUserAccountSettings(playerID)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	jsonStr, err := json.Marshal(accountSettings)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonStr)
}

func updateEmailHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() { app.perfLog.Printf("updateEmailHandler took: %s\n", time.Since(start)) }()

	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	if !app.sessionManager.Exists(r.Context(), "playerID") {
		app.clientError(w, http.StatusUnauthorized)
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
	start := time.Now()
	defer func() { app.perfLog.Printf("updatePasswordHandler took: %s\n", time.Since(start)) }()

	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	if !app.sessionManager.Exists(r.Context(), "playerID") || !app.sessionManager.Exists(r.Context(), "username") {
		app.clientError(w, http.StatusUnauthorized)
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
