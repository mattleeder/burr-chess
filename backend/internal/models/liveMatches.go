package models

import (
	"burrchess/internal/chess"
	"database/sql"
	"errors"
	"time"
)

type LiveMatch struct {
	MatchID                              int64         `json:"matchID"`
	WhitePlayerID                        int64         `json:"whitePlayerID"`
	BlackPlayerID                        int64         `json:"blackPlayerID"`
	LastMovePiece                        sql.NullInt64 `json:"lastMovePiece"`
	LastMoveMove                         sql.NullInt64 `json:"lastMoveMove"`
	CurrentFEN                           string        `json:"currentFEN"`
	TimeFormatInMilliseconds             int64         `json:"timeFormatInMilliseconds"`
	IncrementInMilliseconds              int64         `json:"incrementInMilliseconds"`
	WhitePlayerTimeRemainingMilliseconds int64         `json:"whitePlayerTimeRemainingMilliseconds"`
	BlackPlayerTimeRemainingMilliseconds int64         `json:"blackPlayerTimeRemainingMilliseconds"`
	GameHistoryJSONString                []byte        `json:"gameHistoryJSONstring"` // []MatchStateHistory{}
	UnixMsTimeOfLastMove                 int64         `json:"unixTimeOfLastMove"`
	AverageElo                           float64       `json:"averageElo"`
	WhitePlayerElo                       int64         `json:"whitePlayerElo"`
	BlackPlayerElo                       int64         `json:"blackPlayerElo"`
	MatchStartTime                       int64         `json:"matchStartTime"`
}

type LiveMatchWithUsernames struct {
	MatchID                              int64          `json:"matchID"`
	WhitePlayerID                        int64          `json:"whitePlayerID"`
	BlackPlayerID                        int64          `json:"blackPlayerID"`
	LastMovePiece                        sql.NullInt64  `json:"lastMovePiece"`
	LastMoveMove                         sql.NullInt64  `json:"lastMoveMove"`
	CurrentFEN                           string         `json:"currentFEN"`
	TimeFormatInMilliseconds             int64          `json:"timeFormatInMilliseconds"`
	IncrementInMilliseconds              int64          `json:"incrementInMilliseconds"`
	WhitePlayerTimeRemainingMilliseconds int64          `json:"whitePlayerTimeRemainingMilliseconds"`
	BlackPlayerTimeRemainingMilliseconds int64          `json:"blackPlayerTimeRemainingMilliseconds"`
	GameHistoryJSONString                []byte         `json:"gameHistoryJSONstring"` // []MatchStateHistory{}
	UnixMsTimeOfLastMove                 int64          `json:"unixTimeOfLastMove"`
	AverageElo                           float64        `json:"averageElo"`
	WhitePlayerElo                       int64          `json:"whitePlayerElo"`
	BlackPlayerElo                       int64          `json:"blackPlayerElo"`
	MatchStartTime                       int64          `json:"matchStartTime"`
	WhitePlayerUsername                  sql.NullString `json:"whitePlayerUsername"`
	BlackPlayerUsername                  sql.NullString `json:"blackPlayerUsername"`
}

type LiveMatchModel struct {
	DB *sql.DB
}

type InsertNewParams struct {
	PlayerOneID              int64
	PlayerTwoID              int64
	PlayerOneIsWhite         bool
	TimeFormatInMilliseconds int64
	IncrementInMilliseconds  int64
	GameHistory              []byte
	AverageElo               float64
	WhitePlayerElo           int64
	BlackPlayerElo           int64
}

type UpdateMatchParams struct {
	MatchID                              int64
	NewFEN                               string
	LastMovePiece                        int
	LastMoveMove                         int
	WhitePlayerTimeRemainingMilliseconds int64
	BlackPlayerTimeRemainingMilliseconds int64
	MatchStateHistoryJSON                []byte
	TimeOfLastMove                       time.Time
}

