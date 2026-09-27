package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Settings struct {
	RepoPath string
	DataDir  string
	DBPath   string
}

func Load(repoPathOverride, dbPathOverride string) (Settings, error) {
	repoPath := strings.TrimSpace(repoPathOverride)
	if repoPath == "" {
		repoPath = strings.TrimSpace(os.Getenv("COTICKET_REPO_PATH"))
	}
	if repoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return Settings{}, fmt.Errorf("resolve working directory: %w", err)
		}
		repoPath = cwd
	}

	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return Settings{}, fmt.Errorf("resolve repository path: %w", err)
	}

	dbPath := strings.TrimSpace(dbPathOverride)
	if dbPath == "" {
		dbPath = strings.TrimSpace(os.Getenv("COTICKET_DB_PATH"))
	}

	dataDir := strings.TrimSpace(os.Getenv("COTICKET_DATA_DIR"))
	if dataDir == "" {
		dataDir = filepath.Join(absRepoPath, ".coticket")
	} else if !filepath.IsAbs(dataDir) {
		dataDir = filepath.Join(absRepoPath, dataDir)
	}

	if dbPath == "" {
		dbPath = filepath.Join(dataDir, "coticket.db")
	} else if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(absRepoPath, dbPath)
	}

	absDBPath, err := filepath.Abs(dbPath)
	if err != nil {
		return Settings{}, fmt.Errorf("resolve database path: %w", err)
	}

	settings := Settings{
		RepoPath: absRepoPath,
		DataDir:  dataDir,
		DBPath:   absDBPath,
	}
	settings.DataDir = filepath.Dir(settings.DBPath)
	if err = settings.Validate(); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func (s Settings) Validate() error {
	if strings.TrimSpace(s.RepoPath) == "" {
		return fmt.Errorf("repository path is required")
	}
	if strings.TrimSpace(s.DBPath) == "" {
		return fmt.Errorf("database path is required")
	}

	absRepoPath, err := filepath.Abs(s.RepoPath)
	if err != nil {
		return fmt.Errorf("resolve repository path: %w", err)
	}
	absDBPath, err := filepath.Abs(s.DBPath)
	if err != nil {
		return fmt.Errorf("resolve database path: %w", err)
	}

	rel, err := filepath.Rel(absRepoPath, absDBPath)
	if err != nil {
		return fmt.Errorf("db path must be within repository scope: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("db path %q is outside repository scope %q", absDBPath, absRepoPath)
	}

	return nil
}
