// generated 2025-03-14, Mozilla Guideline v5.7, Go 1.23.3, intermediate config
// https://ssl-config.mozilla.org/#server=go&version=1.23.3&config=intermediate&guideline=5.7

package main

import (
	"burrchess/internal/models"
	"database/sql"
	"encoding/hex"
	"flag"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/alexedwards/scs/sqlite3store" //Sqlite3 ?
	"github.com/alexedwards/scs/v2"
)

type application struct {
	errorLog       *log.Logger
	infoLog        *log.Logger
	perfLog        *log.Logger
	debugLog       *log.Logger
	secretKey      []byte
	liveMatches    *models.LiveMatchModel
	pastMatches    *models.PastMatchModel
	users          *models.UserModel
	userRatings    *models.UserRatingsModel
	dbTaskQueue    *models.TaskQueue
	sessionManager *scs.SessionManager
	allowedOrigin  string
	rateLimiters   sync.Map
}

var app *application

func main() {
	// addr := flag.String("addr", ":8080", "HTTPS network address")
	dbDriverName := flag.String("db", "sqlite", "Database Driver Name")
	// dbDataSourceName := flag.String("dsn", "file:chess_site.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", "Database Data Source Name")

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	dsn := os.Getenv("DSN")
	if dsn == "" {
		dsn = "file:chess_site.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	}

	addrFlag := flag.String("addr", addr, "HTTPS network address")
	dbDataSourceName := flag.String("dsn", dsn, "Database Data Source Name")

	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Llongfile)
	perfLog := log.New(os.Stdout, "PERF\t", log.Lshortfile)
	debugLog := log.New(os.Stdout, "DEBUG\t", log.Lshortfile)

	models.InitDatabase(*dbDriverName, *dbDataSourceName)
	db, err := sql.Open(*dbDriverName, *dbDataSourceName)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0)

	// Set busy_timeout to 2 seconds
	// _, err = db.Exec("PRAGMA busy_timeout = 5000;")
	// if err != nil {
	// 	errorLog.Fatal(err)
	// }
	var busyTimeout int
	err = db.QueryRow("SELECT * FROM pragma_busy_timeout()").Scan(&busyTimeout)
	if err != nil {
		errorLog.Fatal(err)
	}
	infoLog.Printf("Busy timeout %d ms\n", busyTimeout)

	// Write-Ahead Logging
	// _, err = db.Exec("PRAGMA journal_mode=WAL;")
	// if err != nil {
	// 	errorLog.Fatal(err)
	// }

	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.New(db)
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.IdleTimeout = 1 * time.Hour
	sessionManager.HashTokenInStore = true
	sessionManager.Cookie = scs.SessionCookie{
		Name:     "session",
		Path:     "/",
		HttpOnly: true,
		Persist:  false,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	}

	secretKeyHex := os.Getenv("SECRET_KEY")
	if secretKeyHex == "" {
		errorLog.Fatal("SECRET_KEY environment variable not set")
	}

	secretKey, err := hex.DecodeString(secretKeyHex)
	if err != nil {
		errorLog.Fatal("SECRET_KEY must be a valid hex string: ", err)
	}

	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		errorLog.Fatal("ALLOWED_ORIGIN environment variable not set")
	}

	app = &application{
		errorLog:       errorLog,
		infoLog:        infoLog,
		perfLog:        perfLog,
		debugLog:       debugLog,
		secretKey:      secretKey,
		liveMatches:    &models.LiveMatchModel{DB: db},
		pastMatches:    &models.PastMatchModel{DB: db},
		users:          &models.UserModel{DB: db},
		userRatings:    &models.UserRatingsModel{DB: db},
		dbTaskQueue:    models.DBTaskQueue,
		sessionManager: sessionManager,
		allowedOrigin:  allowedOrigin,
	}

	srv := &http.Server{
		Addr:         *addrFlag,
		ErrorLog:     errorLog,
		Handler:      app.routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go matchmakingService()
	go app.cleanupRateLimiters()

	app.infoLog.Printf("Starting server on %s", addr)
	err = srv.ListenAndServe()
	if err != nil {
		errorLog.Fatal(err)
	}
}
