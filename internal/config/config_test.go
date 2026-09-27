package config

import (
	"path/filepath"
	"testing"
)

func TestLoadUsesRepoScopedDefaultDB(t *testing.T) {
	t.Setenv("COTICKET_REPO_PATH", "")
	t.Setenv("COTICKET_DB_PATH", "")
	t.Setenv("COTICKET_DATA_DIR", "")

	repoPath := t.TempDir()
	settings, err := Load(repoPath, "")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	expectedDBPath := filepath.Join(repoPath, ".coticket", "coticket.db")
	if settings.DBPath != expectedDBPath {
		t.Fatalf("expected DB path %q, got %q", expectedDBPath, settings.DBPath)
	}
}

func TestLoadRejectsDBOutsideRepoScope(t *testing.T) {
	t.Setenv("COTICKET_REPO_PATH", "")
	t.Setenv("COTICKET_DB_PATH", "")
	t.Setenv("COTICKET_DATA_DIR", "")

	repoPath := t.TempDir()
	outsideDBPath := filepath.Join(t.TempDir(), "coticket.db")
	if _, err := Load(repoPath, outsideDBPath); err == nil {
		t.Fatal("expected outside DB path to fail validation")
	}
}
