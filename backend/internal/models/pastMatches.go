package models

import (
	"database/sql"
)

// @TODO: DOES SENDING THINGS AS sql.NullType GIVE AWAY THAT IT IS SQL DATABASE?

type PastMatch struct {
	MatchID                  int64          `json:"matchID"`
	WhitePlayerID            int64         `json:"whitePlayerID"`
	BlackPlayerID            int64         `json:"blackPlayerID"`
	LastMovePiece            sql.NullInt64  `json:"lastMovePiece"`
	LastMoveMove             sql.NullInt64  `json:"lastMoveMove"`
	FinalFEN                 string         `json:"currentFEN"`
	TimeFormatInMilliseconds int64          `json:"timeFormatInMilliseconds"`
	IncrementInMilliseconds  int64          `json:"incrementInMilliseconds"`
	GameHistoryJSONString    []byte         `json:"gameHistoryJSONstring"` // []MatchStateHistory{}
	Result                   int64          `json:"result"`
	ResultReason             int64          `json:"resultReason"`
	WhitePlayerElo           float64        `json:"whitePlayerElo"`
	BlackPlayerElo           float64        `json:"blackPlayerElo"`
	WhitePlayerEloGain       float64        `json:"whitePlayerEloGain"`
	BlackPlayerEloGain       float64        `json:"blackPlayerEloGain"`
	AverageElo               float64        `json:"averageElo"`
	MatchStartTime           int64          `json:"matchStartTime"`
	MatchEndTime             int64          `json:"matchEndTime"`
}

type PastMatchSummary struct {
	MatchID                  int64          `json:"matchID"`
	WhitePlayerUsername      sql.NullString `json:"whitePlayerUsername"`
	BlackPlayerUsername      sql.NullString `json:"blackPlayerUsername"`
	LastMovePiece            sql.NullInt64  `json:"lastMovePiece"`
	LastMoveMove             sql.NullInt64  `json:"lastMoveMove"`
	FinalFEN                 string         `json:"finalFEN"`
	TimeFormatInMilliseconds int64          `json:"timeFormatInMilliseconds"`
	IncrementInMilliseconds  int64          `json:"incrementInMilliseconds"`
	Result                   int64          `json:"result"`
	ResultReason             int64          `json:"resultReason"`
	WhitePlayerElo           float64        `json:"whitePlayerElo"`
	BlackPlayerElo           float64        `json:"blackPlayerElo"`
	WhitePlayerEloGain       float64        `json:"whitePlayerEloGain"`
	BlackPlayerEloGain       float64        `json:"blackPlayerEloGain"`
	AverageElo               float64        `json:"averageElo"`
	MatchStartTime           int64          `json:"matchStartTime"`
	MatchEndTime             int64          `json:"matchEndTime"`
}

type PastMatchModel struct {
	DB *sql.DB
}

type PastMatchFilters struct {
	TimeFormatLower *int64
	TimeFormatUpper *int64
	Username        *string
}

func (m *PastMatchModel) LogAll() {
	app.infoLog.Println("Past Matches:")

	rows, err := QueryWithRetry(m.DB, "select * from past_matches;")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	defer rows.Close()
	for rows.Next() {
		app.rowsLog.Println(rows)
	}
	err = rows.Err()
	if err != nil {
		app.errorLog.Println(err)
	}
}

func (m *PastMatchModel) GetPastMatchesWithFormat(filters PastMatchFilters) ([]PastMatchSummary, error) {
	args := []any{}
	// Left join for anonymous players
	sqlStmt := `
	SELECT m.match_id,
	       white_player.username,
		   black_player.username,
		   m.last_move_piece,
		   m.last_move_move,
		   m.final_fen,
		   m.time_format_in_milliseconds,
		   m.increment_in_milliseconds,
		   m.result,
		   m.result_reason,
		   m.white_player_elo,
		   m.black_player_elo,
		   m.white_player_elo_gain,
		   m.black_player_elo_gain,
		   m.average_elo,
		   m.match_start_time,
		   m.match_end_time
	  FROM past_matches as m
	  LEFT JOIN users as white_player
	    ON m.white_player_id = white_player.player_id
	  LEFT JOIN users as black_player
	    on m.black_player_id = black_player.player_id
	 WHERE 1=1
	`

	var output []PastMatchSummary

	if filters.TimeFormatLower != nil {
		sqlStmt += " AND m.time_format_in_milliseconds > ?"
		app.infoLog.Printf("TimeFormatLower: %v\n", filters.TimeFormatLower)
		args = append(args, filters.TimeFormatLower)
	}

	if filters.TimeFormatUpper != nil {
		sqlStmt += " AND m.time_format_in_milliseconds <= ?"
		app.infoLog.Printf("TimeFormatUpper: %v\n", filters.TimeFormatUpper)
		args = append(args, filters.TimeFormatUpper)
	}

	if filters.Username != nil {
		sqlStmt += ` AND (white_player.username = ? OR black_player.username = ?)`
		args = append(args, filters.Username, filters.Username)
	}

	rows, err := QueryWithRetry(m.DB, sqlStmt, args...)
	if err != nil {
		app.errorLog.Printf("Error getting past matches with format: %s\n", err.Error())
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var m PastMatchSummary
		err := rows.Scan(
			&m.MatchID,
			&m.WhitePlayerUsername,
			&m.BlackPlayerUsername,
			&m.LastMovePiece,
			&m.LastMoveMove,
			&m.FinalFEN,
			&m.TimeFormatInMilliseconds,
			&m.IncrementInMilliseconds,
			&m.Result,
			&m.ResultReason,
			&m.WhitePlayerElo,
			&m.BlackPlayerElo,
			&m.WhitePlayerEloGain,
			&m.BlackPlayerEloGain,
			&m.AverageElo,
			&m.MatchStartTime,
			&m.MatchEndTime,
		)

		if err != nil {
			app.errorLog.Printf("Error in GetPastMatchesWithFormat: %s\n", err.Error())
			return nil, err
		}

		output = append(output, m)
	}

	return output, nil
}
