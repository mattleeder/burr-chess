package main

import (
	"sync"

	"github.com/gorilla/websocket"
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

func (hubManager *MatchRoomHubManager) registerNewHub(matchID int64) (*MatchRoomHub, error) {
	newHub, err := newMatchRoomHub(matchID)
	if err != nil {
		app.errorLog.Println(err)
		return nil, err
	}
	hubManager.hubs[matchID] = newHub
	return hubManager.hubs[matchID], nil
}

func (hubManager *MatchRoomHubManager) unregisterHub(matchID int64) {
	hubManager.mu.Lock()
	defer hubManager.mu.Unlock()
	delete(hubManager.hubs, matchID)
}

func (hubManager *MatchRoomHubManager) getHubFromMatchID(matchID int64) (*MatchRoomHub, error) {
	hubManager.mu.Lock()
	defer hubManager.mu.Unlock()
	val, ok := hubManager.hubs[matchID]

	if !ok {
		var err error
		val, err = hubManager.registerNewHub(matchID)
		if err != nil {
			app.errorLog.Println(err)
			return nil, err
		}
		go val.run()
	}

	return val, nil
}

func (hubManager *MatchRoomHubManager) registerClientToMatchRoomHub(conn *websocket.Conn, matchID int64, playerID *int64) (*MatchRoomHubClient, error) {
	val, err := hubManager.getHubFromMatchID(matchID)
	if err != nil {
		app.errorLog.Println(err)
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

	return &MatchRoomHubClient{hub: val, conn: conn, playerIdentifier: playerCode, send: make(chan []byte, 256)}, nil
}
