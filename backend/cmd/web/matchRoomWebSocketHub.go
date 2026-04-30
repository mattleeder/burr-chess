package main

import (
	"burrchess/internal/chess"
	"burrchess/internal/models"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

const pingTimeout = 20 * time.Second

type playerTurn byte

const (
	WhiteTurn = byte(iota)
	BlackTurn = byte(iota)
)

type playerState struct {
	id             int64
	username       sql.NullString
	timeRemaining  time.Duration
	connected      bool
	timeout        <-chan time.Time
	timeoutStarted time.Time
	elo            int64
}

func opponent(color byte) byte {
	return 1 - color
}

func colorName(color byte) string {
	if color == WhitePlayer {
		return "white"
	}
	return "black"
}

// MatchRoomHub maintains the set of active clients and broadcasts messages to the clients.
type MatchRoomHub struct {
	matchID int64

	clients    map[*MatchRoomHubClient]bool
	broadcast  chan []byte
	register   chan *MatchRoomHubClient
	unregister chan *MatchRoomHubClient

	players [2]playerState

	isTimerActive    bool
	turn             playerTurn
	currentGameState []byte // marshalled onMoveResponse
	currentFEN       string
	moveHistory      []MatchStateHistory
	timeOfLastMove   time.Time
	flagTimer        <-chan time.Time

	timeFormatInMilliseconds int64
	increment                time.Duration
	fenFreqMap               map[string]int

	// opponentCanClaim[i] means player i can claim their opponent disconnected
	opponentCanClaim [2]bool

	offerActive           *offerInfo
	gameEnded             bool
	isThreefoldRepetition bool
	averageElo            float64
	matchStartTime        int64
	taskQueueWaitGroup    *sync.WaitGroup
}

func newMatchRoomHub(matchID int64) (*MatchRoomHub, error) {
	matchState, err := app.liveMatches.EnQueueReturnGetFromMatchID(matchID, nil, nil)
	if err != nil {
		app.errorLog.Println(err)
		return nil, err
	}

	var matchStateHistory []MatchStateHistory
	err = json.Unmarshal(matchState.GameHistoryJSONString, &matchStateHistory)
	if err != nil {
		app.errorLog.Printf("Error unmarshalling matchStateHistory %v\n", err)
		return nil, err
	}

	fenFreqMap := make(map[string]int)
	isThreefoldRepetition := false

	for _, val := range matchStateHistory {
		fen := strings.Join(strings.Split(val.FEN, " ")[:4], " ")
		fenFreqMap[fen]++
		if fenFreqMap[fen] >= 3 {
			isThreefoldRepetition = true
		}
	}

	currentGameState := onMoveResponse{
		MessageType: onMove,
		Body: onMoveBody{
			MatchStateHistory:   matchStateHistory,
			GameOverStatusCode:  chess.Ongoing,
			ThreefoldRepetition: isThreefoldRepetition,
		},
	}

	timeOfLastMove := time.UnixMilli(matchState.UnixMsTimeOfLastMove)
	splitFEN := strings.Split(matchState.CurrentFEN, " ")

	var turn playerTurn
	if splitFEN[1] == "w" {
		turn = playerTurn(WhiteTurn)
	} else {
		turn = playerTurn(BlackTurn)
	}

	var players [2]playerState
	players[WhitePlayer] = playerState{
		id:            matchState.WhitePlayerID,
		username:      matchState.WhitePlayerUsername,
		timeRemaining: time.Duration(matchState.WhitePlayerTimeRemainingMilliseconds) * time.Millisecond,
		elo:           matchState.WhitePlayerElo,
	}
	players[BlackPlayer] = playerState{
		id:            matchState.BlackPlayerID,
		username:      matchState.BlackPlayerUsername,
		timeRemaining: time.Duration(matchState.BlackPlayerTimeRemainingMilliseconds) * time.Millisecond,
		elo:           matchState.BlackPlayerElo,
	}

	isTimerActive := splitFEN[5] != "1"
	var flagTimer <-chan time.Time

	if isTimerActive {
		idx := byte(turn)
		players[idx].timeRemaining -= time.Since(timeOfLastMove)
		flagTimer = time.After(players[idx].timeRemaining)
	}

	jsonStr, err := json.Marshal(currentGameState)
	if err != nil {
		app.errorLog.Printf("Error marshalling JSON: %v\n", err)
		return nil, err
	}

	hub := &MatchRoomHub{
		matchID:                  matchID,
		broadcast:                make(chan []byte),
		register:                 make(chan *MatchRoomHubClient),
		unregister:               make(chan *MatchRoomHubClient),
		clients:                  make(map[*MatchRoomHubClient]bool),
		players:                  players,
		isTimerActive:            isTimerActive,
		turn:                     turn,
		currentGameState:         jsonStr,
		currentFEN:               matchState.CurrentFEN,
		moveHistory:              currentGameState.Body.MatchStateHistory,
		timeOfLastMove:           timeOfLastMove,
		flagTimer:                flagTimer,
		timeFormatInMilliseconds: matchState.TimeFormatInMilliseconds,
		increment:                time.Duration(matchState.IncrementInMilliseconds) * time.Millisecond,
		fenFreqMap:               fenFreqMap,
		isThreefoldRepetition:    isThreefoldRepetition,
		averageElo:               matchState.AverageElo,
		matchStartTime:           matchState.MatchStartTime,
	}

	return hub, nil
}

// Message sending

func (hub *MatchRoomHub) sendMessageToAllClients(message []byte) {
	for client := range hub.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(hub.clients, client)
		}
	}
}

