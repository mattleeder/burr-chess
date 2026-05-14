package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

type messageIdentifier byte

const (
	WhitePlayer = byte(iota)
	BlackPlayer = byte(iota)
	Spectator   = byte(iota)
)

func (id messageIdentifier) String() string {
	switch id {
	case messageIdentifier(WhitePlayer):
		return "white"
	case messageIdentifier(BlackPlayer):
		return "black"
	case messageIdentifier(Spectator):
		return "spectator"
	default:
		return "unknown"
	}
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 20 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

func (app *application) newUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := strings.TrimRight(r.Header.Get("Origin"), "/")
			want := strings.TrimRight(app.allowedOrigin, "/")
			return origin == want
		},
	}
}

const (
	wsRateLimit = 5  // messages per second
	wsRateBurst = 10 // burst allowance
)

type MatchRoomHubClient struct {
	hub              *MatchRoomHub
	conn             *websocket.Conn
	playerIdentifier messageIdentifier
	send             chan []byte
	limiter          *rate.Limiter
}

// readPump pumps messages from the websocket connection to the hub.
func (c *MatchRoomHubClient) readPump() {
	defer func() {
		c.hub.app.logger.Info("websocket closed", "matchID", c.hub.matchID, "playerIdentifier", c.playerIdentifier, "colour", c.playerIdentifier)
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.app.logger.Warn("unexpected websocket close", "matchID", c.hub.matchID, "colour", c.playerIdentifier, "err", err)
			}
			break
		}

		if !c.limiter.Allow() {
			c.hub.app.logger.Warn("WebSocket message rate limited", "matchID", c.hub.matchID, "colour", c.playerIdentifier)
			continue
		}
		sender := []byte{byte(c.playerIdentifier)}
		message = append(sender, message...)
		message = bytes.TrimSpace(bytes.ReplaceAll(message, newline, space))
		c.hub.broadcast <- message
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *MatchRoomHubClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				return
			}

			n := len(c.send)
			for i := 0; i < n; i++ {
				if _, err := w.Write(newline); err != nil {
					return
				}
				if _, err := w.Write(<-c.send); err != nil {
					return
				}
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveMatchroomWs handles websocket requests from the peer.
func (app *application) serveMatchroomWs(w http.ResponseWriter, r *http.Request) {

	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		app.serverError(w, err)
		return
	}

	matchID, err := strconv.ParseInt(r.PathValue("matchID"), 10, 64)
	if err != nil || matchID <= 0 {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	playerID, ok := app.sessionPlayerID(w, r)
	if !ok {
		return
	}

	upgrader := app.newUpgrader()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.serverError(w, err)
		return
	}

	client, err := matchRoomHubManager.registerClientToMatchRoomHub(conn, matchID, &playerID, app)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			app.websocketError(conn, err)
		} else {
			conn.WriteMessage(websocket.CloseMessage, []byte{})
			conn.Close()
		}
		return
	}

	codeMessage := sendPlayerCodeResponse{
		MessageType: sendPlayerCode,
		Body:        sendPlayerCodeBody{PlayerCode: client.playerIdentifier},
	}
	jsonStr, err := json.Marshal(codeMessage)
	if err != nil {
		app.websocketError(conn, err)
		return
	}

	client.send <- jsonStr
	client.hub.register <- client

	go client.writePump()
	go client.readPump()

	app.logger.Info("websocket opened", "matchID", matchID, "playerID", playerID, "colour", client.playerIdentifier)
}
