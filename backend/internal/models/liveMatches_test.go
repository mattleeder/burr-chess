package models

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"burrchess/internal/chess"
)

const (
	testTimeFormat  int64 = 5 * 60_000 // 5-minute blitz
	testIncrement   int64 = 0
	testWhitePlayer int64 = 100
	testBlackPlayer int64 = 200
)

func insertTestMatch(t *testing.T, m *LiveMatchModel) int64 {
	t.Helper()
	id, err := m.InsertNew(InsertNewParams{
		PlayerOneID:              testWhitePlayer,
		PlayerTwoID:              testBlackPlayer,
		PlayerOneIsWhite:         true,
		TimeFormatInMilliseconds: testTimeFormat,
		IncrementInMilliseconds:  testIncrement,
		GameHistory:              []byte("[]"),
		AverageElo:               1500,
		WhitePlayerElo:           1500,
		BlackPlayerElo:           1500,
	})
	if err != nil {
		t.Fatalf("insertTestMatch: %v", err)
	}
	return id
}

func TestLiveMatchModel_InsertNew(t *testing.T) {
	m := &LiveMatchModel{DB: newTestDB(t)}

	id := insertTestMatch(t, m)
	if id <= 0 {
		t.Errorf("expected positive matchID, got %d", id)
	}
}

func TestLiveMatchModel_InsertNew_PlayerOneIsWhite(t *testing.T) {
	m := &LiveMatchModel{DB: newTestDB(t)}

	// PlayerOneID=100 should become black when PlayerOneIsWhite=false
	id, err := m.InsertNew(InsertNewParams{
		PlayerOneID:              100,
		PlayerTwoID:              200,
		PlayerOneIsWhite:         false,
		TimeFormatInMilliseconds: testTimeFormat,
		GameHistory:              []byte("[]"),
		AverageElo:               1500,
		WhitePlayerElo:           1500,
		BlackPlayerElo:           1500,
	})
	if err != nil {
		t.Fatalf("InsertNew: %v", err)
	}

	match, err := m.GetFromMatchID(id)
	if err != nil {
		t.Fatalf("GetFromMatchID: %v", err)
	}
	if match.WhitePlayerID != 200 || match.BlackPlayerID != 100 {
		t.Errorf("got white=%d black=%d, want white=200 black=100", match.WhitePlayerID, match.BlackPlayerID)
	}
}

func TestLiveMatchModel_GetFromMatchID(t *testing.T) {
	m := &LiveMatchModel{DB: newTestDB(t)}
	id := insertTestMatch(t, m)

	match, err := m.GetFromMatchID(id)
	if err != nil {
		t.Fatalf("GetFromMatchID: %v", err)
	}

	if match.MatchID != id {
		t.Errorf("MatchID = %d, want %d", match.MatchID, id)
	}
	if match.WhitePlayerID != testWhitePlayer {
		t.Errorf("WhitePlayerID = %d, want %d", match.WhitePlayerID, testWhitePlayer)
	}
	if match.BlackPlayerID != testBlackPlayer {
		t.Errorf("BlackPlayerID = %d, want %d", match.BlackPlayerID, testBlackPlayer)
	}
	if match.TimeFormatInMilliseconds != testTimeFormat {
		t.Errorf("TimeFormat = %d, want %d", match.TimeFormatInMilliseconds, testTimeFormat)
	}
	if match.WhitePlayerTimeRemainingMilliseconds != testTimeFormat {
		t.Errorf("WhiteTimeRemaining = %d, want %d (full time format)", match.WhitePlayerTimeRemainingMilliseconds, testTimeFormat)
	}
	// Anonymous players have no username row, so usernames should be null
	if match.WhitePlayerUsername.Valid {
		t.Errorf("expected null white username for anonymous player, got %q", match.WhitePlayerUsername.String)
	}
}

