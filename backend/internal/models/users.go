package models

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	MinUsernameLength       = 3
	MaxUsernameLength       = 32
	MinPasswordLength       = 1
	MaxPasswordLength       = 72 // bcrypt silently truncates at 72 bytes
	maxPlayerIDRetries      = 5
	playerIDUniqueErrSubstr = "users.player_id"
)

type UserQuery struct {
	whereClause string
	arg         any
}

func ByUsername(username string) UserQuery {
	return UserQuery{whereClause: " WHERE username = ?", arg: username}
}

func ByPlayerID(playerID int64) UserQuery {
	return UserQuery{whereClause: " WHERE player_id = ?", arg: playerID}
}

type NewUserOptions struct {
	email *string
}

type NewUserInfo struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Email      string `json:"email"`
	RememberMe bool   `json:"rememberMe"`
}

type UserLoginInfo struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe"`
}

type UserModel struct {
	DB         *sql.DB
	BcryptCost int // must be set by the application; 0 is rejected
}

type UserServerSide struct {
	PlayerID int64  `json:"playerID"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	JoinDate int64  `json:"joinDate"`
	LastSeen int64  `json:"lastSeen"`
}

type UserClientSide struct {
	PlayerID int64  `json:"playerID"`
	Username string `json:"username"`
	JoinDate int64  `json:"joinDate"`
	LastSeen int64  `json:"lastSeen"`
}

type Ratings struct {
	BulletRating    int64 `json:"bullet"`
	BlitzRating     int64 `json:"blitz"`
	RapidRating     int64 `json:"rapid"`
	ClassicalRating int64 `json:"classical"`
}

type UserTileInfo struct {
	PlayerID      int64   `json:"playerID"`
	Username      string  `json:"username"`
	PingStatus    bool    `json:"pingStatus"`
	JoinDate      int64   `json:"joinDate"`
	LastSeen      int64   `json:"lastSeen"`
	Ratings       Ratings `json:"ratings"`
	NumberOfGames int64   `json:"numberOfGames"`
}

type AccountSettings struct {
	Email *string `json:"email"`
}

// isValidEmail returns false if the string is non-empty but not a plausible email address.
// An empty string is allowed (means "remove email").
func IsValidEmail(email string) bool {
	if email == "" {
		return true
	}
	atIdx := strings.Index(email, "@")
	if atIdx <= 0 || atIdx == len(email)-1 {
		return false
	}
	domain := email[atIdx+1:]
	return strings.Contains(domain, ".") && !strings.HasSuffix(domain, ".") && strings.Count(email, "@") == 1
}

func (m *UserModel) hashPassword(password string) (string, error) {
	if m.BcryptCost == 0 {
		return "", fmt.Errorf("UserModel.BcryptCost not set")
	}
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), m.BcryptCost)
	return string(hashedPasswordBytes), err
}

func doesPasswordMatch(plaintextPassword string, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plaintextPassword))
	return err == nil
}

func (m *UserModel) execInTransaction(sqlStmt string, args ...any) error {
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

	_, err = ExecStatementWithRetry(stmt, args...)
	if err != nil {
		slog.Error("error executing statement", "err", err)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			slog.Error("unable to rollback", "err", rollbackErr)
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		slog.Error("error committing transaction", "err", err)
		return err
	}

	return nil
}

func CreateNewUserOptions(newUser NewUserInfo) (options NewUserOptions) {
	if newUser.Email != "" {
		options.email = &newUser.Email
	}

	return options
}

func (m *UserModel) InsertNew(username string, password string, options *NewUserOptions) (int64, error) {

	slog.Info("inserting new user", "username", username)

	hashedPassword, err := m.hashPassword(password)
	password = "" // Overwrite password to avoid accidental usage
	if err != nil {
		slog.Error("error generating password hash", "err", err)
		return 0, err
	}
	var email = sql.NullString{Valid: false}
	if options.email != nil {
		email = sql.NullString{String: *options.email, Valid: true}
	}

	stepOne := `
	insert into users (player_id, username, password, email) VALUES (?, ?, ?, ?);
	`
	stepTwo := `
	insert into user_ratings (player_id, username) VALUES (?, ?);
	`
	var stmtOne, stmtTwo *sql.Stmt

	tx, err := m.DB.Begin()
	if err != nil {
		slog.Error("error starting transaction", "err", err)
		return 0, err
	}

	stmtOne, err = tx.Prepare(stepOne)
	if err != nil {
		slog.Error("error preparing first statement", "err", err)
		return 0, err
	}
	defer stmtOne.Close()

	stmtTwo, err = tx.Prepare(stepTwo)
	if err != nil {
		slog.Error("error preparing second statement", "err", err)
		return 0, err
	}
	defer stmtTwo.Close()

	var playerID int64
	for range maxPlayerIDRetries {
		playerID = GenerateNewPlayerId()
		_, err = ExecStatementWithRetry(stmtOne, playerID, username, hashedPassword, email)
		if err == nil {
			break
		}
		if strings.Contains(err.Error(), playerIDUniqueErrSubstr) {
			slog.Debug("player_id collision, retrying")
			continue
		}
		// Any other error (e.g. username UNIQUE violation) — don't retry
		slog.Error("error inserting new user", "err", err)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			slog.Error("insert users: unable to rollback", "err", rollbackErr)
		}
		return 0, err
	}
	if err != nil {
		slog.Error("failed to insert user after max retries", "maxRetries", maxPlayerIDRetries)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			slog.Error("insert users: unable to rollback", "err", rollbackErr)
		}
		return 0, err
	}

	_, err = ExecStatementWithRetry(stmtTwo, playerID, username)

	if err != nil {
		slog.Error("error executing second statement", "err", err)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			slog.Error("insert user_ratings: unable to rollback", "err", rollbackErr)
		}
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		slog.Error("error committing transaction", "err", err)
		return 0, err
	}

	slog.Info("inserted new user", "playerID", playerID)

	return playerID, nil
}

func (m *UserModel) Authenticate(username string, password string) (playerID int64, authorized bool) {
	slog.Debug("checking password", "username", username)

	sqlStmt := `
	select player_id, password from users where username = ?
	`
	var hashedPassword string
	err := QueryRowWithRetry(m.DB, sqlStmt, []any{username}, []any{&playerID, &hashedPassword})
	if err != nil {
		slog.Error("error getting password for user", "err", err)
		return 0, false
	}

	authorized = doesPasswordMatch(password, hashedPassword)

	if !authorized {
		return 0, false
	}

	return playerID, true
}

func (m *UserModel) LogAll() {
	slog.Debug("Users")

	rows, err := QueryWithRetry(m.DB, "select * from users;")
	if err != nil {
		slog.Error("error querying users", "err", err)
		return
	}

	defer rows.Close()
	for rows.Next() {
		slog.Debug("row", "data", rows)
	}
	err = rows.Err()
	if err != nil {
		slog.Error("error iterating users rows", "err", err)
	}
}

func (m *UserModel) GetUserClientSide(query UserQuery) (UserClientSide, error) {
	sqlStmt := `
	SELECT player_id,
	       username,
		   join_date,
		   last_seen
	  FROM users
	` + query.whereClause

	var playerID int64
	var username string
	var joinDate int64
	var lastSeen int64

	err := QueryRowWithRetry(m.DB, sqlStmt, []any{query.arg}, []any{&playerID, &username, &joinDate, &lastSeen})
	if err != nil {
		slog.Error("error getting user", "err", err)
		return UserClientSide{}, err
	}
	return UserClientSide{
		PlayerID: playerID,
		Username: username,
		JoinDate: joinDate,
		LastSeen: lastSeen,
	}, nil
}

func (m *UserModel) GetUserServerSide(query UserQuery) (UserServerSide, error) {
	sqlStmt := `
	SELECT player_id,
	       username,
		   password,
		   email,
		   join_date,
		   last_seen
	  FROM users
	` + query.whereClause

	var playerID int64
	var username string
	var password string
	var email sql.NullString
	var joinDate int64
	var lastSeen int64

	err := QueryRowWithRetry(m.DB, sqlStmt, []any{query.arg}, []any{&playerID, &username, &password, &email, &joinDate, &lastSeen})
	if err != nil {
		slog.Error("error getting user", "err", err)
		return UserServerSide{}, err
	}
	return UserServerSide{
		PlayerID: playerID,
		Username: username,
		Password: password,
		Email:    email.String,
		JoinDate: joinDate,
		LastSeen: lastSeen,
	}, nil
}

func (m *UserModel) UpdateLastSeen(query UserQuery) error {
	sqlStmt := `
	UPDATE users
	   SET last_seen = unixepoch('now')
	` + query.whereClause

	return m.execInTransaction(sqlStmt, query.arg)
}

func (m *UserModel) SearchForUsers(searchString string) ([]UserClientSide, error) {
	sqlStmt := `
	SELECT player_id, username, join_date, last_seen
	  FROM users
	 WHERE UPPER(username) GLOB ?
	 LIMIT 20
	`

	var output []UserClientSide
	var playerID int64
	var username string
	var joinDate int64
	var lastSeen int64

	// Escape GLOB special characters so user input can't act as wildcards,
	// then append * for prefix matching.
	escaped := strings.NewReplacer("*", "[*]", "?", "[?]", "[", "[[").Replace(strings.ToUpper(searchString))
	rows, err := QueryWithRetry(m.DB, sqlStmt, escaped+"*")
	if err != nil {
		slog.Error("error in SearchForUsers", "err", err)
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&playerID, &username, &joinDate, &lastSeen)

		if err != nil {
			slog.Error("error scanning user in SearchForUsers", "err", err)
			return nil, err
		}

		output = append(output, UserClientSide{
			PlayerID: playerID,
			Username: username,
			JoinDate: joinDate,
			LastSeen: lastSeen,
		})
	}

	return output, nil
}

func (m *UserModel) GetTileInfoFromUsername(username string) (*UserTileInfo, error) {
	sqlStmt := `
	SELECT users.player_id,
	       users.join_date,
	       users.last_seen,
	       user_ratings.bullet_rating,
	       user_ratings.blitz_rating,
	       user_ratings.rapid_rating,
	       user_ratings.classical_rating,
	       COUNT(past_matches.match_id) as number_of_games
	  FROM users
	 INNER JOIN user_ratings
	    ON users.player_id = user_ratings.player_id
	  LEFT JOIN past_matches
	    ON past_matches.white_player_id = users.player_id
	    OR past_matches.black_player_id = users.player_id
	 WHERE users.username = ?
	 GROUP BY users.player_id
	`

	var playerID int64
	var joinDate int64
	var lastSeen int64
	var bulletRating int64
	var blitzRating int64
	var rapidRating int64
	var classicalRating int64
	var numberOfGames int64

	err := QueryRowWithRetry(m.DB, sqlStmt, []any{username}, []any{&playerID, &joinDate, &lastSeen, &bulletRating, &blitzRating, &rapidRating, &classicalRating, &numberOfGames})
	if err != nil {
		slog.Error("error in GetTileInfoFromUsername", "err", err)
		return nil, err
	}

	pingStatus := time.Since(time.Unix(lastSeen, 0)) < 10*time.Second

	return &UserTileInfo{
		PlayerID:   playerID,
		Username:   username,
		PingStatus: pingStatus,
		JoinDate:   joinDate,
		LastSeen:   lastSeen,
		Ratings: Ratings{
			BulletRating:    bulletRating,
			BlitzRating:     blitzRating,
			RapidRating:     rapidRating,
			ClassicalRating: classicalRating,
		},
		NumberOfGames: numberOfGames,
	}, nil
}

func (m *UserModel) GetUserAccountSettings(playerID int64) (AccountSettings, error) {
	sqlStmt := `
	SELECT email
	  FROM users
	 WHERE player_id = ?
	`

	var email sql.NullString
	var err error

	err = QueryRowWithRetry(m.DB, sqlStmt, []any{playerID}, []any{&email})

	if err != nil {
		slog.Error("error getting account settings", "err", err)
		return AccountSettings{}, err
	}
	var result AccountSettings
	if email.Valid {
		result.Email = &email.String
	}
	return result, nil
}

func (m *UserModel) UpdateEmail(playerID int64, newEmail string) error {
	sqlStmt := `
	UPDATE users
	   SET email = ?
	 WHERE player_id = ?
	`

	updateEmail := sql.NullString{
		String: newEmail,
		Valid:  newEmail != "",
	}

	return m.execInTransaction(sqlStmt, updateEmail, playerID)
}

func (m *UserModel) UpdatePassword(playerID int64, newPassword string) error {
	sqlStmt := `
	UPDATE users
	   SET password = ?
	 WHERE player_id = ?
	`

	hashedPassword, err := m.hashPassword(newPassword)
	newPassword = "" // Overwrite password to avoid accidental usage
	if err != nil {
		slog.Error("error generating password hash", "err", err)
		return err
	}

	return m.execInTransaction(sqlStmt, hashedPassword, playerID)
}
