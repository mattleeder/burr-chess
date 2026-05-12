// generated 2025-03-14, Mozilla Guideline v5.7, Go 1.23.3, intermediate config
// https://ssl-config.mozilla.org/#server=go&version=1.23.3&config=intermediate&guideline=5.7

package main

import (
	"burrchess/internal/models"
	"context"
	"database/sql"
	"encoding/hex"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"github.com/alexedwards/scs/sqlite3store" //Sqlite3 ?
	"github.com/alexedwards/scs/v2"
)

type application struct {
	// errorLog       *log.Logger
	// infoLog        *log.Logger
	// perfLog        *log.Logger
	// debugLog       *log.Logger
	logger         *slog.Logger
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
	resetDB := flag.Bool("reset-db", false, "Drop and recreate all database tables (destroys all data)")

	flag.Parse()

	var logger *slog.Logger

	if os.Getenv("ENV") == "production" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		}))
	}

	models.InitDatabase(*dbDriverName, *dbDataSourceName, *resetDB)
	db, err := sql.Open(*dbDriverName, *dbDataSourceName)
	if err != nil {
		logger.Error("failed to open database", "err", err)
		os.Exit(1)
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
		logger.Error("failed to read busy timeout")
		os.Exit(1)
	}

	logger.Info("Busy timeout ms", "busyTimeout", busyTimeout)

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
		logger.Error("SECRET_KEY environment variable not set")
		os.Exit(1)
	}

	secretKey, err := hex.DecodeString(secretKeyHex)
	if err != nil {
		logger.Error("SECRET_KEY must be a valid hex string")
		os.Exit(1)
	}

	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		logger.Error("ALLOWED_ORIGIN environment variable not set")
		os.Exit(1)
	}

	app = &application{
		logger:         logger,
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
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		Handler:      app.routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go matchmakingService()
	go app.cleanupRateLimiters()

	go func() {
		app.logger.Info(
			"Starting server",
			"addr", *addrFlag,
			"allowedOrigin", allowedOrigin,
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	app.logger.Info(
		"Shutting down server",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		app.logger.Error("server shutdown error", "err", err)
	}

	app.logger.Info("Draining DB task queue...")
	models.DBTaskQueue.Drain()
	app.logger.Info("Shutdown complete")
}
