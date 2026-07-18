package main

import (
	"log"
	"net/http"
	"os"

	"camper-vane/internal/api"
	"camper-vane/internal/db"
)

func main() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "camper_vane.db"
	}

	repo, err := db.NewSQLiteRepo(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer repo.Close()

	authHandler := api.NewAuthHandler(repo)
	userHandler := api.NewUserHandler(repo)
	chatHandler := api.NewChatStreamHandler(repo, repo)

	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("/api/v1/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("/api/v1/auth/callback", authHandler.HandleCallback)
	mux.HandleFunc("/api/v1/auth/me", authHandler.RequireAuth(authHandler.HandleMe))
	mux.HandleFunc("/api/v1/auth/logout", authHandler.HandleLogout)

	// User config route
	mux.HandleFunc("/api/v1/user/config", authHandler.RequireAuth(userHandler.HandleUserConfig))

	// SSE Chat Stream route
	mux.HandleFunc("/api/v1/chat/stream", authHandler.RequireAuth(chatHandler.HandleStream))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s...", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}
