package main

import (
	"encoding/json"
	"strings"
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

// ---------------------------------------------------------------------------
// Helpers for message-based tests
// ---------------------------------------------------------------------------

const startingFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// buildMsg prepends a sender byte to a JSON-marshalled value, mirroring what
// readPump does before writing to hub.broadcast.
func buildMsg(t *testing.T, sender byte, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("buildMsg: marshal: %v", err)
	}
	return append([]byte{sender}, b...)
}

// hubForMoveTests returns a hub set up at the start of a game, ready for
// updateGameStateAfterMove calls. The timer is inactive so no clock arithmetic
// runs, and the DB task queue writes fail silently (matchID=0, no row).
func hubForMoveTests(t *testing.T) *MatchRoomHub {
	t.Helper()
	hub := minimalHub(t)
	hub.currentFEN = startingFEN
	hub.isTimerActive = false
	hub.turn = playerTurn(WhiteTurn)
	return hub
}

// ---------------------------------------------------------------------------
// getMessageType
// ---------------------------------------------------------------------------

func TestGetMessageType_PostMove_FromWhite(t *testing.T) {
	hub := minimalHub(t)
	msg := buildMsg(t, WhitePlayer, postMoveResponse{MessageType: postMove, Body: postMoveBody{Piece: 52, Move: 36}})
	if got := hub.getMessageType(msg); got != postMove {
		t.Errorf("got %q, want postMove", got)
	}
}

func TestGetMessageType_PostMove_FromBlack(t *testing.T) {
	hub := minimalHub(t)
	msg := buildMsg(t, BlackPlayer, postMoveResponse{MessageType: postMove, Body: postMoveBody{Piece: 52, Move: 36}})
	if got := hub.getMessageType(msg); got != postMove {
		t.Errorf("got %q, want postMove", got)
	}
}

func TestGetMessageType_PostMove_FromSpectator_IsUnknown(t *testing.T) {
	// Spectators must not be able to post moves — they are silently rejected.
	hub := minimalHub(t)
	msg := buildMsg(t, Spectator, postMoveResponse{MessageType: postMove, Body: postMoveBody{Piece: 52, Move: 36}})
	if got := hub.getMessageType(msg); got != unknown {
		t.Errorf("got %q, want unknown (spectator blocked from postMove)", got)
	}
}

func TestGetMessageType_PlayerEvent_FromPlayer(t *testing.T) {
	hub := minimalHub(t)
	msg := buildMsg(t, WhitePlayer, playerEventResponse{MessageType: playerEvent, Body: playerEventBody{EventType: resign}})
	if got := hub.getMessageType(msg); got != playerEvent {
		t.Errorf("got %q, want playerEvent", got)
	}
}

func TestGetMessageType_PlayerEvent_FromSpectator_IsUnknown(t *testing.T) {
	hub := minimalHub(t)
	msg := buildMsg(t, Spectator, playerEventResponse{MessageType: playerEvent, Body: playerEventBody{EventType: resign}})
	if got := hub.getMessageType(msg); got != unknown {
		t.Errorf("got %q, want unknown (spectator blocked from playerEvent)", got)
	}
}

func TestGetMessageType_GetMoves_FromSpectator(t *testing.T) {
	// Spectators are allowed to request legal moves (read-only).
	hub := minimalHub(t)
	msg := buildMsg(t, Spectator, getMovesRequest{MessageType: getMoves, Body: getMovesBody{Piece: 52}})
	if got := hub.getMessageType(msg); got != getMoves {
		t.Errorf("got %q, want getMoves", got)
	}
}

func TestGetMessageType_MalformedJSON_IsUnknown(t *testing.T) {
	hub := minimalHub(t)
	msg := append([]byte{WhitePlayer}, []byte("not valid json")...)
	if got := hub.getMessageType(msg); got != unknown {
		t.Errorf("got %q, want unknown for malformed JSON", got)
	}
}

func TestGetMessageType_UnknownType_IsUnknown(t *testing.T) {
	hub := minimalHub(t)
	msg := buildMsg(t, WhitePlayer, map[string]string{"messageType": "someFutureType"})
	if got := hub.getMessageType(msg); got != unknown {
		t.Errorf("got %q, want unknown for unrecognised message type", got)
	}
}

// ---------------------------------------------------------------------------
// updateGameStateAfterMove
// ---------------------------------------------------------------------------