func TestLiveMatchModel_GetFromMatchID_NotFound(t *testing.T) {
	m := &LiveMatchModel{DB: newTestDB(t)}

	_, err := m.GetFromMatchID(9999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestLiveMatchModel_GetFromMatchID_WithUsernames(t *testing.T) {
	db := newTestDB(t)
	liveModel := &LiveMatchModel{DB: db}
	userModel := &UserModel{DB: db, BcryptCost: bcrypt.MinCost}

	whiteID := insertTestUser(t, userModel, "white_player", "pass")
	blackID := insertTestUser(t, userModel, "black_player", "pass")

	matchID, err := liveModel.InsertNew(InsertNewParams{
		PlayerOneID:              whiteID,
		PlayerTwoID:              blackID,
		PlayerOneIsWhite:         true,
		TimeFormatInMilliseconds: testTimeFormat,
		GameHistory:              []byte("[]"),
		AverageElo:               1500,
		WhitePlayerElo:           1500,
		BlackPlayerElo:           1500,
	})
	if err != nil {
		t.Fatalf("InsertNew: %v", err)
	}

	match, err := liveModel.GetFromMatchID(matchID)
	if err != nil {
		t.Fatalf("GetFromMatchID: %v", err)
	}

	if !match.WhitePlayerUsername.Valid || match.WhitePlayerUsername.String != "white_player" {
		t.Errorf("WhitePlayerUsername = %v, want white_player", match.WhitePlayerUsername)
	}
	if !match.BlackPlayerUsername.Valid || match.BlackPlayerUsername.String != "black_player" {
		t.Errorf("BlackPlayerUsername = %v, want black_player", match.BlackPlayerUsername)
	}
}

func TestLiveMatchModel_UpdateLiveMatch(t *testing.T) {
	m := &LiveMatchModel{DB: newTestDB(t)}
	id := insertTestMatch(t, m)

	newFEN := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	err := m.UpdateLiveMatch(UpdateMatchParams{
		MatchID:                              id,
		NewFEN:                               newFEN,
		LastMovePiece:                        5,
		LastMoveMove:                         36,
		WhitePlayerTimeRemainingMilliseconds: 290_000,
		BlackPlayerTimeRemainingMilliseconds: 300_000,
		MatchStateHistoryJSON:                []byte("[{}]"),
		TimeOfLastMove:                       time.Now(),
	})
	if err != nil {
		t.Fatalf("UpdateLiveMatch: %v", err)
	}

	match, err := m.GetFromMatchID(id)
	if err != nil {
		t.Fatalf("GetFromMatchID after update: %v", err)
	}
	if match.CurrentFEN != newFEN {
		t.Errorf("CurrentFEN = %q, want %q", match.CurrentFEN, newFEN)
	}
	if !match.LastMovePiece.Valid || match.LastMovePiece.Int64 != 5 {
		t.Errorf("LastMovePiece = %v, want 5", match.LastMovePiece)
	}
	if !match.LastMoveMove.Valid || match.LastMoveMove.Int64 != 36 {
		t.Errorf("LastMoveMove = %v, want 36", match.LastMoveMove)
	}
	if match.WhitePlayerTimeRemainingMilliseconds != 290_000 {
		t.Errorf("WhiteTimeRemaining = %d, want 290000", match.WhitePlayerTimeRemainingMilliseconds)
	}
	if match.BlackPlayerTimeRemainingMilliseconds != 300_000 {
		t.Errorf("BlackTimeRemaining = %d, want 300000", match.BlackPlayerTimeRemainingMilliseconds)
	}
}

func TestLiveMatchModel_DeleteMatch(t *testing.T) {
	m := &LiveMatchModel{DB: newTestDB(t)}
	id := insertTestMatch(t, m)

	if err := m.DeleteMatch(id); err != nil {
		t.Fatalf("DeleteMatch: %v", err)
	}

	_, err := m.GetFromMatchID(id)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestLiveMatchModel_IsPlayerInMatch(t *testing.T) {
	m := &LiveMatchModel{DB: newTestDB(t)}
	insertTestMatch(t, m)

	t.Run("white player in match", func(t *testing.T) {
		in, err := m.IsPlayerInMatch(testWhitePlayer)
		if err != nil {
			t.Fatalf("IsPlayerInMatch: %v", err)
		}
		if !in {
			t.Errorf("expected player %d to be in a match", testWhitePlayer)
		}
	})

	t.Run("black player in match", func(t *testing.T) {
		in, err := m.IsPlayerInMatch(testBlackPlayer)
		if err != nil {
			t.Fatalf("IsPlayerInMatch: %v", err)
		}
		if !in {
			t.Errorf("expected player %d to be in a match", testBlackPlayer)
		}
	})

	t.Run("unrelated player not in match", func(t *testing.T) {
		in, err := m.IsPlayerInMatch(9999)
		if err != nil {
			t.Fatalf("IsPlayerInMatch: %v", err)
		}
		if in {
			t.Error("expected player 9999 not to be in any match")
		}
	})
}

func TestLiveMatchModel_GetHighestEloMatch(t *testing.T) {
	m := &LiveMatchModel{DB: newTestDB(t)}

	_, err := m.InsertNew(InsertNewParams{
		PlayerOneID: 100, PlayerTwoID: 200, PlayerOneIsWhite: true,
		TimeFormatInMilliseconds: testTimeFormat,
		GameHistory:              []byte("[]"),
		AverageElo:               1200,
		WhitePlayerElo:           1200, BlackPlayerElo: 1200,
	})
	if err != nil {
		t.Fatalf("InsertNew (low elo): %v", err)
	}

	topID, err := m.InsertNew(InsertNewParams{
		PlayerOneID: 300, PlayerTwoID: 400, PlayerOneIsWhite: true,
		TimeFormatInMilliseconds: testTimeFormat,
		GameHistory:              []byte("[]"),
		AverageElo:               2000,
		WhitePlayerElo:           2000, BlackPlayerElo: 2000,
	})
	if err != nil {
		t.Fatalf("InsertNew (high elo): %v", err)
	}

	id, err := m.GetHighestEloMatch()
	if err != nil {
		t.Fatalf("GetHighestEloMatch: %v", err)
	}
	if id != topID {
		t.Errorf("GetHighestEloMatch() = %d, want %d", id, topID)
	}
}

func TestLiveMatchModel_GetHighestEloMatch_Empty(t *testing.T) {
	m := &LiveMatchModel{DB: newTestDB(t)}

	_, err := m.GetHighestEloMatch()
	if err == nil {
		t.Error("expected error when no matches exist, got nil")
	}
}

func TestLiveMatchModel_MoveMatchToPastMatches(t *testing.T) {
	db := newTestDB(t)
	liveModel := &LiveMatchModel{DB: db}
	pastModel := &PastMatchModel{DB: db}

	matchID := insertTestMatch(t, liveModel)

	// Set last_move_piece and last_move_move — required NOT NULL in past_matches
	err := liveModel.UpdateLiveMatch(UpdateMatchParams{
		MatchID:                              matchID,
		NewFEN:                               "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		LastMovePiece:                        1,
		LastMoveMove:                         36,
		WhitePlayerTimeRemainingMilliseconds: testTimeFormat,
		BlackPlayerTimeRemainingMilliseconds: testTimeFormat,
		MatchStateHistoryJSON:                []byte("[]"),
		TimeOfLastMove:                       time.Now(),
	})
	if err != nil {
		t.Fatalf("UpdateLiveMatch: %v", err)
	}

	err = liveModel.MoveMatchToPastMatches(matchID, chess.OutcomeWhiteWins, chess.Checkmate, 10, -10)
	if err != nil {
		t.Fatalf("MoveMatchToPastMatches: %v", err)
	}

	// Match must be gone from live_matches
	_, err = liveModel.GetFromMatchID(matchID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected ErrNoRows for archived match, got %v", err)
	}

	// Match must appear in past_matches
	past, err := pastModel.GetPastMatchesWithFormat(PastMatchFilters{})
	if err != nil {
		t.Fatalf("GetPastMatchesWithFormat: %v", err)
	}
	if len(past) != 1 {
		t.Fatalf("expected 1 past match, got %d", len(past))
	}
	if past[0].MatchID != matchID {
		t.Errorf("past match ID = %d, want %d", past[0].MatchID, matchID)
	}
	if past[0].Result != int64(chess.OutcomeWhiteWins) {
		t.Errorf("Result = %d, want %d (OutcomeWhiteWins)", past[0].Result, chess.OutcomeWhiteWins)
	}
	if past[0].WhitePlayerEloGain != 10 {
		t.Errorf("WhitePlayerEloGain = %d, want 10", past[0].WhitePlayerEloGain)
	}
	if past[0].BlackPlayerEloGain != -10 {
		t.Errorf("BlackPlayerEloGain = %d, want -10", past[0].BlackPlayerEloGain)
	}
}

func TestPastMatchModel_GetPastMatchesWithFormat_Filters(t *testing.T) {
	db := newTestDB(t)
	liveModel := &LiveMatchModel{DB: db}
	userModel := &UserModel{DB: db, BcryptCost: bcrypt.MinCost}
	pastModel := &PastMatchModel{DB: db}

	whiteID := insertTestUser(t, userModel, "player_w", "pass")
	blackID := insertTestUser(t, userModel, "player_b", "pass")

	archiveMatch := func(playerOne, playerTwo int64, timeFormat int64) {
		t.Helper()
		id, err := liveModel.InsertNew(InsertNewParams{
			PlayerOneID:              playerOne,
			PlayerTwoID:              playerTwo,
			PlayerOneIsWhite:         true,
			TimeFormatInMilliseconds: timeFormat,
			GameHistory:              []byte("[]"),
			AverageElo:               1500,
			WhitePlayerElo:           1500,
			BlackPlayerElo:           1500,
		})
		if err != nil {
			t.Fatalf("InsertNew: %v", err)
		}
		err = liveModel.UpdateLiveMatch(UpdateMatchParams{
			MatchID: id, NewFEN: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			LastMovePiece: 1, LastMoveMove: 1,
			WhitePlayerTimeRemainingMilliseconds: timeFormat,
			BlackPlayerTimeRemainingMilliseconds: timeFormat,
			MatchStateHistoryJSON:                []byte("[]"),
			TimeOfLastMove:                       time.Now(),
		})
		if err != nil {
			t.Fatalf("UpdateLiveMatch: %v", err)
		}
		if err := liveModel.MoveMatchToPastMatches(id, chess.OutcomeDraw, chess.Draw, 0, 0); err != nil {
			t.Fatalf("MoveMatchToPastMatches: %v", err)
		}
	}

	blitzFormat := int64(3 * 60_000)  // 3 min blitz
	rapidFormat := int64(10 * 60_000) // 10 min rapid

	archiveMatch(whiteID, blackID, blitzFormat)
	archiveMatch(whiteID, blackID, rapidFormat)

	t.Run("no filter returns all", func(t *testing.T) {
		results, err := pastModel.GetPastMatchesWithFormat(PastMatchFilters{})
		if err != nil {
			t.Fatalf("GetPastMatchesWithFormat: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("got %d results, want 2", len(results))
		}
	})

	t.Run("filter by time format upper bound", func(t *testing.T) {
		upper := int64(5 * 60_000) // only blitz (3 min) passes
		results, err := pastModel.GetPastMatchesWithFormat(PastMatchFilters{TimeFormatUpper: &upper})
		if err != nil {
			t.Fatalf("GetPastMatchesWithFormat: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("got %d results, want 1", len(results))
		}
		if results[0].TimeFormatInMilliseconds != blitzFormat {
			t.Errorf("got time format %d, want %d", results[0].TimeFormatInMilliseconds, blitzFormat)
		}
	})

	t.Run("filter by username", func(t *testing.T) {
		name := "player_w"
		results, err := pastModel.GetPastMatchesWithFormat(PastMatchFilters{Username: &name})
		if err != nil {
			t.Fatalf("GetPastMatchesWithFormat: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("got %d results, want 2 (player_w played both games)", len(results))
		}
	})

	t.Run("filter by non-participant username", func(t *testing.T) {
		name := "nobody"
		results, err := pastModel.GetPastMatchesWithFormat(PastMatchFilters{Username: &name})
		if err != nil {
			t.Fatalf("GetPastMatchesWithFormat: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("got %d results, want 0", len(results))
		}
	})
}
