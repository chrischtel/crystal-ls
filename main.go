package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"gocry/internal/lsp"
)

// version is set at build time via -ldflags
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("Crystal Language Server %s\n", version)
		return
	}

	// Create a new Crystal LSP server
	server := lsp.NewServer()

	// Start the server
	log.Println("Starting Crystal Language Server...")
	if err := server.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
