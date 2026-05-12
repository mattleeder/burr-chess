package main

import (
	"net/http"
	"net/http/pprof"

	"github.com/alexedwards/scs/v2"
)

func wrapWithSessionManager(sm *scs.SessionManager, handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// sessionID := sm.Token(r.Context())
		// app.infoLog.Println("Session ID:", sessionID)
		sm.LoadAndSave(handler).ServeHTTP(w, r)
	}
}

func (app *application) withLogSessionSecureCorsChain(handlerFunc http.HandlerFunc) http.Handler {
	return app.withPerfLog(
		app.logRequest(
			app.recoverPanic(
				app.corsHeaders(
					app.rateLimit(
						wrapWithSessionManager(
							app.sessionManager, app.requireSameOrigin(
								secureHeaders(
									http.HandlerFunc(handlerFunc)))))))))
}

func (app *application) withLogSecureCorsChain(handlerFunc http.HandlerFunc) http.Handler {
	return app.withPerfLog(
		app.logRequest(
			app.recoverPanic(
				app.corsHeaders(
					app.rateLimit(
						app.requireSameOrigin(
							secureHeaders(
								http.HandlerFunc(handlerFunc))))))))
}

// withLogSessionSecureCorsAuthChain adds a strict auth rate limit on top of the standard chain.
// Use this for login/register endpoints.
func (app *application) withLogSessionSecureCorsAuthChain(handlerFunc http.HandlerFunc) http.Handler {
	return app.withPerfLog(
		app.logRequest(
			app.recoverPanic(
				app.corsHeaders(
					app.rateLimit(
						app.authRateLimit(
							wrapWithSessionManager(
								app.sessionManager, app.requireSameOrigin(
									secureHeaders(
										http.HandlerFunc(handlerFunc))))))))))
}

func (app *application) routes() http.Handler {

	mux := http.NewServeMux()

	mux.Handle("GET /health", requireLocalhost(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })))
	mux.Handle("/", app.withLogSessionSecureCorsChain(app.rootHandler))
	mux.Handle("POST /getMoves", app.withLogSessionSecureCorsChain(app.getChessMovesHandler))
	mux.Handle("POST /joinQueue", app.withLogSessionSecureCorsChain(app.joinQueueHandler))
	mux.Handle("/matchroom/{matchID}/ws", app.withLogSessionSecureCorsChain(app.serveMatchroomWs))
	mux.Handle("GET /getHighestEloMatch", app.withLogSessionSecureCorsChain(app.getHighestEloMatchHandler))
	mux.Handle("POST /register", app.withLogSessionSecureCorsAuthChain(app.registerUserHandler))
	mux.Handle("POST /login", app.withLogSessionSecureCorsAuthChain(app.loginHandler))
	mux.Handle("POST /logout", app.withLogSessionSecureCorsChain(app.logoutHandler))
	mux.Handle("POST /validateSession", app.withLogSessionSecureCorsChain(app.validateSessionHandler))
	mux.Handle("GET /getAccountSettings", app.withLogSessionSecureCorsChain(app.getUserAccountSettingsHandler))
	mux.Handle("POST /emailChange", app.withLogSessionSecureCorsChain(app.updateEmailHandler))
	mux.Handle("POST /passwordChange", app.withLogSessionSecureCorsChain(app.updatePasswordHandler))

	mux.Handle("GET /userSearch", app.withLogSecureCorsChain(app.userSearchHandler))
	mux.Handle("GET /getTileInfo", app.withLogSecureCorsChain(app.getTileInfoHandler))
	mux.Handle("GET /getPastMatches", app.withLogSecureCorsChain(app.getPastMatchesListHandler))

	mux.Handle("/listenformatch", app.withPerfLog(app.logRequest(app.recoverPanic(app.corsHeaders(app.rateLimit(secureHeaders(http.HandlerFunc(app.matchFoundSSEHandler))))))))

	// Add the pprof routes (localhost only)
	mux.Handle("/debug/pprof/", requireLocalhost(http.HandlerFunc(pprof.Index)))
	mux.Handle("/debug/pprof/cmdline", requireLocalhost(http.HandlerFunc(pprof.Cmdline)))
	mux.Handle("/debug/pprof/profile", requireLocalhost(http.HandlerFunc(pprof.Profile)))
	mux.Handle("/debug/pprof/symbol", requireLocalhost(http.HandlerFunc(pprof.Symbol)))
	mux.Handle("/debug/pprof/trace", requireLocalhost(http.HandlerFunc(pprof.Trace)))

	mux.Handle("/debug/pprof/block", requireLocalhost(pprof.Handler("block")))
	mux.Handle("/debug/pprof/goroutine", requireLocalhost(pprof.Handler("goroutine")))
	mux.Handle("/debug/pprof/heap", requireLocalhost(pprof.Handler("heap")))
	mux.Handle("/debug/pprof/threadcreate", requireLocalhost(pprof.Handler("threadcreate")))

	return mux
}
