package models

import (
	"database/sql"
	"errors"
	"math/rand"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const PASSWORD_COST = 16

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
	DB *sql.DB
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
	Email sql.NullString `json:"email"`
}

func hashPassword(password string) (string, error) {
	var passwordBytes = []byte(password)
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword(passwordBytes, PASSWORD_COST)
	return string(hashedPasswordBytes), err
}

func doesPasswordMatch(plaintextPassword string, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plaintextPassword))
	return err == nil
}

func (m *UserModel) execInTransaction(sqlStmt string, args ...any) error {
	tx, err := m.DB.Begin()
	if err != nil {
		app.errorLog.Printf("Error starting transaction: %v\n", err)
		return err
	}

	stmt, err := tx.Prepare(sqlStmt)
	if err != nil {
		app.errorLog.Printf("Error preparing statement: %v\n", err)
		return err
	}
	defer stmt.Close()

	_, err = ExecStatementWithRetry(stmt, args...)
	if err != nil {
		app.errorLog.Printf("Error executing statement: %v\n", err)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			app.errorLog.Printf("unable to rollback: %v", rollbackErr)
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		app.errorLog.Printf("Error committing transaction: %v\n", err)
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

	app.infoLog.Printf("Inserting new user: %v\n", username)

	if username == "" {
		app.errorLog.Printf("Username not given")
		return 0, errors.New("username not given")
	}

	if password == "" {
		app.errorLog.Printf("Password not given")
		return 0, errors.New("password not given")
	}

	var playerID = rand.Int63()
	hashedPassword, err := hashPassword(password)
	password = "" // Overwrite password to avoid accidental usage
	if err != nil {
		app.errorLog.Printf("Error generating password hash: %v\n", err.Error())
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
		app.errorLog.Printf("Error starting transaction: %v\n", err)
		return 0, err
	}

	stmtOne, err = tx.Prepare(stepOne)
	if err != nil {
		app.errorLog.Printf("Error preparing first statement: %v\n", err)
		return 0, err
	}
	defer stmtOne.Close()

	stmtTwo, err = tx.Prepare(stepTwo)
	if err != nil {
		app.errorLog.Printf("Error preparing second statement: %v\n", err)
		return 0, err
	}
	defer stmtTwo.Close()

	_, err = ExecStatementWithRetry(stmtOne, playerID, username, hashedPassword, email)

	if err != nil {
		app.errorLog.Printf("Error inserting new user: %s\n", err.Error())
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			app.errorLog.Printf("insert users: unable to rollback: %s", rollbackErr)
		}
		return 0, err
	}

	_, err = ExecStatementWithRetry(stmtTwo, playerID, username)

	if err != nil {
		app.errorLog.Printf("Error executing second statement: %v\n", err)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			app.errorLog.Printf("insert user_ratings: unable to rollback: %v", rollbackErr)
		}
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		app.errorLog.Printf("Error commiting transaction: %v\n", err)
		return 0, err
	}

	app.infoLog.Printf("Inserted player_id: %v\n", playerID)

	return playerID, nil
}

func (m *UserModel) Authenticate(username string, password string) (playerID int64, authorized bool) {
	app.infoLog.Printf("Checking password for user: %v\n", username)

	sqlStmt := `
	select player_id, password from users where username = ?
	`
	var hashedPassword string
	err := QueryRowWithRetry(m.DB, sqlStmt, []any{username}, []any{&playerID, &hashedPassword})
	if err != nil {
		app.errorLog.Printf("Error getting password for user: %v\n", err.Error())
		return 0, false
	}

	authorized = doesPasswordMatch(password, hashedPassword)

	if !authorized {
		return 0, false
	}

	return playerID, true
}

