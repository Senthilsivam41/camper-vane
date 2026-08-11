package main

import (
	"log"
	"net/http"
	"os"

	"camper-vane/internal/api"
	"camper-vane/internal/auth"
	"camper-vane/internal/db"
	"camper-vane/internal/proxy"
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

	handler := api.NewHTTPHandler(api.ServerDeps{
		Store: store,
		OAuth: auth.NewOAuthManagerFromEnv(),
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s...", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}
