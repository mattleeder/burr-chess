package main

import (
	"burrchess/internal/models"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

type playerMatchmakingData struct {
	playerID             int64
	elo                  int64
	matchmakingThreshold int64
	isMatched            bool
}

type matchingScore struct {
	playerOneID  int64
	playerOneIdx int
	playerTwoID  int64
	playerTwoIdx int
	score        int64
}

type queueKey struct {
	timeFormatInMilliseconds int64
	incrementInMilliseconds  int64
}

type QueueData struct {
	mu                       sync.Mutex
	waitingPool              []*playerMatchmakingData
	matchmakingPool          []*playerMatchmakingData
	awaitingRemoval          map[int64]bool
	timeFormatInMilliseconds int64
	incrementInMilliseconds  int64
}

var (
	queueMu  sync.Mutex
	queueMap = make(map[queueKey]*QueueData)
)

const defaultMatchmakingThreshold = 400
const matchmakingThresholdIncrement = 50

func getOrCreateQueue(timeFormatInMilliseconds int64, incrementInMilliseconds int64) *QueueData {
	key := queueKey{timeFormatInMilliseconds, incrementInMilliseconds}
	queueMu.Lock()
	defer queueMu.Unlock()

	queue, ok := queueMap[key]
	if !ok {
		app.infoLog.Printf("Creating new queue: %v %v\n", timeFormatInMilliseconds, incrementInMilliseconds)
		queue = &QueueData{
			awaitingRemoval:          make(map[int64]bool),
			timeFormatInMilliseconds: timeFormatInMilliseconds,
			incrementInMilliseconds:  incrementInMilliseconds,
		}
		queueMap[key] = queue
	}
	return queue
}

func addPlayerToWaitingPool(playerID int64, timeFormatInMilliseconds int64, incrementInMilliseconds int64) {
	queue := getOrCreateQueue(timeFormatInMilliseconds, incrementInMilliseconds)

	queue.mu.Lock()
	defer queue.mu.Unlock()

	// If player is already in queue, cancel any pending removal
	if _, ok := queue.awaitingRemoval[playerID]; ok {
		queue.awaitingRemoval[playerID] = false
		return
	}

	var elo int64
	playerRatings, err := app.userRatings.GetRatingFromPlayerID(playerID)
	if err != nil {
		elo = 1500
	} else {
		elo = playerRatings.GetRatingForTimeFormat(timeFormatInMilliseconds)
	}

	queue.waitingPool = append(queue.waitingPool, &playerMatchmakingData{
		playerID:             playerID,
		elo:                  elo,
		matchmakingThreshold: defaultMatchmakingThreshold,
		isMatched:            false,
	})

	queue.awaitingRemoval[playerID] = false
}

func removePlayerFromWaitingPool(playerID int64, timeFormatInMilliseconds int64, incrementInMilliseconds int64) {
	key := queueKey{timeFormatInMilliseconds, incrementInMilliseconds}

	queueMu.Lock()
	queue, ok := queueMap[key]
	queueMu.Unlock()

	if !ok {
		app.errorLog.Printf("Queue not found for time=%v inc=%v\n", timeFormatInMilliseconds, incrementInMilliseconds)
		return
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	if _, ok := queue.awaitingRemoval[playerID]; ok {
		queue.awaitingRemoval[playerID] = true
	}
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func calculateMatchingScore(playerOne *playerMatchmakingData, playerOneIdx int, playerTwo *playerMatchmakingData, playerTwoIdx int) *matchingScore {
	return &matchingScore{
		playerOneID:  playerOne.playerID,
		playerOneIdx: playerOneIdx,
		playerTwoID:  playerTwo.playerID,
		playerTwoIdx: playerTwoIdx,
		score:        abs(playerOne.elo - playerTwo.elo),
	}
}

func swapRemove[T any](arr []T, idx int) []T {
	arr[idx] = arr[len(arr)-1]
	return arr[:len(arr)-1]
}

func startingMatchHistory(timeFormatInMilliseconds int64) ([]byte, error) {
	startingHistory := []MatchStateHistory{{
		FEN:                                  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		LastMove:                             [2]int{0, 0},
		AlgebraicNotation:                    "a8",
		WhitePlayerTimeRemainingMilliseconds: timeFormatInMilliseconds,
		BlackPlayerTimeRemainingMilliseconds: timeFormatInMilliseconds,
	}}

	jsonStr, err := json.Marshal(startingHistory)
	if err != nil {
		app.errorLog.Printf("Error marshalling JSON: %v\n", err)
		return nil, err
	}

	return jsonStr, nil
}

func createMatch(playerOneData *playerMatchmakingData, playerTwoData *playerMatchmakingData, timeFormatInMilliseconds int64, incrementInMilliseconds int64) error {
	playerOneID := playerOneData.playerID
	playerTwoID := playerTwoData.playerID

	playerOneIsWhite := rand.Intn(2) == 1

	var whitePlayerData, blackPlayerData *playerMatchmakingData
	if playerOneIsWhite {
		whitePlayerData = playerOneData
		blackPlayerData = playerTwoData
	} else {
		whitePlayerData = playerTwoData
		blackPlayerData = playerOneData
	}
	startingHistory, err := startingMatchHistory(timeFormatInMilliseconds)
	if err != nil {
		app.errorLog.Printf("Error creating starting history for new match: %v\n", err)
		return err
	}

	var averageElo float64 = (float64(playerOneData.elo) + float64(playerTwoData.elo)) / 2

	var matchID int64
	matchID, err = app.liveMatches.EnQueueReturnInsertNew(models.InsertNewParams{
		PlayerOneID:              playerOneID,
		PlayerTwoID:              playerTwoID,
		PlayerOneIsWhite:         playerOneIsWhite,
		TimeFormatInMilliseconds: timeFormatInMilliseconds,
		IncrementInMilliseconds:  incrementInMilliseconds,
		GameHistory:              startingHistory,
		AverageElo:               averageElo,
		WhitePlayerElo:           whitePlayerData.elo,
		BlackPlayerElo:           blackPlayerData.elo,
	}, nil, nil)
	if err != nil {
		app.errorLog.Printf("Error inserting new match: %v\n", err)
		return err
	}

	clients.mu.Lock()
	defer clients.mu.Unlock()

	if _, ok := clients.clients[playerOneID]; !ok {
		clients.clients[playerOneID] = &Client{id: playerOneID, channel: make(chan string, 1)}
	}
	clients.clients[playerOneID].channel <- fmt.Sprintf("%v,%v,%v", matchID, timeFormatInMilliseconds, incrementInMilliseconds)

	if _, ok := clients.clients[playerTwoID]; !ok {
		clients.clients[playerTwoID] = &Client{id: playerTwoID, channel: make(chan string, 1)}
	}
	clients.clients[playerTwoID].channel <- fmt.Sprintf("%v,%v,%v", matchID, timeFormatInMilliseconds, incrementInMilliseconds)

	return nil
}

func matchPlayers() {
	// Snapshot the queues so we don't hold queueMu while processing
	queueMu.Lock()
	queues := make([]*QueueData, 0, len(queueMap))
	for _, queue := range queueMap {
		queues = append(queues, queue)
	}
	queueMu.Unlock()

	for _, queue := range queues {
		queue.mu.Lock()

		// Merge waiting pool into matchmaking pool
		queue.matchmakingPool = append(queue.matchmakingPool, queue.waitingPool...)
		queue.waitingPool = queue.waitingPool[:0]

		var validMatches []*matchingScore

		// Score all player pairs
		for playerOneIdx, playerOne := range queue.matchmakingPool {
			for playerTwoIdx, playerTwo := range queue.matchmakingPool[playerOneIdx+1:] {
				playerTwoIdx += playerOneIdx + 1
				score := calculateMatchingScore(playerOne, playerOneIdx, playerTwo, playerTwoIdx)

				if score.score*2 <= playerOne.matchmakingThreshold+playerTwo.matchmakingThreshold {
					validMatches = append(validMatches, score)
				}
			}
		}

		sort.Slice(validMatches, func(i, j int) bool {
			return validMatches[i].score < validMatches[j].score
		})

		for _, score := range validMatches {
			playerOne := queue.matchmakingPool[score.playerOneIdx]
			playerTwo := queue.matchmakingPool[score.playerTwoIdx]

			if playerOne.isMatched || queue.awaitingRemoval[playerOne.playerID] {
				continue
			}
			if playerTwo.isMatched || queue.awaitingRemoval[playerTwo.playerID] {
				continue
			}

			// Unlock before createMatch to avoid holding queue.mu while locking clients.mu
			queue.mu.Unlock()
			err := createMatch(playerOne, playerTwo, queue.timeFormatInMilliseconds, queue.incrementInMilliseconds)
			queue.mu.Lock()

			if err != nil {
				app.errorLog.Println(err)
				continue
			}
			playerOne.isMatched = true
			playerTwo.isMatched = true
		}

		// Cleanup matched/removed players, increase threshold for unmatched players
		for i := len(queue.matchmakingPool) - 1; i >= 0; i-- {
			player := queue.matchmakingPool[i]
			if player.isMatched || queue.awaitingRemoval[player.playerID] {
				queue.matchmakingPool = swapRemove(queue.matchmakingPool, i)
				delete(queue.awaitingRemoval, player.playerID)
			} else {
				player.matchmakingThreshold += matchmakingThresholdIncrement
			}
		}

		queue.mu.Unlock()
	}
}

func matchmakingService() {
	app.infoLog.Printf("Starting matchmakingService")
	defer app.infoLog.Printf("Ending matchmakingService")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		matchPlayers()
	}
}
