package main

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// Test setup helpers
// ---------------------------------------------------------------------------

// resetMatchmaking resets all global matchmaking state so tests are isolated.
func resetMatchmaking(t *testing.T) {
	t.Helper()

	app = newTestApp(t)

	queueMu.Lock()
	queueMap = make(map[queueKey]*QueueData)
	queueMu.Unlock()

	clients.mu.Lock()
	clients.clients = make(map[int64]*Client)
	clients.mu.Unlock()
}

// makeQueueWithPlayers builds a QueueData whose matchmakingPool is pre-populated
// with the given players (bypassing addPlayerToWaitingPool so tests can set
// arbitrary ELOs without needing real DB rows).
func makeQueueWithPlayers(timeFormat, increment int64, players ...*playerMatchmakingData) *QueueData {
	q := &QueueData{
		awaitingRemoval:          make(map[int64]bool),
		timeFormatInMilliseconds: timeFormat,
		incrementInMilliseconds:  increment,
	}
	for _, p := range players {
		q.matchmakingPool = append(q.matchmakingPool, p)
		q.awaitingRemoval[p.playerID] = false
	}
	return q
}

// registerQueue inserts a QueueData directly into the global queueMap.
func registerQueue(q *QueueData) {
	key := queueKey{q.timeFormatInMilliseconds, q.incrementInMilliseconds}
	queueMu.Lock()
	queueMap[key] = q
	queueMu.Unlock()
}

// player is a convenience constructor for playerMatchmakingData.
func player(id, elo int64) *playerMatchmakingData {
	return &playerMatchmakingData{
		playerID:             id,
		elo:                  elo,
		matchmakingThreshold: defaultMatchmakingThreshold,
	}
}

// poolSize returns the current matchmakingPool length for a given time format.
func poolSize(timeFormat, increment int64) int {
	key := queueKey{timeFormat, increment}
	queueMu.Lock()
	q := queueMap[key]
	queueMu.Unlock()
	if q == nil {
		return 0
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.matchmakingPool)
}

// ---------------------------------------------------------------------------
// Pure functions
// ---------------------------------------------------------------------------

