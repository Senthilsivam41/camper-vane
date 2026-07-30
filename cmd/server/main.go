package main

import (
	"log"
	"net/http"
	"os"

	"camper-vane/internal/api"
	"camper-vane/internal/auth"
	"camper-vane/internal/db"
	"camper-vane/internal/proxy"
	"camper-vane/internal/router"
)

func main() {
	if err := auth.InitFromEnv(); err != nil {
		log.Fatalf("Auth configuration error: %v", err)
	}
	proxy.InitCredentialsFromEnv()

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "camper_vane.db"
	}

	repo, err := db.NewSQLiteRepo(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer repo.Close()

	oauth := auth.NewOAuthManagerFromEnv()
	routerEngine := router.NewRouter(repo, repo)
	authHandler := api.NewAuthHandler(repo, oauth)
	userHandler := api.NewUserHandler(repo)
	chatHandler := api.NewChatStreamHandler(repo, repo, routerEngine)

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("/api/v1/auth/callback", authHandler.HandleCallback)
	mux.HandleFunc("/api/v1/auth/me", authHandler.RequireAuth(authHandler.HandleMe))
	mux.HandleFunc("/api/v1/auth/logout", authHandler.HandleLogout)

	mux.HandleFunc("/api/v1/user/config", authHandler.RequireAuth(userHandler.HandleUserConfig))
	mux.HandleFunc("/api/v1/chat/stream", authHandler.RequireAuth(chatHandler.HandleStream))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s (mock_auth=%v)...", port, oauth.AllowMock())
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}