func (hub *MatchRoomHub) sendMessageToOnePlayer(message []byte, colour messageIdentifier) {
	for client := range hub.clients {
		if client.playerIdentifier != colour {
			continue
		}
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(hub.clients, client)
		}
	}
}

func (hub *MatchRoomHub) hasActiveClients() bool {
	return len(hub.clients) > 0
}

// Turn and timer management

func (hub *MatchRoomHub) changeTurn() {
	hub.turn = playerTurn(opponent(byte(hub.turn)))

	if hub.isTimerActive {
		hub.flagTimer = time.After(hub.players[byte(hub.turn)].timeRemaining)
	} else if hub.turn == playerTurn(WhiteTurn) {
		// Timer activates after black's first move
		hub.flagTimer = time.After(hub.players[WhitePlayer].timeRemaining)
		hub.isTimerActive = true
	}
}

func (hub *MatchRoomHub) updateTimeRemaining() {
	if !hub.isTimerActive {
		return
	}
	idx := byte(hub.turn)
	hub.players[idx].timeRemaining -= time.Since(hub.timeOfLastMove)
	hub.players[idx].timeRemaining += hub.increment
}

// Game outcome

func (hub *MatchRoomHub) getOutcomeInt(gameOverStatus chess.GameOverStatusCode) int {
	switch gameOverStatus {
	case chess.Checkmate:
		if hub.turn == chess.Black {
			return 2
		}
		return 1
	case chess.WhiteFlagged, chess.WhiteResigned, chess.WhiteDisconnected:
		return 2
	case chess.BlackFlagged, chess.BlackResigned, chess.BlackDisconnected:
		return 1
	}
	return 0
}

// Move handling

