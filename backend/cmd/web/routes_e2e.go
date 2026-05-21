//go:build e2e

package main

import (
	"burrchess/internal/models"
	"net/http"
)

// registerE2ERoutes adds reset endpoints used by Playwright E2E tests.
// Only compiled when building with -tags e2e.
// requireLocalhost is intentionally omitted: in Docker the test runner is a
// separate container with a non-loopback IP. The routes are safe to expose
// without that guard because they only exist in e2e builds, not production.
func registerE2ERoutes(mux *http.ServeMux, app *application) {
	mux.Handle("GET /resetQueues", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queueMu.Lock()
		queueMap = make(map[queueKey]*QueueData)
		queueMu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	mux.Handle("GET /resetLiveMatches", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := models.DBTaskQueue.EnQueueReturnErrorOnlyTask(func() error {
			return app.liveMatches.DeleteAllLiveMatches()
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	mux.Handle("GET /resetMatchClients", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clients.mu.Lock()
		for _, c := range clients.clients {
			close(c.channel)
		}
		clients.clients = make(map[int64]*Client)
		clients.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	mux.Handle("GET /resetRateLimiters", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.rateLimiters.Range(func(key, _ any) bool {
			app.rateLimiters.Delete(key)
			return true
		})
		w.WriteHeader(http.StatusOK)
	}))
}