func TestAbs(t *testing.T) {
	tests := []struct{ in, want int64 }{
		{0, 0},
		{5, 5},
		{-5, 5},
		{-1, 1},
	}
	for _, tt := range tests {
		if got := abs(tt.in); got != tt.want {
			t.Errorf("abs(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestSwapRemove(t *testing.T) {
	t.Run("remove middle element", func(t *testing.T) {
		s := []int{1, 2, 3, 4}
		s = swapRemove(s, 1)
		if len(s) != 3 {
			t.Fatalf("len = %d, want 3", len(s))
		}
		// 2 should be gone; order may change (swap with last)
		for _, v := range s {
			if v == 2 {
				t.Error("element 2 should have been removed")
			}
		}
	})

	t.Run("remove last element", func(t *testing.T) {
		s := []int{1, 2, 3}
		s = swapRemove(s, 2)
		if len(s) != 2 {
			t.Fatalf("len = %d, want 2", len(s))
		}
	})

	t.Run("remove only element", func(t *testing.T) {
		s := []int{42}
		s = swapRemove(s, 0)
		if len(s) != 0 {
			t.Fatalf("len = %d, want 0", len(s))
		}
	})
}

func TestCalculateMatchingScore(t *testing.T) {
	p1 := player(1, 1500)
	p2 := player(2, 1700)

	score := calculateMatchingScore(p1, 0, p2, 1)

	if score.playerOneID != 1 || score.playerTwoID != 2 {
		t.Errorf("player IDs: got (%d,%d), want (1,2)", score.playerOneID, score.playerTwoID)
	}
	if score.score != 200 {
		t.Errorf("score = %d, want 200 (|1500-1700|)", score.score)
	}
	if score.playerOneIdx != 0 || score.playerTwoIdx != 1 {
		t.Errorf("indices: got (%d,%d), want (0,1)", score.playerOneIdx, score.playerTwoIdx)
	}
}

func TestCalculateMatchingScore_SameElo(t *testing.T) {
	score := calculateMatchingScore(player(1, 1500), 0, player(2, 1500), 1)
	if score.score != 0 {
		t.Errorf("score = %d, want 0 for equal ELOs", score.score)
	}
}

func TestStartingMatchHistory(t *testing.T) {
	const timeFormat = 5 * 60_000

	data, err := startingMatchHistory(timeFormat)
	if err != nil {
		t.Fatalf("startingMatchHistory: %v", err)
	}

	var history []MatchStateHistory
	if err := json.Unmarshal(data, &history); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("len = %d, want 1", len(history))
	}
	h := history[0]
	if h.FEN != "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" {
		t.Errorf("FEN = %q, want starting position", h.FEN)
	}
	if h.WhitePlayerTimeRemainingMilliseconds != timeFormat {
		t.Errorf("white time = %d, want %d", h.WhitePlayerTimeRemainingMilliseconds, timeFormat)
	}
	if h.BlackPlayerTimeRemainingMilliseconds != timeFormat {
		t.Errorf("black time = %d, want %d", h.BlackPlayerTimeRemainingMilliseconds, timeFormat)
	}
}

// ---------------------------------------------------------------------------
// Queue management
// ---------------------------------------------------------------------------

func TestAddPlayerToWaitingPool_AddsPlayer(t *testing.T) {
	resetMatchmaking(t)

	addPlayerToWaitingPool(100, 5*60_000, 0)

	key := queueKey{5 * 60_000, 0}
	queueMu.Lock()
	q := queueMap[key]
	queueMu.Unlock()

	if q == nil {
		t.Fatal("queue not created")
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.waitingPool) != 1 {
		t.Fatalf("waitingPool len = %d, want 1", len(q.waitingPool))
	}
	if q.waitingPool[0].playerID != 100 {
		t.Errorf("playerID = %d, want 100", q.waitingPool[0].playerID)
	}
	if q.waitingPool[0].matchmakingThreshold != defaultMatchmakingThreshold {
		t.Errorf("threshold = %d, want %d", q.waitingPool[0].matchmakingThreshold, defaultMatchmakingThreshold)
	}
	if pending, ok := q.awaitingRemoval[100]; !ok || pending {
		t.Errorf("awaitingRemoval[100] = %v (ok=%v), want false", pending, ok)
	}
}

func TestAddPlayerToWaitingPool_DefaultElo(t *testing.T) {
	// A player not in the DB should default to ELO 1500.
	resetMatchmaking(t)

	const unknownPlayerID = 99999
	addPlayerToWaitingPool(unknownPlayerID, 5*60_000, 0)

	key := queueKey{5 * 60_000, 0}
	queueMu.Lock()
	q := queueMap[key]
	queueMu.Unlock()

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.waitingPool[0].elo != 1500 {
		t.Errorf("elo = %d, want 1500 for unknown player", q.waitingPool[0].elo)
	}
}

func TestAddPlayerToWaitingPool_CancelsRemoval(t *testing.T) {
	// Re-joining while removal is pending should cancel the removal.
	resetMatchmaking(t)

	addPlayerToWaitingPool(100, 5*60_000, 0)
	removePlayerFromWaitingPool(100, 5*60_000, 0) // mark for removal

	key := queueKey{5 * 60_000, 0}
	queueMu.Lock()
	q := queueMap[key]
	queueMu.Unlock()

	q.mu.Lock()
	if !q.awaitingRemoval[100] {
		q.mu.Unlock()
		t.Fatal("precondition: player should be pending removal")
	}
	q.mu.Unlock()

	addPlayerToWaitingPool(100, 5*60_000, 0) // re-join

	q.mu.Lock()
	defer q.mu.Unlock()
	if q.awaitingRemoval[100] {
		t.Error("removal should have been cancelled on re-join")
	}
}

func TestRemovePlayerFromWaitingPool_MarksForRemoval(t *testing.T) {
	resetMatchmaking(t)

	addPlayerToWaitingPool(100, 5*60_000, 0)
	removePlayerFromWaitingPool(100, 5*60_000, 0)

	key := queueKey{5 * 60_000, 0}
	queueMu.Lock()
	q := queueMap[key]
	queueMu.Unlock()

	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.awaitingRemoval[100] {
		t.Error("awaitingRemoval[100] should be true after remove call")
	}
}

func TestRemovePlayerFromWaitingPool_UnknownPlayerIgnored(t *testing.T) {
	// Calling remove for a player who never joined should not panic or error.
	resetMatchmaking(t)

	addPlayerToWaitingPool(100, 5*60_000, 0) // create the queue first
	removePlayerFromWaitingPool(999, 5*60_000, 0)

	key := queueKey{5 * 60_000, 0}
	queueMu.Lock()
	q := queueMap[key]
	queueMu.Unlock()

	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.awaitingRemoval[999]; ok {
		t.Error("unknown player 999 should not appear in awaitingRemoval")
	}
}

// ---------------------------------------------------------------------------
// matchPlayers — matching logic
// ---------------------------------------------------------------------------

func TestMatchPlayers_MatchesTwoEqualEloPlayers(t *testing.T) {
	resetMatchmaking(t)

	// Both players default to ELO 1500 (not in DB). Score = 0, well within threshold.
	addPlayerToWaitingPool(100, 5*60_000, 0)
	addPlayerToWaitingPool(200, 5*60_000, 0)

	matchPlayers()

	// Both players must now be in a live match.
	for _, id := range []int64{100, 200} {
		in, err := app.liveMatches.IsPlayerInMatch(id)
		if err != nil {
			t.Fatalf("IsPlayerInMatch(%d): %v", id, err)
		}
		if !in {
			t.Errorf("player %d should be in a match after matchmaking", id)
		}
	}
}

func TestMatchPlayers_MatchedPlayersRemovedFromPool(t *testing.T) {
	resetMatchmaking(t)

	addPlayerToWaitingPool(100, 5*60_000, 0)
	addPlayerToWaitingPool(200, 5*60_000, 0)

	matchPlayers()

	if n := poolSize(5*60_000, 0); n != 0 {
		t.Errorf("matchmakingPool size = %d, want 0 after both players matched", n)
	}
}

func TestMatchPlayers_EloTooFarApart_NoMatch(t *testing.T) {
	// ELO diff = 600 > threshold sum 800 / 2 = 400 — no match on first round.
	resetMatchmaking(t)

	q := makeQueueWithPlayers(5*60_000, 0,
		player(100, 1000),
		player(200, 1600),
	)
	registerQueue(q)

	matchPlayers()

	for _, id := range []int64{100, 200} {
		in, err := app.liveMatches.IsPlayerInMatch(id)
		if err != nil {
			t.Fatalf("IsPlayerInMatch(%d): %v", id, err)
		}
		if in {
			t.Errorf("player %d should NOT be matched yet (ELO gap too large)", id)
		}
	}
}

func TestMatchPlayers_ThresholdGrowsUntilMatch(t *testing.T) {
	// ELO diff = 600.
	// Score condition: 600*2 <= threshold1 + threshold2
	// With starting threshold 400 and increment 50:
	//   round 0: 400+400=800  < 1200 → no match, threshold → 450
	//   round 1: 450+450=900  < 1200 → no match, threshold → 500
	//   round 2: 500+500=1000 < 1200 → no match, threshold → 550
	//   round 3: 550+550=1100 < 1200 → no match, threshold → 600
	//   round 4: 600+600=1200 = 1200 → MATCH
	resetMatchmaking(t)

	q := makeQueueWithPlayers(5*60_000, 0,
		player(100, 1000),
		player(200, 1600),
	)
	registerQueue(q)

	for range 4 {
		matchPlayers()
		// Should still not be matched
		in, _ := app.liveMatches.IsPlayerInMatch(100)
		if in {
			t.Fatal("matched too early — threshold growth logic may be wrong")
		}
	}

	matchPlayers() // 5th call — should match now

	in, err := app.liveMatches.IsPlayerInMatch(100)
	if err != nil {
		t.Fatalf("IsPlayerInMatch: %v", err)
	}
	if !in {
		t.Error("players should have matched once threshold grew large enough")
	}
}

func TestMatchPlayers_PendingRemovalPreventsMatch(t *testing.T) {
	resetMatchmaking(t)

	addPlayerToWaitingPool(100, 5*60_000, 0)
	addPlayerToWaitingPool(200, 5*60_000, 0)
	removePlayerFromWaitingPool(100, 5*60_000, 0) // player 100 leaves before tick

	matchPlayers()

	// Neither player should be in a match (100 was leaving; 200 has no partner).
	for _, id := range []int64{100, 200} {
		in, err := app.liveMatches.IsPlayerInMatch(id)
		if err != nil {
			t.Fatalf("IsPlayerInMatch(%d): %v", id, err)
		}
		if in {
			t.Errorf("player %d should NOT be in a match (100 requested removal)", id)
		}
	}
}

func TestMatchPlayers_BestPairMatchedFirst(t *testing.T) {
	// Three players: 100@1500, 200@1510, 300@2000.
	// Scores: (100,200)=10, (100,300)=500, (200,300)=490.
	// Best pair is (100,200); player 300 should remain unmatched.
	resetMatchmaking(t)

	q := makeQueueWithPlayers(5*60_000, 0,
		player(100, 1500),
		player(200, 1510),
		player(300, 2000),
	)
	registerQueue(q)

	matchPlayers()

	in100, _ := app.liveMatches.IsPlayerInMatch(100)
	in200, _ := app.liveMatches.IsPlayerInMatch(200)
	in300, _ := app.liveMatches.IsPlayerInMatch(300)

	if !in100 || !in200 {
		t.Error("players 100 and 200 (closest ELO) should have been matched")
	}
	if in300 {
		t.Error("player 300 (far ELO) should remain unmatched")
	}
}

func TestMatchPlayers_WaitingPoolMergedIntoMatchmaking(t *testing.T) {
	// Players added to waitingPool must be moved to matchmakingPool on the next tick.
	resetMatchmaking(t)

	addPlayerToWaitingPool(100, 5*60_000, 0)

	key := queueKey{5 * 60_000, 0}
	queueMu.Lock()
	q := queueMap[key]
	queueMu.Unlock()

	q.mu.Lock()
	if len(q.waitingPool) != 1 {
		q.mu.Unlock()
		t.Fatal("precondition: player should be in waitingPool")
	}
	q.mu.Unlock()

	matchPlayers() // merge happens here (no match since only one player)

	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.waitingPool) != 0 {
		t.Error("waitingPool should be empty after matchPlayers drains it")
	}
	if len(q.matchmakingPool) != 1 {
		t.Errorf("matchmakingPool len = %d, want 1", len(q.matchmakingPool))
	}
}

func TestMatchPlayers_UnmatchedPlayerRemainsInPool(t *testing.T) {
	// A lone player should remain in the pool across ticks.
	resetMatchmaking(t)

	addPlayerToWaitingPool(100, 5*60_000, 0)

	matchPlayers()
	matchPlayers()

	if n := poolSize(5*60_000, 0); n != 1 {
		t.Errorf("pool size = %d, want 1 (lone player should stay)", n)
	}
}

func TestMatchPlayers_SeparateQueuesIndependent(t *testing.T) {
	// Players in different time formats must not match each other.
	resetMatchmaking(t)

	addPlayerToWaitingPool(100, 3*60_000, 0) // 3-min blitz
	addPlayerToWaitingPool(200, 5*60_000, 0) // 5-min blitz

	matchPlayers()

	for _, id := range []int64{100, 200} {
		in, err := app.liveMatches.IsPlayerInMatch(id)
		if err != nil {
			t.Fatalf("IsPlayerInMatch(%d): %v", id, err)
		}
		if in {
			t.Errorf("player %d should not be matched across different time formats", id)
		}
	}
}

// ---------------------------------------------------------------------------
// notifyMatchFound
// ---------------------------------------------------------------------------

func TestNotifyMatchFound_SendsToClients(t *testing.T) {
	// Both players should receive a message on their channels.
	resetMatchmaking(t)

	notifyMatchFound(100, 200, 42, 5*60_000, 0)

	clients.mu.Lock()
	defer clients.mu.Unlock()

	for _, id := range []int64{100, 200} {
		c, ok := clients.clients[id]
		if !ok {
			t.Fatalf("no client entry for player %d", id)
		}
		select {
		case msg := <-c.channel:
			if msg != "42,300000,0" {
				t.Errorf("player %d message = %q, want \"42,300000,0\"", id, msg)
			}
		default:
			t.Errorf("no message queued for player %d", id)
		}
	}
}

func TestNotifyMatchFound_DrainsStalePendingMessage(t *testing.T) {
	// If a client channel already has a stale message, it should be drained
	// before the new one is written, avoiding a block.
	resetMatchmaking(t)

	clients.mu.Lock()
	clients.clients[100] = &Client{id: 100, channel: make(chan string, 1)}
	clients.clients[100].channel <- "stale"
	clients.mu.Unlock()

	// Should not block even though channel was full.
	notifyMatchFound(100, 200, 99, 5*60_000, 0)

	clients.mu.Lock()
	defer clients.mu.Unlock()

	msg := <-clients.clients[100].channel
	if msg == "stale" {
		t.Error("stale message was not drained before new notification")
	}
}
