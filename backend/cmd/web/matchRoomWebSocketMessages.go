package main

import (
	"burrchess/internal/chess"
	"database/sql"
	"encoding/json"
)

// Hub-to-client message types

type hubMessageType string

const (
	onConnect        hubMessageType = "onConnect"
	onMove           hubMessageType = "onMove"
	connectionStatus hubMessageType = "connectionStatus"
	opponentEvent    hubMessageType = "opponentEvent"
	userMessage      hubMessageType = "userMessage"
	sendPlayerCode   hubMessageType = "sendPlayerCode"
)

// Event types for player-to-player offers and actions

type eventType string

const (
	takeback            eventType = "takeback"
	draw                eventType = "draw"
	resign              eventType = "resign"
	extraTime           eventType = "extraTime"
	abort               eventType = "abort"
	rematch             eventType = "rematch"
	disconnect          eventType = "disconnect"
	decline             eventType = "decline"
	threefoldRepetition eventType = "threefoldRepetition"
)

// Hub-to-client bodies

type onConnectBody struct {
	MatchStateHistory        []MatchStateHistory     `json:"matchStateHistory"`
	GameOverStatusCode       chess.GameOverStatusCode `json:"gameOverStatus"`
	ThreefoldRepetition      bool                    `json:"threefoldRepetition"`
	WhitePlayerConnected     bool                    `json:"whitePlayerConnected"`
	BlackPlayerConnected     bool                    `json:"blackPlayerConnected"`
	MillisecondsUntilTimeout int64                   `json:"millisecondsUntilTimeout"`
	WhitePlayerUsername      sql.NullString          `json:"whitePlayerUsername"`
	BlackPlayerUsername      sql.NullString          `json:"blackPlayerUsername"`
}

type onMoveBody struct {
	MatchStateHistory   []MatchStateHistory     `json:"matchStateHistory"`
	GameOverStatusCode  chess.GameOverStatusCode `json:"gameOverStatus"`
	ThreefoldRepetition bool                    `json:"threefoldRepetition"`
}

type onPlayerConnectionChangeBody struct {
	PlayerColour             string `json:"playerColour"`
	IsConnected              bool   `json:"isConnected"`
	MillisecondsUntilTimeout int64  `json:"millisecondsUntilTimeout"`
}

type opponentEventBody struct {
	Sender    string    `json:"sender"`
	EventType eventType `json:"eventType"`
}

type onUserMessageBody struct {
	Sender         string `json:"sender"`
	MessageContent string `json:"messageContent"`
}

// Hub-to-client responses

type onConnectResponse struct {
	MessageType hubMessageType `json:"messageType"`
	Body        onConnectBody  `json:"body"`
}

type onMoveResponse struct {
	MessageType hubMessageType `json:"messageType"`
	Body        onMoveBody     `json:"body"`
}

type onPlayerConnectionChangeResponse struct {
	MessageType hubMessageType               `json:"messageType"`
	Body        onPlayerConnectionChangeBody `json:"body"`
}

type opponentEventResponse struct {
	MessageType hubMessageType    `json:"messageType"`
	Body        opponentEventBody `json:"body"`
}

type onUserMessageResponse struct {
	MessageType hubMessageType    `json:"messageType"`
	Body        onUserMessageBody `json:"body"`
}

type sendPlayerCodeResponse struct {
	MessageType hubMessageType     `json:"messageType"`
	Body        sendPlayerCodeBody `json:"body"`
}

type sendPlayerCodeBody struct {
	PlayerCode messageIdentifier `json:"playerCode"`
}

// Client-to-hub message types

type clientMessageType string

const (
	postMove    clientMessageType = "postMove"
	playerEvent clientMessageType = "playerEvent"
	chatMessage clientMessageType = "userMessage"
	unknown     clientMessageType = "unknown"
)

// Client-to-hub bodies

type postMoveBody struct {
	Piece           int    `json:"piece"`
	Move            int    `json:"move"`
	PromotionString string `json:"promotionString"`
}

type playerEventBody struct {
	EventType eventType `json:"eventType"`
}

type userMessageBody struct {
	MessageContent string `json:"messageContent"`
}

// Client-to-hub responses

type postMoveResponse struct {
	MessageType clientMessageType `json:"messageType"`
	Body        postMoveBody      `json:"body"`
}

type playerEventResponse struct {
	MessageType clientMessageType `json:"messageType"`
	Body        playerEventBody   `json:"body"`
}

type userMessageResponse struct {
	MessageType clientMessageType `json:"messageType"`
	Body        userMessageBody   `json:"body"`
}

// Shared types

type userJSON struct {
	MessageType string          `json:"messageType"`
	Body        json.RawMessage `json:"body"`
}

type MatchStateHistory struct {
	FEN                                  string `json:"FEN"`
	LastMove                             [2]int `json:"lastMove"`
	AlgebraicNotation                    string `json:"algebraicNotation"`
	WhitePlayerTimeRemainingMilliseconds int64  `json:"whitePlayerTimeRemainingMilliseconds"`
	BlackPlayerTimeRemainingMilliseconds int64  `json:"blackPlayerTimeRemainingMilliseconds"`
}

type offerInfo struct {
	sender messageIdentifier
	event  eventType
}