func (m *UserModel) LogAll() {
	app.infoLog.Println("Users:")

	rows, err := QueryWithRetry(m.DB, "select * from users;")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	defer rows.Close()
	for rows.Next() {
		app.rowsLog.Printf("%v\n", rows)
	}
	err = rows.Err()
	if err != nil {
		app.errorLog.Println(err)
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
		app.errorLog.Printf("Error getting user: %v\n", err.Error())
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
	var email string
	var joinDate int64
	var lastSeen int64

	err := QueryRowWithRetry(m.DB, sqlStmt, []any{query.arg}, []any{&playerID, &username, &password, &email, &joinDate, &lastSeen})
	if err != nil {
		app.errorLog.Printf("Error getting user: %v\n", err.Error())
		return UserServerSide{}, err
	}
	return UserServerSide{
		PlayerID: playerID,
		Username: username,
		Password: password,
		Email:    email,
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
	`

	var output []UserClientSide
	var playerID int64
	var username string
	var joinDate int64
	var lastSeen int64

	rows, err := QueryWithRetry(m.DB, sqlStmt, strings.ToUpper(searchString))
	if err != nil {
		app.errorLog.Printf("Error in SearchForUsers: %s\n", err.Error())
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&playerID, &username, &joinDate, &lastSeen)

		if err != nil {
			app.errorLog.Printf("Error in SearchForUsers: %s\n", err.Error())
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
	SELECT users.player_id, join_date, last_seen, bullet_rating, blitz_rating, rapid_rating, classical_rating
	  FROM users
	 INNER JOIN user_ratings
	    ON users.player_id = user_ratings.player_id
	 WHERE users.username = ?
	`

	var playerID int64
	var joinDate int64
	var lastSeen int64
	var bullet_rating int64
	var blitz_rating int64
	var rapid_rating int64
	var classical_rating int64

	err := QueryRowWithRetry(m.DB, sqlStmt, []any{username}, []any{&playerID, &joinDate, &lastSeen, &bullet_rating, &blitz_rating, &rapid_rating, &classical_rating})
	if err != nil {
		app.errorLog.Printf("Error in GetTileInfoFromPlayerID: %s\n", err.Error())
		return nil, err
	}

	sqlStmt = `
	SELECT Count(past_matches.match_id) as number_of_games
	  FROM users
	 INNER JOIN past_matches
	    ON past_matches.white_player_id = users.player_id
		OR past_matches.black_player_id = users.player_id
	 WHERE users.username = ?
	`

	var numberOfGames int64

	err = QueryRowWithRetry(m.DB, sqlStmt, []any{username}, []any{&numberOfGames})
	if err != nil {
		app.errorLog.Printf("Error in GetTileInfoFromPlayerID gameCount: %s\n", err.Error())
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
			BulletRating:    bullet_rating,
			BlitzRating:     blitz_rating,
			RapidRating:     rapid_rating,
			ClassicalRating: classical_rating,
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
		app.errorLog.Printf("Error getting account settings: %v\n", err.Error())
		return AccountSettings{}, err
	}
	return AccountSettings{
		Email: email,
	}, nil
}

func (m *UserModel) UpdateEmail(playerID int64, newEmail string) error {
	sqlStmt := `
	UPDATE users
	   SET email = ?
	 WHERE player_id = ?
	`

	var validEmail = false
	if newEmail != "" {
		validEmail = true
	}

	updateEmail := sql.NullString{
		String: newEmail,
		Valid:  validEmail,
	}

	return m.execInTransaction(sqlStmt, updateEmail, playerID)
}

func (m *UserModel) UpdatePassword(playerID int64, newPassword string) error {
	sqlStmt := `
	UPDATE users
	   SET password = ?
	 WHERE player_id = ?
	`

	if newPassword == "" {
		app.errorLog.Printf("Password not given")
		return errors.New("password not given")
	}

	hashedPassword, err := hashPassword(newPassword)
	newPassword = "" // Overwrite password to avoid accidental usage
	if err != nil {
		app.errorLog.Printf("Error generating password hash: %v\n", err.Error())
		return err
	}

	return m.execInTransaction(sqlStmt, hashedPassword, playerID)
}
