package main

import (
	"testing"
	"time"

	"burrchess/internal/chess"
)

// minimalHub returns a MatchRoomHub with just enough state to exercise
// methods that don't need a live DB match or running goroutines.
func minimalHub(t *testing.T) *MatchRoomHub {
	t.Helper()
	return &MatchRoomHub{
		app:       newTestApp(t),
		fenFreqMap: make(map[string]int),
		players: [2]playerState{
			{timeRemaining: 5 * time.Minute},
			{timeRemaining: 5 * time.Minute},
		},
		increment: 0,
	}
}

// ---------------------------------------------------------------------------
// opponent / colorName
// ---------------------------------------------------------------------------

func TestOpponent(t *testing.T) {
	if opponent(WhitePlayer) != BlackPlayer {
		t.Errorf("opponent(WhitePlayer) = %d, want %d", opponent(WhitePlayer), BlackPlayer)
	}
	if opponent(BlackPlayer) != WhitePlayer {
		t.Errorf("opponent(BlackPlayer) = %d, want %d", opponent(BlackPlayer), WhitePlayer)
	}
}

func TestColorName(t *testing.T) {
	if colorName(WhitePlayer) != "white" {
		t.Errorf("colorName(WhitePlayer) = %q, want white", colorName(WhitePlayer))
	}
	if colorName(BlackPlayer) != "black" {
		t.Errorf("colorName(BlackPlayer) = %q, want black", colorName(BlackPlayer))
	}
}

// ---------------------------------------------------------------------------
// isOneSidedEvent
// ---------------------------------------------------------------------------

func TestIsOneSidedEvent(t *testing.T) {
	oneSided := []eventType{extraTime, resign, abort, disconnect}
	for _, e := range oneSided {
		if !isOneSidedEvent(e) {
			t.Errorf("isOneSidedEvent(%q) = false, want true", e)
		}
	}

	twoSided := []eventType{takeback, draw, rematch, decline, threefoldRepetition}
	for _, e := range twoSided {
		if isOneSidedEvent(e) {
			t.Errorf("isOneSidedEvent(%q) = true, want false", e)
		}
	}
}

// ---------------------------------------------------------------------------
// getOutcome
// ---------------------------------------------------------------------------

func TestGetOutcome_Checkmate(t *testing.T) {
	// On checkmate hub.turn is the checkmated side (turn already flipped).
	tests := []struct {
		turn playerTurn
		want chess.MatchOutcome
	}{
		{playerTurn(BlackTurn), chess.OutcomeBlackWins}, // black checkmated → black loses
		{playerTurn(WhiteTurn), chess.OutcomeWhiteWins}, // white checkmated → white loses
	}
	for _, tt := range tests {
		hub := minimalHub(t)
		hub.turn = tt.turn
		got := hub.getOutcome(chess.Checkmate)
		if got != tt.want {
			t.Errorf("turn=%d: getOutcome(Checkmate) = %v, want %v", tt.turn, got, tt.want)
		}
	}
}

