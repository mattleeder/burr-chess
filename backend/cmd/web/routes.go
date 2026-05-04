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

func withLogSessionSecureCorsChain(handlerFunc http.HandlerFunc) http.Handler {
	return app.withPerfLog(app.logRequest(app.recoverPanic(wrapWithSessionManager(app.sessionManager, secureHeaders(app.corsHeaders(http.HandlerFunc(handlerFunc)))))))
}

func withLogSecureCorsChain(handlerFunc http.HandlerFunc) http.Handler {
	return app.withPerfLog(app.logRequest(app.recoverPanic(secureHeaders(app.corsHeaders(http.HandlerFunc(handlerFunc))))))
}

func (app *application) routes() http.Handler {

	mux := http.NewServeMux()

	mux.Handle("/", withLogSessionSecureCorsChain(rootHandler))
	mux.Handle("POST /getMoves", withLogSessionSecureCorsChain(getChessMovesHandler))
	mux.Handle("POST /joinQueue", withLogSessionSecureCorsChain(joinQueueHandler))
	mux.Handle("/matchroom/{matchID}/ws", withLogSessionSecureCorsChain(serveMatchroomWs))
	mux.Handle("GET /getHighestEloMatch", withLogSessionSecureCorsChain(getHighestEloMatchHandler))
	mux.Handle("POST /register", withLogSessionSecureCorsChain(registerUserHandler))
	mux.Handle("POST /login", withLogSessionSecureCorsChain(loginHandler))
	mux.Handle("POST /logout", withLogSessionSecureCorsChain(logoutHandler))
	mux.Handle("POST /validateSession", withLogSessionSecureCorsChain(validateSessionHandler))
	mux.Handle("GET /getAccountSettings", withLogSessionSecureCorsChain(getUserAccountSettingsHandler))
	mux.Handle("POST /emailChange", withLogSessionSecureCorsChain(updateEmailHandler))
	mux.Handle("POST /passwordChange", withLogSessionSecureCorsChain(updatePasswordHandler))

	mux.Handle("GET /userSearch", withLogSecureCorsChain(userSearchHandler))
	mux.Handle("GET /getTileInfo", withLogSecureCorsChain(getTileInfoHandler))
	mux.Handle("GET /getPastMatches", withLogSecureCorsChain(getPastMatchesListHandler))

	mux.Handle("/listenformatch", app.withPerfLog(app.logRequest(app.recoverPanic(http.HandlerFunc(app.matchFoundSSEHandler)))))

	// Add the pprof routes
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

	return mux
}
