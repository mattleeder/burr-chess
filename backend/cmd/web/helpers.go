package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gorilla/websocket"
)

func nullStringPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// Errors

var (
	ErrValueTooLong = errors.New("cookie value too long")
	ErrInvalidValue = errors.New("invalid cookie value")
)

func (app *application) serverError(w http.ResponseWriter, err error) {
	app.logger.Error("server error", "err", err, "trace", string(debug.Stack()))
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

func (app *application) websocketError(conn *websocket.Conn, err error) {
	app.logger.Error("websocket error", "err", err, "trace", string(debug.Stack()))
	conn.WriteMessage(websocket.CloseMessage, []byte{})
	conn.Close()
}

// Cookies

func Write(w http.ResponseWriter, cookie http.Cookie) error {
	cookie.Value = base64.URLEncoding.EncodeToString([]byte(cookie.Value))

	if len(cookie.String()) > 4096 {
		return ErrValueTooLong
	}

	http.SetCookie(w, &cookie)

	return nil
}

func Read(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}

	value, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", ErrInvalidValue
	}

	return string(value), nil
}

func WriteSigned(w http.ResponseWriter, cookie http.Cookie, secretKey []byte) error {
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(cookie.Name))
	mac.Write([]byte(cookie.Value))
	signature := mac.Sum(nil)

	cookie.Value = string(signature) + cookie.Value

	return Write(w, cookie)
}

func ReadSigned(r *http.Request, secretKey []byte, name string) (string, error) {
	signedValue, err := Read(r, name)
	if err != nil {
		return "", err
	}

	if len(signedValue) < sha256.Size {
		return "", ErrInvalidValue
	}

	signature := signedValue[:sha256.Size]
	value := signedValue[sha256.Size:]

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(name))
	mac.Write([]byte(value))
	expectedMAC := mac.Sum(nil)

	if !hmac.Equal([]byte(signature), expectedMAC) {
		return "", ErrInvalidValue
	}

	return value, nil
}

func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate CSRF token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (app *application) writeJSON(w http.ResponseWriter, data any) {
	jsonStr, err := json.Marshal(data)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(jsonStr); err != nil {
		app.logger.Warn("failed to write JSON response", "err", err)
	}
}

// sessionPlayerID returns the playerID stored in the session.
// If it is absent it writes a 401 and returns false.
func (app *application) sessionPlayerID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	if !app.sessionManager.Exists(r.Context(), "playerID") {
		app.clientError(w, http.StatusUnauthorized)
		return 0, false
	}
	return app.sessionManager.GetInt64(r.Context(), "playerID"), true
}

// sessionPlayer returns the playerID and username stored in the session.
// If either is absent it writes a 401 and returns false.
func (app *application) sessionPlayer(w http.ResponseWriter, r *http.Request) (int64, string, bool) {
	if !app.sessionManager.Exists(r.Context(), "playerID") || !app.sessionManager.Exists(r.Context(), "username") {
		app.clientError(w, http.StatusUnauthorized)
		return 0, "", false
	}
	return app.sessionManager.GetInt64(r.Context(), "playerID"),
		app.sessionManager.GetString(r.Context(), "username"),
		true
}

func (app *application) withPerfLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			app.logger.Info("perf", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
		}()
		next.ServeHTTP(w, r)
	})
}