func (m *LiveMatchModel) InsertNew(p InsertNewParams) (int64, error) {
	app.infoLog.Printf("Inserting new match")
	var result sql.Result
	var err error

	app.infoLog.Printf("Inserting new match with: timeFormat: %v, increment: %v\n", p.TimeFormatInMilliseconds, p.IncrementInMilliseconds)

	sqlStmt := `
	INSERT INTO live_matches (
	    white_player_id,
		black_player_id,
		time_format_in_milliseconds,
		increment_in_milliseconds,
		white_player_time_remaining_in_milliseconds,
		black_player_time_remaining_in_milliseconds,
		game_history_json_string,
		unix_ms_time_of_last_move,
		average_elo,
		white_player_elo,
		black_player_elo,
		match_start_time
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	// Set white and black remaining time equal to the time format

	tx, err := m.DB.Begin()
	if err != nil {
		app.errorLog.Printf("Error starting transaction: %v\n", err)
		return 0, err
	}

	insertStmt, err := tx.Prepare(sqlStmt)
	if err != nil {
		app.errorLog.Printf("Error preparing statement: %v\n", err)
		return 0, err
	}
	defer insertStmt.Close()

	whiteID, blackID := p.PlayerOneID, p.PlayerTwoID
	if !p.PlayerOneIsWhite {
		whiteID, blackID = p.PlayerTwoID, p.PlayerOneID
	}

	result, err = ExecStatementWithRetry(insertStmt, whiteID, blackID, p.TimeFormatInMilliseconds, p.IncrementInMilliseconds, p.TimeFormatInMilliseconds, p.TimeFormatInMilliseconds, p.GameHistory, time.Time.UnixMilli(time.Now()), p.AverageElo, p.WhitePlayerElo, p.BlackPlayerElo, time.Time.Unix(time.Now()))

	if err != nil {
		app.errorLog.Printf("Error executing statement: %v\n", err)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			app.errorLog.Printf("insert live_matches: unable to rollback: %v", rollbackErr)
		}
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		app.errorLog.Printf("Error commiting transaction in InsertNew: %v\n", err)
		return 0, err
	}

	insertID, err := result.LastInsertId()
	if err != nil {
		app.errorLog.Printf("Unsuccesfully inserted new match with err: %s", err.Error())
	} else {
		app.infoLog.Printf("Succesfully inserted new match with id: %v", insertID)
	}

	return result.LastInsertId()
}

func (m *LiveMatchModel) EnQueueReturnInsertNew(p InsertNewParams) (int64, error) {
	result, err := DBTaskQueue.EnQueueReturn(func() (any, error) {
		return m.InsertNew(p)
	})
	if err != nil {
		return 0, err
	}
	coercedResult, ok := result.(int64)
	if !ok {
		app.errorLog.Println("coercedResult is not int64")
		return 0, errors.New("coercedResult is not int64")
	}
	return coercedResult, nil
}

func (m *LiveMatchModel) EnQueueInsertNew(p InsertNewParams) {
	DBTaskQueue.EnQueue(func() (any, error) {
		return m.InsertNew(p)
	})
}

func (m *LiveMatchModel) GetFromMatchID(matchID int64) (*LiveMatchWithUsernames, error) {
	sqlStmt := `
	SELECT live_matches.match_id,
           live_matches.white_player_id,
           live_matches.black_player_id,
           live_matches.last_move_piece,
           live_matches.last_move_move,
           live_matches.current_fen,
           live_matches.time_format_in_milliseconds,
           live_matches.increment_in_milliseconds,
           live_matches.white_player_time_remaining_in_milliseconds,
           live_matches.black_player_time_remaining_in_milliseconds,
           live_matches.game_history_json_string,
           live_matches.unix_ms_time_of_last_move,
           live_matches.average_elo,
           live_matches.white_player_elo,
           live_matches.black_player_elo,
           live_matches.match_start_time,
		   white_player.username,
		   black_player.username
	  FROM live_matches
	  LEFT JOIN users as white_player
	    ON live_matches.white_player_id = white_player.player_id
	  LEFT JOIN users as black_player
	    on live_matches.black_player_id = black_player.player_id
	 WHERE match_id = ?
	`

	var match LiveMatchWithUsernames

	err := QueryRowWithRetry(
		m.DB,
		sqlStmt,
		[]any{matchID},
		[]any{
			&match.MatchID,
			&match.WhitePlayerID,
			&match.BlackPlayerID,
			&match.LastMovePiece,
			&match.LastMoveMove,
			&match.CurrentFEN,
			&match.TimeFormatInMilliseconds,
			&match.IncrementInMilliseconds,
			&match.WhitePlayerTimeRemainingMilliseconds,
			&match.BlackPlayerTimeRemainingMilliseconds,
			&match.GameHistoryJSONString,
			&match.UnixMsTimeOfLastMove,
			&match.AverageElo,
			&match.WhitePlayerElo,
			&match.BlackPlayerElo,
			&match.MatchStartTime,
			&match.WhitePlayerUsername,
			&match.BlackPlayerUsername,
		},
	)
	if err != nil {
		return nil, err
	}

	app.infoLog.Printf("%+v", match)

	return &match, nil
}

func (m *LiveMatchModel) EnQueueReturnGetFromMatchID(matchID int64) (*LiveMatchWithUsernames, error) {
	result, err := DBTaskQueue.EnQueueReturn(func() (any, error) {
		return m.GetFromMatchID(matchID)
	})
	if err != nil {
		return nil, err
	}
	matchState, ok := result.(*LiveMatchWithUsernames)
	if !ok {
		app.errorLog.Println("matchState is not *models.LiveMatchWithUsernames")
		return nil, errors.New("matchState is not *models.LiveMatchWithUsernames")
	}
	return matchState, nil
}

func (m *LiveMatchModel) LogAll() {
	app.infoLog.Println("Live Matches:")

	rows, err := QueryWithRetry(m.DB, "select * from live_matches;")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	defer rows.Close()
	app.rowsLog.Println(rows.Columns())
	for rows.Next() {
		app.rowsLog.Printf("%v\n", rows)
	}
	err = rows.Err()
	if err != nil {
		app.errorLog.Println(err)
	}
}

func (m *LiveMatchModel) EnQueueLogAll() {
	DBTaskQueue.EnQueueErrorOnlyTask(func() error {
		m.LogAll()
		return nil
	})
}

func (m *LiveMatchModel) UpdateLiveMatch(p UpdateMatchParams) error {
	sqlStmt := `
	UPDATE live_matches
	   SET last_move_piece = ?,
	       last_move_move = ?,
		   current_fen = ?,
		   white_player_time_remaining_in_milliseconds = ?,
		   black_player_time_remaining_in_milliseconds = ?,
		   game_history_json_string = ?,
		   unix_ms_time_of_last_move = ?
	 WHERE match_id = ?
	`

	tx, err := m.DB.Begin()
	if err != nil {
		app.errorLog.Printf("Error starting transaction: %v\n", err)
		return err
	}

	updateStmt, err := tx.Prepare(sqlStmt)
	if err != nil {
		app.errorLog.Printf("Error preparing statement: %v\n", err)
		return err
	}
	defer updateStmt.Close()

	_, err = ExecStatementWithRetry(updateStmt, p.LastMovePiece, p.LastMoveMove, p.NewFEN, p.WhitePlayerTimeRemainingMilliseconds, p.BlackPlayerTimeRemainingMilliseconds, p.MatchStateHistoryJSON, time.Time.UnixMilli(p.TimeOfLastMove), p.MatchID)
	if err != nil {
		app.errorLog.Printf("Error executing statement: %v\n", err)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			app.errorLog.Printf("UpdateLiveMatch: unable to rollback: %v", rollbackErr)
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		app.errorLog.Printf("Error commiting transaction in updateLiveMatch: %v\n", err)
		return err
	}

	return err
}

func (m *LiveMatchModel) EnQueueReturnUpdateLiveMatch(p UpdateMatchParams) error {
	return DBTaskQueue.EnQueueReturnErrorOnlyTask(func() error {
		return m.UpdateLiveMatch(p)
	})
}

func (m *LiveMatchModel) EnQueueUpdateLiveMatch(p UpdateMatchParams) {
	DBTaskQueue.EnQueueErrorOnlyTask(func() error {
		return m.UpdateLiveMatch(p)
	})
}

func (m *LiveMatchModel) MoveMatchToPastMatches(matchID int64, result int, resultReason chess.GameOverStatusCode, whitePlayerEloGain int64, blackPlayerEloGain int64) error {
	// outcome int
	// draw      = 0
	// whiteWins = 1
	// blackWins = 2
	app.infoLog.Printf("Moving %v to past matches", matchID)

	stepOne := `
		-- Step 1: Insert row into past_matches table
	INSERT INTO past_matches (
	    match_id,
		white_player_id,
		black_player_id,
		last_move_piece,
		last_move_move,
		final_fen,
		time_format_in_milliseconds,
		increment_in_milliseconds,
		game_history_json_string,
		result,
		result_reason,
		white_player_elo,
		black_player_elo,
		white_player_elo_gain,
        black_player_elo_gain,
		average_elo,
		match_start_time,
		match_end_time
		)

	SELECT match_id,
           white_player_id,
           black_player_id,
           last_move_piece,
           last_move_move,
           current_fen as final_fen,
           time_format_in_milliseconds,
           increment_in_milliseconds,
           game_history_json_string,
		   ?,
		   ?,
		   white_player_elo,
		   black_player_elo,
		   ?,
		   ?,
		   average_elo,
		   match_start_time,
		   ?
	  FROM live_matches
	 WHERE match_id = ?;`

	stepTwo := `
	-- Step 2: Delete row from live_matches table
	DELETE FROM live_matches
	 WHERE match_id = ?;
	`

	var stmtOne, stmtTwo *sql.Stmt

	tx, err := m.DB.Begin()
	if err != nil {
		app.errorLog.Printf("Error starting transaction: %v\n", err)
		return err
	}

	stmtOne, err = tx.Prepare(stepOne)
	if err != nil {
		app.errorLog.Printf("Error preparing first statement: %v\n", err)
		return err
	}
	defer stmtOne.Close()

	stmtTwo, err = tx.Prepare(stepTwo)
	if err != nil {
		app.errorLog.Printf("Error preparing second statement: %v\n", err)
		return err
	}
	defer stmtTwo.Close()

	_, err = ExecStatementWithRetry(stmtOne, result, resultReason, whitePlayerEloGain, blackPlayerEloGain, time.Now().Unix(), matchID)
	if err != nil {
		app.errorLog.Printf("Error executing first statement: %v\n", err)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			app.errorLog.Printf("insert past_matches: unable to rollback: %v", rollbackErr)
		}
		return err
	}

	_, err = ExecStatementWithRetry(stmtTwo, matchID)
	if err != nil {
		app.errorLog.Printf("Error executing second statement: %v\n", err)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			app.errorLog.Printf("delete live_matches: unable to rollback: %v", rollbackErr)
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		app.errorLog.Printf("Error commiting transaction: %v\n", err)
		return err
	}

	return err
}

func (m *LiveMatchModel) EnQueueReturnMoveMatchToPastMatches(matchID int64, result int, resultReason chess.GameOverStatusCode, whitePlayerEloGain int64, blackPlayerEloGain int64) error {
	return DBTaskQueue.EnQueueReturnErrorOnlyTask(func() error {
		return m.MoveMatchToPastMatches(matchID, result, resultReason, whitePlayerEloGain, blackPlayerEloGain)
	})
}

func (m *LiveMatchModel) EnQueueMoveMatchToPastMatches(matchID int64, result int, resultReason chess.GameOverStatusCode, whitePlayerEloGain int64, blackPlayerEloGain int64) {
	DBTaskQueue.EnQueueErrorOnlyTask(func() error {
		return m.MoveMatchToPastMatches(matchID, result, resultReason, whitePlayerEloGain, blackPlayerEloGain)
	})
}

func (m *LiveMatchModel) GetHighestEloMatch() (matchID int64, err error) {

	var matchIDorNull sql.NullInt64

	sqlStmt := `
	SELECT match_id
	  FROM live_matches
	 ORDER by average_elo DESC
	 LIMIT 1
	`

	err = QueryRowWithRetry(m.DB, sqlStmt, []any{}, []any{&matchIDorNull})
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			app.errorLog.Println("No matches currently being played")
		} else {
			app.errorLog.Printf("Error getting matchID: %s", err.Error())
		}
		return 0, err
	}

	if !matchIDorNull.Valid {
		app.errorLog.Println("MatchID is null")
		return 0, errors.New("MatchID is null")
	}

	matchID = matchIDorNull.Int64

	return matchID, nil
}

func (m *LiveMatchModel) IsPlayerInMatch(playerID int64) (bool, error) {
	sqlStmt := `
	SELECT match_id
	  FROM live_matches
	  WHERE white_player_id = ?
	     OR black_player_id = ?
	 LIMIT 1
	`

	var matchIDorNull sql.NullInt64

	err := QueryRowWithRetry(m.DB, sqlStmt, []any{playerID, playerID}, []any{&matchIDorNull})
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false, nil
		} else {
			app.errorLog.Printf("Error getting matchID: %s", err.Error())
		}
	}

	app.infoLog.Printf("Player: %v in match %v\n", playerID, matchIDorNull)

	return true, err
}
