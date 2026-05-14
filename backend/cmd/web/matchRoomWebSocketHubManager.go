package main

import (
	"database/sql"
	"errors"
	"sync"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

type MatchRoomHubManager struct {
	mu   sync.Mutex
	hubs map[int64]*MatchRoomHub
}

func newMatchRoomHubManager() *MatchRoomHubManager {
	return &MatchRoomHubManager{
		hubs: make(map[int64]*MatchRoomHub),
	}
}

var matchRoomHubManager = newMatchRoomHubManager()

func (hubManager *MatchRoomHubManager) unregisterHub(matchID int64) {
	hubManager.mu.Lock()
	defer hubManager.mu.Unlock()
	delete(hubManager.hubs, matchID)
}

func (hubManager *MatchRoomHubManager) getHubFromMatchID(matchID int64, app *application) (*MatchRoomHub, error) {
	// Fast path
	hubManager.mu.Lock()
	val, ok := hubManager.hubs[matchID]
	hubManager.mu.Unlock()
	if ok {
		return val, nil
	}

	// Slow path: create hub outside the lock
	newHub, err := newMatchRoomHub(matchID, app) // DB call happens here, no lock held
	if err != nil {
		return nil, err
	}

	// Re-lock and insert only if another goroutine hasn't beaten us to it
	hubManager.mu.Lock()
	if existing, ok := hubManager.hubs[matchID]; ok {
		hubManager.mu.Unlock()
		return existing, nil // discard newHub; it has no running goroutine so it's GC'd
	}
	hubManager.hubs[matchID] = newHub
	hubManager.mu.Unlock()

	go newHub.run()
	return newHub, nil
}

func (hubManager *MatchRoomHubManager) registerClientToMatchRoomHub(conn *websocket.Conn, matchID int64, playerID *int64, app *application) (*MatchRoomHubClient, error) {
	val, err := hubManager.getHubFromMatchID(matchID, app)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			app.logger.Warn("match not found", "matchID", matchID)
		} else {
			app.logger.Error("failed to get hub for match", "matchID", matchID, "err", err)
		}
		return nil, err
	}

	var playerCode messageIdentifier = messageIdentifier(Spectator)

	if playerID == nil {
		// Do nothing
	} else if *playerID == val.players[WhitePlayer].id {
		playerCode = messageIdentifier(WhitePlayer)
	} else if *playerID == val.players[BlackPlayer].id {
		playerCode = messageIdentifier(BlackPlayer)
	}

	return &MatchRoomHubClient{hub: val, conn: conn, playerIdentifier: playerCode, send: make(chan []byte, 256), limiter: rate.NewLimiter(wsRateLimit, wsRateBurst)}, nil
}