func (hub *MatchRoomHub) updateGameStateAfterMove(message []byte) error {
	var chessMove postMoveResponse
	err := json.Unmarshal(message[1:], &chessMove)
	if err != nil {
		return fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	if !chess.IsMoveValid(hub.currentFEN, chessMove.Body.Piece, chessMove.Body.Move) {
		return errors.New("move is not valid")
	}

	hub.updateTimeRemaining()

	newFEN, gameOverStatus, algebraicNotation := chess.GetFENAfterMove(hub.currentFEN, chessMove.Body.Piece, chessMove.Body.Move, chessMove.Body.PromotionString)

	splitFEN := strings.Join(strings.Split(newFEN, " ")[:4], " ")
	hub.fenFreqMap[splitFEN]++
	hub.isThreefoldRepetition = hub.fenFreqMap[splitFEN] >= 3

	data := onMoveResponse{
		MessageType: onMove,
		Body: onMoveBody{
			MatchStateHistory: append(hub.moveHistory, MatchStateHistory{
				FEN:                                  newFEN,
				LastMove:                             [2]int{chessMove.Body.Piece, chessMove.Body.Move},
				AlgebraicNotation:                    algebraicNotation,
				WhitePlayerTimeRemainingMilliseconds: hub.players[WhitePlayer].timeRemaining.Milliseconds(),
				BlackPlayerTimeRemainingMilliseconds: hub.players[BlackPlayer].timeRemaining.Milliseconds(),
			}),
			GameOverStatusCode:  gameOverStatus,
			ThreefoldRepetition: hub.isThreefoldRepetition,
		},
	}

	jsonStr, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %v", err)
	}

	hub.currentFEN = newFEN
	hub.currentGameState = jsonStr
	hub.moveHistory = data.Body.MatchStateHistory
	hub.timeOfLastMove = time.Now()

	hub.changeTurn()

	matchStateHistoryData, err := json.Marshal(data.Body.MatchStateHistory)
	if err != nil {
		app.errorLog.Printf("Error marshalling matchStateHistoryData: %s", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	app.liveMatches.EnQueueUpdateLiveMatch(models.UpdateMatchParams{
		MatchID:                              hub.matchID,
		NewFEN:                               newFEN,
		LastMovePiece:                        chessMove.Body.Piece,
		LastMoveMove:                         chessMove.Body.Move,
		WhitePlayerTimeRemainingMilliseconds: hub.players[WhitePlayer].timeRemaining.Milliseconds(),
		BlackPlayerTimeRemainingMilliseconds: hub.players[BlackPlayer].timeRemaining.Milliseconds(),
		MatchStateHistoryJSON:                matchStateHistoryData,
		TimeOfLastMove:                       hub.timeOfLastMove,
	}, hub.taskQueueWaitGroup, &wg)
	hub.taskQueueWaitGroup = &wg

	if gameOverStatus != chess.Ongoing {
		return hub.endGame(gameOverStatus)
	}

	return nil
}

// Connection state for new clients

func (hub *MatchRoomHub) getCurrentMatchStateForNewConnection(playerIdentifier messageIdentifier) ([]byte, error) {
	var gameState onMoveResponse
	err := json.Unmarshal(hub.currentGameState, &gameState)
	if err != nil {
		app.errorLog.Printf("Error unmarshalling JSON: %v\n", err)
		return nil, err
	}

	if hub.isTimerActive {
		latest := &gameState.Body.MatchStateHistory[len(gameState.Body.MatchStateHistory)-1]
		elapsed := time.Since(hub.timeOfLastMove).Milliseconds()
		if hub.turn == playerTurn(WhiteTurn) {
			latest.WhitePlayerTimeRemainingMilliseconds -= elapsed
		} else {
			latest.BlackPlayerTimeRemainingMilliseconds -= elapsed
		}
	}

	var millisecondsUntilTimeout int64
	idx := byte(playerIdentifier)
	if idx <= BlackPlayer {
		opp := opponent(idx)
		if !hub.players[opp].connected {
			millisecondsUntilTimeout = pingTimeout.Milliseconds() - time.Since(hub.players[opp].timeoutStarted).Milliseconds()
		}
	}

	response := onConnectResponse{
		MessageType: onConnect,
		Body: onConnectBody{
			MatchStateHistory:        gameState.Body.MatchStateHistory,
			GameOverStatusCode:       gameState.Body.GameOverStatusCode,
			ThreefoldRepetition:      gameState.Body.ThreefoldRepetition,
			WhitePlayerConnected:     hub.players[WhitePlayer].connected,
			BlackPlayerConnected:     hub.players[BlackPlayer].connected,
			MillisecondsUntilTimeout: millisecondsUntilTimeout,
			WhitePlayerUsername:      hub.players[WhitePlayer].username,
			BlackPlayerUsername:      hub.players[BlackPlayer].username,
		},
	}

	jsonStr, err := json.Marshal(response)
	if err != nil {
		app.errorLog.Printf("Error marshalling JSON: %v\n", err)
		return nil, err
	}

	return jsonStr, nil
}

// Message parsing and routing

func (hub *MatchRoomHub) getMessageType(message []byte) clientMessageType {
	var msg userJSON
	err := json.Unmarshal(message[1:], &msg)
	if err != nil {
		app.errorLog.Printf("Unable to parse message into userJSON: %s\n", err)
		return unknown
	}

	switch msg.MessageType {
	case "postMove", "playerEvent":
		if message[0] != byte(WhitePlayer) && message[0] != byte(BlackPlayer) {
			app.errorLog.Printf("Non-player trying to send %s: %s\n", msg.MessageType, message)
			return unknown
		}
		if msg.MessageType == "postMove" {
			return postMove
		}
		return playerEvent
	}

	app.errorLog.Printf("Unknown message type\n")
	return unknown
}

func (hub *MatchRoomHub) handleMessage(message []byte) {
	switch hub.getMessageType(message) {
	case postMove:
		if hub.gameEnded {
			return
		}
		if message[0] != byte(hub.turn) {
			return
		}
		err := hub.updateGameStateAfterMove(message)
		if err != nil {
			app.errorLog.Println(err)
			return
		}
		hub.sendMessageToAllClients(hub.currentGameState)

	case playerEvent:
		if hub.gameEnded {
			return
		}
		hub.handlePlayerEvent(message)

	default:
		app.errorLog.Printf("Could not understand message: %s\n", message)
	}
}

// Player events

func isOneSidedEvent(event eventType) bool {
	return event == extraTime || event == resign || event == abort || event == disconnect
}

func (hub *MatchRoomHub) handlePlayerEvent(message []byte) {
	var data playerEventResponse
	err := json.Unmarshal(message[1:], &data)
	if err != nil {
		app.errorLog.Printf("Could not unmarshal playerEvent: %s", err)
		return
	}

	if data.Body.EventType == threefoldRepetition {
		if hub.isThreefoldRepetition {
			hub.endGame(chess.ThreefoldRepetition)
			hub.sendMessageToAllClients(hub.currentGameState)
		}
		return
	}

	sender := messageIdentifier(message[0])
	if isOneSidedEvent(data.Body.EventType) {
		hub.oneSidedEvent(sender, data.Body.EventType)
	} else if hub.offerActive == nil || hub.offerActive.event != data.Body.EventType {
		hub.makeNewEventOffer(sender, data.Body.EventType)
	} else if hub.offerActive != nil && byte(hub.offerActive.sender) != message[0] {
		hub.acceptEventOffer(data.Body.EventType)
	}
}

func (hub *MatchRoomHub) makeNewEventOffer(sender messageIdentifier, event eventType) {
	app.infoLog.Printf("Making new event from %v of type %s\n", sender, event)
	hub.offerActive = &offerInfo{sender, event}

	senderIdx := byte(sender)
	receiverIdx := opponent(senderIdx)

	response := opponentEventResponse{
		MessageType: opponentEvent,
		Body:        opponentEventBody{Sender: colorName(senderIdx), EventType: event},
	}

	jsonStr, err := json.Marshal(response)
	if err != nil {
		app.errorLog.Printf("Could not marshal opponentEventResponse: %s\n", err)
		return
	}

	hub.sendMessageToOnePlayer(jsonStr, messageIdentifier(receiverIdx))
}

func (hub *MatchRoomHub) oneSidedEvent(sender messageIdentifier, event eventType) {
	senderIdx := byte(sender)

	switch event {
	case extraTime:
		return
	case resign:
		resignCodes := [2]chess.GameOverStatusCode{chess.WhiteResigned, chess.BlackResigned}
		hub.endGame(resignCodes[senderIdx])
		hub.sendMessageToAllClients(hub.currentGameState)
	case disconnect:
		if hub.opponentCanClaim[senderIdx] {
			disconnectCodes := [2]chess.GameOverStatusCode{chess.BlackDisconnected, chess.WhiteDisconnected}
			hub.endGame(disconnectCodes[senderIdx])
			hub.sendMessageToAllClients(hub.currentGameState)
		}
	}
}

// @TODO: implement this
func (hub *MatchRoomHub) takeBack() {
}

func (hub *MatchRoomHub) acceptEventOffer(event eventType) {
	app.infoLog.Printf("Accepting event of type %s\n", event)
	switch event {
	case takeback:
		hub.takeBack()
	case draw:
		hub.endGame(chess.Draw)
		hub.sendMessageToAllClients(hub.currentGameState)
	}
}

// End game and ELO

func (hub *MatchRoomHub) endGame(reason chess.GameOverStatusCode) error {
	app.infoLog.Println("Ending Match")
	hub.flagTimer = nil

	var gameState onMoveResponse
	err := json.Unmarshal(hub.currentGameState, &gameState)
	if err != nil {
		app.errorLog.Printf("Error unmarshalling JSON: %v\n", err)
		return err
	}

	gameState.Body.GameOverStatusCode = reason

	jsonStr, err := json.Marshal(gameState)
	if err != nil {
		app.errorLog.Printf("Error marshalling JSON: %v\n", err)
		return err
	}

	hub.currentGameState = jsonStr
	outcome := hub.getOutcomeInt(reason)

	var whitePoints, blackPoints float64
	switch outcome {
	case 1:
		whitePoints, blackPoints = 1, 0
	case 2:
		whitePoints, blackPoints = 0, 1
	default:
		whitePoints, blackPoints = 0.5, 0.5
	}

	whiteEloGain, blackEloGain := calculateEloChanges(hub.players[WhitePlayer].elo, whitePoints, hub.players[BlackPlayer].elo, blackPoints)
	app.infoLog.Printf("whitePlayerElo: %v, whitePlayerEloGain: %v\n", hub.players[WhitePlayer].elo, whiteEloGain)

	whiteNewElo := int64(math.Max(float64(hub.players[WhitePlayer].elo)+math.Round(whiteEloGain), 0))
	blackNewElo := int64(math.Max(float64(hub.players[BlackPlayer].elo)+math.Round(blackEloGain), 0))

	ratingType := models.GetRatingTypeFromTimeFormat(hub.timeFormatInMilliseconds)
	go app.userRatings.UpdateRatingFromPlayerID(hub.players[WhitePlayer].id, ratingType, whiteNewElo)
	go app.userRatings.UpdateRatingFromPlayerID(hub.players[BlackPlayer].id, ratingType, blackNewElo)

	hub.gameEnded = true
	app.liveMatches.EnQueueMoveMatchToPastMatches(hub.matchID, outcome, reason, whiteNewElo-hub.players[WhitePlayer].elo, blackNewElo-hub.players[BlackPlayer].elo, hub.taskQueueWaitGroup, nil)
	return nil
}

// Connection status

func (hub *MatchRoomHub) pingStatusMessage(playerColour string, isConnected bool, millisecondsUntilTimeout int64) ([]byte, error) {
	data := onPlayerConnectionChangeResponse{
		MessageType: connectionStatus,
		Body:        onPlayerConnectionChangeBody{PlayerColour: playerColour, IsConnected: isConnected, MillisecondsUntilTimeout: millisecondsUntilTimeout},
	}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		app.errorLog.Printf("Unable to marshal pingStatus: %s", err)
	}
	return jsonStr, err
}

func (hub *MatchRoomHub) setConnected(client *MatchRoomHubClient) {
	idx := byte(client.playerIdentifier)
	if idx > BlackPlayer {
		return
	}
	hub.players[idx].connected = true
	hub.opponentCanClaim[opponent(idx)] = false
	hub.players[idx].timeout = nil

	pingMessage, err := hub.pingStatusMessage(colorName(idx), true, 0)
	if err != nil {
		app.errorLog.Printf("Could not generate pingMessage: %s", err)
		return
	}
	hub.sendMessageToAllClients(pingMessage)
}

func (hub *MatchRoomHub) setDisconnected(client *MatchRoomHubClient) {
	idx := byte(client.playerIdentifier)
	if idx > BlackPlayer {
		return
	}
	hub.players[idx].connected = false
	if !hub.gameEnded {
		hub.players[idx].timeout = time.After(pingTimeout)
		hub.players[idx].timeoutStarted = time.Now()
	}

	pingMessage, err := hub.pingStatusMessage(colorName(idx), false, pingTimeout.Milliseconds())
	if err != nil {
		app.errorLog.Printf("Could not generate pingMessage: %s", err)
		return
	}
	hub.sendMessageToAllClients(pingMessage)
}

// Main event loop

func (hub *MatchRoomHub) run() {
	app.infoLog.Println("Hub running")
	defer app.infoLog.Println("Hub stopped")
	for {
		select {
		case client := <-hub.register:
			hub.clients[client] = true
			hub.setConnected(client)
			jsonStr, err := hub.getCurrentMatchStateForNewConnection(client.playerIdentifier)
			if err != nil {
				app.errorLog.Printf("Could not get json for new connection: %v\n", err)
				continue
			}
			client.send <- jsonStr

		case client := <-hub.unregister:
			if _, ok := hub.clients[client]; ok {
				delete(hub.clients, client)
				close(client.send)
			}
			hub.setDisconnected(client)
			if !hub.hasActiveClients() {
				matchRoomHubManager.unregisterHub(hub.matchID)
				return
			}

		case <-hub.flagTimer:
			flagCodes := [2]chess.GameOverStatusCode{chess.WhiteFlagged, chess.BlackFlagged}
			err := hub.endGame(flagCodes[byte(hub.turn)])
			if err != nil {
				app.errorLog.Println(err)
				continue
			}
			hub.sendMessageToAllClients(hub.currentGameState)

		case <-hub.players[WhitePlayer].timeout:
			hub.opponentCanClaim[BlackPlayer] = true

		case <-hub.players[BlackPlayer].timeout:
			hub.opponentCanClaim[WhitePlayer] = true

		case message := <-hub.broadcast:
			app.infoLog.Printf("WS Message: %s\n", message)
			hub.handleMessage(message)
		}
	}
}
