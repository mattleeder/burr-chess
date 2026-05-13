package models

import (
	"burrchess/internal/chess"
	"database/sql"
	"log/slog"
)

type RatingType int

const (
	bullet = iota
	blitz
	rapid
	classical
)

type UserRatingsModel struct {
	DB *sql.DB
}

type UserRatings struct {
	PlayerID        int64  `json:"playerID"`
	Username        string `json:"username"`
	BulletRating    int64  `json:"bulletRating"`
	BlitzRating     int64  `json:"blitzRating"`
	RapidRating     int64  `json:"rapidRating"`
	ClassicalRating int64  `json:"classicalRating"`
}

func (u *UserRatings) GetRatingForTimeFormat(timeFormatInMilliseconds int64) int64 {
	if timeFormatInMilliseconds < chess.Bullet[1] {
		return u.BulletRating
	} else if timeFormatInMilliseconds < chess.Blitz[1] {
		return u.BlitzRating
	} else if timeFormatInMilliseconds < chess.Rapid[1] {
		return u.RapidRating
	} else {
		return u.ClassicalRating
	}
}

func GetRatingTypeFromTimeFormat(timeFormatInMilliseconds int64) RatingType {
	if timeFormatInMilliseconds < chess.Bullet[1] {
		return bullet
	} else if timeFormatInMilliseconds < chess.Blitz[1] {
		return blitz
	} else if timeFormatInMilliseconds < chess.Rapid[1] {
		return rapid
	} else {
		return classical
	}
}

func (m *UserRatingsModel) getRating(query UserQuery) (UserRatings, error) {
	sqlStmt := `
	SELECT player_id,
	       username,
		   bullet_rating,
		   blitz_rating,
		   rapid_rating,
		   classical_rating
	  FROM user_ratings
	` + query.whereClause

	slog.Debug("getting rating", "arg", query.arg)

	var playerID int64
	var username string
	var bulletRating int64
	var blitzRating int64
	var rapidRating int64
	var classicalRating int64

	err := QueryRowWithRetry(m.DB, sqlStmt, []any{query.arg}, []any{&playerID, &username, &bulletRating, &blitzRating, &rapidRating, &classicalRating})
	if err != nil {
		slog.Error("error getting user_ratings", "err", err)
		return UserRatings{}, err
	}
	return UserRatings{
		PlayerID:        playerID,
		Username:        username,
		BulletRating:    bulletRating,
		BlitzRating:     blitzRating,
		RapidRating:     rapidRating,
		ClassicalRating: classicalRating,
	}, nil
}

func (m *UserRatingsModel) GetRatingFromUsername(username string) (UserRatings, error) {
	return m.getRating(ByUsername(username))
}

func (m *UserRatingsModel) GetRatingFromPlayerID(playerID int64) (UserRatings, error) {
	return m.getRating(ByPlayerID(playerID))
}

func (m *UserRatingsModel) updateRating(query UserQuery, ratingType RatingType, newRating int64) error {
	slog.Debug("updating rating", "newRating", newRating)

	sqlStmt := `
	UPDATE user_ratings
	`

	switch ratingType {
	case bullet:
		sqlStmt += " SET bullet_rating = ?"
	case blitz:
		sqlStmt += " SET blitz_rating = ?"
	case rapid:
		sqlStmt += " SET rapid_rating = ?"
	case classical:
		sqlStmt += " SET classical_rating = ?"
	}

	sqlStmt += query.whereClause

	tx, err := m.DB.Begin()
	if err != nil {
		slog.Error("error starting transaction", "err", err)
		return err
	}

	stmt, err := tx.Prepare(sqlStmt)
	if err != nil {
		slog.Error("error preparing statement", "err", err)
		return err
	}
	defer stmt.Close()

	_, err = ExecStatementWithRetry(stmt, newRating, query.arg)
	if err != nil {
		slog.Error("error executing statement", "err", err)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			slog.Error("updateRating: unable to rollback", "err", rollbackErr)
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		slog.Error("error committing transaction", "err", err)
		return err
	}

	return err
}

func (m *UserRatingsModel) UpdateRatingFromUsername(username string, ratingType RatingType, newRating int64) error {
	return m.updateRating(ByUsername(username), ratingType, newRating)
}

func (m *UserRatingsModel) UpdateRatingFromPlayerID(playerID int64, ratingType RatingType, newRating int64) error {
	return m.updateRating(ByPlayerID(playerID), ratingType, newRating)
}

func (m *UserRatingsModel) LogAll() {
	slog.Debug("UserRatings")

	rows, err := QueryWithRetry(m.DB, "select * from user_ratings;")
	if err != nil {
		slog.Error("error querying user_ratings", "err", err)
		return
	}

	defer rows.Close()
	for rows.Next() {
		slog.Debug("row", "data", rows)
	}
	err = rows.Err()
	if err != nil {
		slog.Error("error iterating user_ratings rows", "err", err)
	}
}