func TestGetOutcome_FlagAndResign(t *testing.T) {
	hub := minimalHub(t)
	tests := []struct {
		status chess.GameOverStatusCode
		want   chess.MatchOutcome
	}{
		{chess.WhiteFlagged, chess.OutcomeBlackWins},
		{chess.BlackFlagged, chess.OutcomeWhiteWins},
		{chess.WhiteResigned, chess.OutcomeBlackWins},
		{chess.BlackResigned, chess.OutcomeWhiteWins},
		{chess.WhiteDisconnected, chess.OutcomeBlackWins},
		{chess.BlackDisconnected, chess.OutcomeWhiteWins},
	}
	for _, tt := range tests {
		got := hub.getOutcome(tt.status)
		if got != tt.want {
			t.Errorf("getOutcome(%v) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestGetOutcome_DrawStatuses(t *testing.T) {
	hub := minimalHub(t)
	drawStatuses := []chess.GameOverStatusCode{
		chess.Stalemate,
		chess.ThreefoldRepetition,
		chess.InsufficientMaterial,
		chess.Draw,
		chess.GameAborted,
	}
	for _, s := range drawStatuses {
		got := hub.getOutcome(s)
		if got != chess.OutcomeDraw {
			t.Errorf("getOutcome(%v) = %v, want OutcomeDraw", s, got)
		}
	}
}

// ---------------------------------------------------------------------------
// updateTimeRemaining
// ---------------------------------------------------------------------------

func TestUpdateTimeRemaining_TimerInactive(t *testing.T) {
	hub := minimalHub(t)
	hub.isTimerActive = false
	hub.turn = playerTurn(WhiteTurn)
	hub.players[WhitePlayer].timeRemaining = 5 * time.Minute
	hub.timeOfLastMove = time.Now().Add(-10 * time.Second)

	hub.updateTimeRemaining()

	// Timer was inactive — clock must not change.
	if hub.players[WhitePlayer].timeRemaining != 5*time.Minute {
		t.Errorf("timeRemaining changed when timer inactive: got %v", hub.players[WhitePlayer].timeRemaining)
	}
}

func TestUpdateTimeRemaining_DeductsMoverClock(t *testing.T) {
	hub := minimalHub(t)
	hub.isTimerActive = true
	hub.turn = playerTurn(WhiteTurn)
	hub.players[WhitePlayer].timeRemaining = 5 * time.Minute
	hub.players[BlackPlayer].timeRemaining = 5 * time.Minute
	hub.timeOfLastMove = time.Now().Add(-10 * time.Second)

	hub.updateTimeRemaining()

	// White's clock should have decreased by roughly 10s.
	remaining := hub.players[WhitePlayer].timeRemaining
	if remaining >= 5*time.Minute {
		t.Error("white clock should have decreased")
	}
	if remaining < 4*time.Minute+45*time.Second {
		t.Errorf("white clock decreased too much: %v", remaining)
	}
	// Black's clock must be untouched.
	if hub.players[BlackPlayer].timeRemaining != 5*time.Minute {
		t.Error("black clock should not change when white is the mover")
	}
}

func TestUpdateTimeRemaining_IncrementsApplied(t *testing.T) {
	hub := minimalHub(t)
	hub.isTimerActive = true
	hub.turn = playerTurn(WhiteTurn)
	hub.increment = 5 * time.Second
	hub.players[WhitePlayer].timeRemaining = 1 * time.Minute
	hub.timeOfLastMove = time.Now().Add(-2 * time.Second) // moved 2s ago

	hub.updateTimeRemaining()

	// Should lose ~2s but gain 5s → net +3s from starting minute.
	remaining := hub.players[WhitePlayer].timeRemaining
	if remaining <= 1*time.Minute {
		t.Errorf("increment not applied: remaining = %v, want > 1 min", remaining)
	}
}

func TestUpdateTimeRemaining_FloorAtZero(t *testing.T) {
	hub := minimalHub(t)
	hub.isTimerActive = true
	hub.turn = playerTurn(WhiteTurn)
	hub.players[WhitePlayer].timeRemaining = 1 * time.Millisecond
	hub.timeOfLastMove = time.Now().Add(-10 * time.Second) // massive overshoot

	hub.updateTimeRemaining()

	if hub.players[WhitePlayer].timeRemaining != 0 {
		t.Errorf("time should floor at 0, got %v", hub.players[WhitePlayer].timeRemaining)
	}
}

// ---------------------------------------------------------------------------
// changeTurn
// ---------------------------------------------------------------------------

func TestChangeTurn_TogglesTurn(t *testing.T) {
	hub := minimalHub(t)
	hub.turn = playerTurn(WhiteTurn)
	hub.isTimerActive = true

	hub.changeTurn()

	if hub.turn != playerTurn(BlackTurn) {
		t.Errorf("turn after changeTurn = %d, want BlackTurn", hub.turn)
	}

	hub.changeTurn()

	if hub.turn != playerTurn(WhiteTurn) {
		t.Errorf("turn after second changeTurn = %d, want WhiteTurn", hub.turn)
	}
}

func TestChangeTurn_ActivatesTimerOnBlacksFirstMove(t *testing.T) {
	// Timer activates after Black's first move: when turn is Black and
	// isTimerActive is false, changeTurn (called after Black moves) should
	// activate the timer.
	hub := minimalHub(t)
	hub.turn = playerTurn(BlackTurn) // Black just moved; changeTurn is called
	hub.isTimerActive = false

	hub.changeTurn()

	if !hub.isTimerActive {
		t.Error("timer should activate after black's first move")
	}
	if hub.flagTimerHandle == nil {
		t.Error("flagTimerHandle should be set after timer activation")
	}
	hub.flagTimerHandle.Stop()
}
