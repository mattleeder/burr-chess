package main

import (
	"bytes"
	"encoding/json"
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
			app.infoLog.Printf("WS CheckOrigin: got=%q want=%q", origin, want)
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
		c.hub.unregister <- c
		c.conn.Close()
		app.infoLog.Println("Client closed")
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		app.infoLog.Println(string(message))
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				app.errorLog.Printf("error: %v", err)
			}
			break
		}

		if !c.limiter.Allow() {
			app.infoLog.Printf("WS rate limit exceeded for player %d", c.playerIdentifier)
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
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
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
func serveMatchroomWs(w http.ResponseWriter, r *http.Request) {
	app.infoLog.Println("WS Request")
	app.infoLog.Printf("Session token: %s\n", app.sessionManager.Token(r.Context()))

	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		app.serverError(w, err, false)
		return
	}

	matchID, err := strconv.ParseInt(r.PathValue("matchID"), 10, 64)
	if err != nil {
		app.errorLog.Println(err)
		http.Error(w, "Could not find match", http.StatusInternalServerError)
		return
	}

	playerID, ok := app.sessionPlayerID(w, r)
	if !ok {
		return
	}

	upgrader := app.newUpgrader()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.serverError(w, err, false)
		return
	}

	client, err := matchRoomHubManager.registerClientToMatchRoomHub(conn, matchID, &playerID)
	if err != nil {
		app.websocketError(conn, err)
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
}
