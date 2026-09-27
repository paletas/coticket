package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/paletas/coticket/internal/config"
	"github.com/paletas/coticket/internal/mcp"
	"github.com/paletas/coticket/internal/service"
	"github.com/paletas/coticket/internal/store"
)

// version can be set at build time with -ldflags "-X main.version=x.y.z"
// Default to 0.1.0-dev if not set
var version = "0.1.0-dev"

func main() {
	repoPath := flag.String("repo-path", "", "Target repository path used for DB scoping")
	dbPath := flag.String("db-path", "", "SQLite DB path override (must be within repo path)")
	flag.Parse()

	settings, err := config.Load(*repoPath, *dbPath)
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	log.Printf("coticket repo scope: %s", settings.RepoPath)
	log.Printf("coticket database path: %s", settings.DBPath)

	dbStore, err := store.NewSQLiteStore(settings.DBPath)
	if err != nil {
		log.Fatalf("failed to initialize storage: %v", err)
	}
	defer func() {
		_ = dbStore.Close()
	}()

	lifecycle := service.NewLifecycleService(dbStore)
	server := mcp.NewServer(lifecycle, os.Stdin, os.Stdout, version)
	if err = server.Run(context.Background()); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