func TestUpdateGameStateAfterMove_ValidMove(t *testing.T) {
	hub := hubForMoveTests(t)

	// e2→e4: piece=52, move=36 (white pawn opening)
	msg := buildMsg(t, WhitePlayer, postMoveResponse{
		MessageType: postMove,
		Body:        postMoveBody{Piece: 52, Move: 36},
	})

	err := hub.updateGameStateAfterMove(msg)
	if err != nil {
		t.Fatalf("updateGameStateAfterMove: %v", err)
	}

	if hub.currentFEN == startingFEN {
		t.Error("FEN should have changed after a valid move")
	}
	// Turn flips: white moved, now black's turn.
	if hub.turn != playerTurn(BlackTurn) {
		t.Errorf("turn = %d, want BlackTurn after white moves", hub.turn)
	}
	// History grows by one entry.
	if len(hub.moveHistory) != 1 {
		t.Errorf("moveHistory length = %d, want 1", len(hub.moveHistory))
	}
	// The new history entry should record the correct squares.
	if hub.moveHistory[0].LastMove != [2]int{52, 36} {
		t.Errorf("LastMove = %v, want [52 36]", hub.moveHistory[0].LastMove)
	}
}

func TestUpdateGameStateAfterMove_PieceOutOfBounds(t *testing.T) {
	hub := hubForMoveTests(t)
	msg := buildMsg(t, WhitePlayer, postMoveResponse{
		MessageType: postMove,
		Body:        postMoveBody{Piece: 100, Move: 36}, // piece index > 63
	})

	if err := hub.updateGameStateAfterMove(msg); err == nil {
		t.Error("expected error for out-of-bounds piece index, got nil")
	}
	// FEN and history must be unchanged.
	if hub.currentFEN != startingFEN {
		t.Error("FEN should not change on rejected move")
	}
}

func TestUpdateGameStateAfterMove_IllegalMove(t *testing.T) {
	hub := hubForMoveTests(t)
	// Attempt to move the e2 pawn to e1 (own king's square) — illegal.
	msg := buildMsg(t, WhitePlayer, postMoveResponse{
		MessageType: postMove,
		Body:        postMoveBody{Piece: 52, Move: 60},
	})

	if err := hub.updateGameStateAfterMove(msg); err == nil {
		t.Error("expected error for illegal move, got nil")
	}
	if hub.currentFEN != startingFEN {
		t.Error("FEN should not change on rejected move")
	}
}

func TestUpdateGameStateAfterMove_ThreefoldRepetition(t *testing.T) {
	// Set the FEN frequency map so the position reached by e2→e4 already
	// appears twice. The move should then flip isThreefoldRepetition to true.
	hub := hubForMoveTests(t)

	// Apply e2→e4 once to find out what FEN it produces.
	msg := buildMsg(t, WhitePlayer, postMoveResponse{
		MessageType: postMove,
		Body:        postMoveBody{Piece: 52, Move: 36},
	})
	if err := hub.updateGameStateAfterMove(msg); err != nil {
		t.Fatalf("first move: %v", err)
	}
	afterE4FEN := hub.currentFEN

	// Reset hub to starting position and pre-seed the frequency map with two
	// prior occurrences of that position (position + castling + en-passant fields,
	// no halfmove/fullmove clocks).
	hub2 := hubForMoveTests(t)
	splitKey := splitFENKey(afterE4FEN)
	hub2.fenFreqMap[splitKey] = 2 // already seen twice

	msg2 := buildMsg(t, WhitePlayer, postMoveResponse{
		MessageType: postMove,
		Body:        postMoveBody{Piece: 52, Move: 36},
	})
	if err := hub2.updateGameStateAfterMove(msg2); err != nil {
		t.Fatalf("seeded move: %v", err)
	}

	if !hub2.isThreefoldRepetition {
		t.Error("isThreefoldRepetition should be true after third occurrence")
	}
}

// splitFENKey returns the first four space-separated fields of a FEN string —
// the part used for threefold-repetition tracking (position, active colour,
// castling, en-passant; no halfmove/fullmove clocks).
func splitFENKey(fen string) string {
	parts := strings.SplitN(fen, " ", 5)
	if len(parts) < 4 {
		return fen
	}
	return strings.Join(parts[:4], " ")
}
