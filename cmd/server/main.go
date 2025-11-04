package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rohits-web03/cryptodrop/internal/api"
	"github.com/rohits-web03/cryptodrop/internal/config"
	"github.com/rohits-web03/cryptodrop/internal/repositories"
)

func main() {
	// Connect to database
	repositories.ConnectDatabase()
	// Initialize R2
	err := repositories.InitR2(
		config.Envs.R2.AccessKeyID,
		config.Envs.R2.SecretAccessKey,
		config.Envs.R2.AccountID,
		config.Envs.R2.BucketName,
		config.Envs.R2.Region,
	)
	if err != nil {
		log.Fatalf("failed to init R2: %v", err)
	}

	const defaultPort = "8080"
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	mux := api.SetupRouter()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
		// Timeouts prevent resource exhaustion from slow clients
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Starting CryptoDrop server on port: %s", port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on port %s: %v", port, err)
	}
}
