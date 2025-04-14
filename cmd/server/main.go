package main

import (
	"log"

	"CacheFlow/internal/server"
)

func main() {
	srv, err := server.New(":6379")
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
