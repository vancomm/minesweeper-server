package main

import "net/http"

func buildHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/register", handleRegister)
	mux.HandleFunc("POST /v1/login", handleLogin)
	mux.HandleFunc("POST /v1/logout", handleLogout)

	mux.HandleFunc("GET /v1/status", handleStatus)
	mux.HandleFunc("GET /v1/records", handleGetRecords)
	mux.HandleFunc("GET /v1/myrecords", handleGetOwnRecords)
	// mux.HandleFunc("GET /v1/frecords", handleFilteredRecords)

	mux.HandleFunc("POST /v1/game", handleNewGame)
	mux.HandleFunc("GET /v1/game/{id}", handleGetGame)
	mux.HandleFunc("POST /v1/game/{id}/open", handleOpen)
	mux.HandleFunc("POST /v1/game/{id}/flag", handleFlag)
	mux.HandleFunc("POST /v1/game/{id}/chord", handleChord)
	mux.HandleFunc("POST /v1/game/{id}/reveal", handleReveal)
	mux.HandleFunc("POST /v1/game/{id}/batch", handleBatch)

	mux.HandleFunc("/v1/game/{id}/connect", handleConnectWs)

	handler := useMiddleware(mux,
		corsMiddleware,
		authMiddleware,
		loggingMiddleware,
	)

	return handler
}
