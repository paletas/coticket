package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/paletas/coticket/internal/store"
)

func setupService(t *testing.T) *LifecycleService {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "tickets.db")
	dbStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = dbStore.Close()
	})
	return NewLifecycleService(dbStore)
}

func TestCreateAndListReady(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	created, err := svc.CreateTicket(ctx, CreateTicketInput{
		Type:        "story",
		Title:       "Implement queue",
		Description: "add list-ready support",
		CreatedBy:   "agent-a",
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	ready, err := svc.ListReady(ctx, 10, "")
	if err != nil {
		t.Fatalf("list ready: %v", err)
	}
	if len(ready) != 1 {
		t.Fatalf("expected 1 ready ticket, got %d", len(ready))
	}
	if ready[0].ID != created.ID {
		t.Fatalf("expected ready ticket id %s, got %s", created.ID, ready[0].ID)
	}
}

func TestClaimTicketOwnershipRules(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	created, err := svc.CreateTicket(ctx, CreateTicketInput{
		Type:      "bug",
		Title:     "Fix race",
		CreatedBy: "agent-a",
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	if _, err = svc.ClaimTicket(ctx, created.ID, "agent-a", false); err != nil {
		t.Fatalf("claim by first owner: %v", err)
	}
	if _, err = svc.ClaimTicket(ctx, created.ID, "agent-b", false); err == nil {
		t.Fatal("expected claim to fail without force")
	}
	if _, err = svc.ClaimTicket(ctx, created.ID, "agent-b", true); err != nil {
		t.Fatalf("force claim failed: %v", err)
	}
}

func TestStatusTransitionRules(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	created, err := svc.CreateTicket(ctx, CreateTicketInput{
		Type:      "epic",
		Title:     "Launch API",
		CreatedBy: "agent-a",
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	updated, err := svc.UpdateStatus(ctx, created.ID, "in_progress", "agent-a")
	if err != nil {
		t.Fatalf("expected valid status transition, got %v", err)
	}
	if updated.Status != "in_progress" {
		t.Fatalf("expected in_progress, got %s", updated.Status)
	}

	updated, err = svc.UpdateStatus(ctx, created.ID, "done", "agent-a")
	if err != nil {
		t.Fatalf("expected transition to done, got %v", err)
	}
	if updated.Status != "done" {
		t.Fatalf("expected done, got %s", updated.Status)
	}

	if _, err = svc.UpdateStatus(ctx, created.ID, "todo", "agent-a"); err == nil {
		t.Fatal("expected transition from done to todo to fail")
	}
}
