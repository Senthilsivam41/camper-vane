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

	store, err := db.NewStoreFromEnv()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer store.Close()

	oauth := auth.NewOAuthManagerFromEnv()
	routerEngine := router.NewRouter(store, store)
	authHandler := api.NewAuthHandler(store, oauth)
	userHandler := api.NewUserHandler(store)
	chatHandler := api.NewChatStreamHandler(store, store, routerEngine)

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
