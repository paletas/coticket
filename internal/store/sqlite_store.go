package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/paletas/coticket/internal/domain"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("ticket not found")

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) initSchema() error {
	const schema = `
CREATE TABLE IF NOT EXISTS tickets (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  status TEXT NOT NULL,
  assignee TEXT NOT NULL DEFAULT '',
  parent_id TEXT NOT NULL DEFAULT '',
  created_by TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tickets_ready ON tickets(status, assignee, updated_at);

CREATE TABLE IF NOT EXISTS ticket_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ticket_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  actor TEXT NOT NULL,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL
);
`
	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}
	return nil
}

func (s *SQLiteStore) CreateTicket(ctx context.Context, ticket domain.Ticket) error {
	const query = `
INSERT INTO tickets (id, type, title, description, status, assignee, parent_id, created_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	_, err := s.db.ExecContext(
		ctx,
		query,
		ticket.ID,
		string(ticket.Type),
		ticket.Title,
		ticket.Description,
		string(ticket.Status),
		ticket.Assignee,
		ticket.ParentID,
		ticket.CreatedBy,
		ticket.CreatedAt.UTC().Format(time.RFC3339Nano),
		ticket.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert ticket: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetTicket(ctx context.Context, ticketID string) (domain.Ticket, error) {
	const query = `
SELECT id, type, title, description, status, assignee, parent_id, created_by, created_at, updated_at
FROM tickets WHERE id = ?
`
	row := s.db.QueryRowContext(ctx, query, ticketID)
	ticket, err := scanTicket(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Ticket{}, ErrNotFound
		}
		return domain.Ticket{}, fmt.Errorf("get ticket: %w", err)
	}
	return ticket, nil
}

func (s *SQLiteStore) UpdateTicket(ctx context.Context, ticket domain.Ticket) error {
	const query = `
UPDATE tickets
SET title = ?, description = ?, status = ?, assignee = ?, parent_id = ?, created_by = ?, created_at = ?, updated_at = ?
WHERE id = ?
`
	result, err := s.db.ExecContext(
		ctx,
		query,
		ticket.Title,
		ticket.Description,
		string(ticket.Status),
		ticket.Assignee,
		ticket.ParentID,
		ticket.CreatedBy,
		ticket.CreatedAt.UTC().Format(time.RFC3339Nano),
		ticket.UpdatedAt.UTC().Format(time.RFC3339Nano),
		ticket.ID,
	)
	if err != nil {
		return fmt.Errorf("update ticket: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLiteStore) ListReady(ctx context.Context, limit int, typeFilter *domain.TicketType) ([]domain.Ticket, error) {
	if limit <= 0 {
		limit = 25
	}

	base := `
SELECT id, type, title, description, status, assignee, parent_id, created_by, created_at, updated_at
FROM tickets
WHERE status = ? AND assignee = ''
`
	args := []any{string(domain.StatusTodo)}
	if typeFilter != nil {
		base += " AND type = ?"
		args = append(args, string(*typeFilter))
	}
	base += " ORDER BY updated_at ASC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, fmt.Errorf("list ready tickets: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Ticket, 0, limit)
	for rows.Next() {
		ticket, scanErr := scanTicket(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan ready ticket: %w", scanErr)
		}
		result = append(result, ticket)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ready tickets: %w", err)
	}
	return result, nil
}

func (s *SQLiteStore) AppendEvent(ctx context.Context, ticketID, eventType, actor string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}
	const query = `
INSERT INTO ticket_events (ticket_id, event_type, actor, payload, created_at)
VALUES (?, ?, ?, ?, ?)
`
	_, err = s.db.ExecContext(
		ctx,
		query,
		ticketID,
		eventType,
		actor,
		string(encoded),
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert ticket event: %w", err)
	}
	return nil
}

func scanTicket(scan func(dest ...any) error) (domain.Ticket, error) {
	var (
		ticket      domain.Ticket
		typeRaw     string
		statusRaw   string
		createdRaw  string
		updatedRaw  string
		assigneeRaw string
		parentRaw   string
	)

	err := scan(
		&ticket.ID,
		&typeRaw,
		&ticket.Title,
		&ticket.Description,
		&statusRaw,
		&assigneeRaw,
		&parentRaw,
		&ticket.CreatedBy,
		&createdRaw,
		&updatedRaw,
	)
	if err != nil {
		return domain.Ticket{}, err
	}

	tType, err := domain.ParseTicketType(typeRaw)
	if err != nil {
		return domain.Ticket{}, err
	}
	status, err := domain.ParseStatus(statusRaw)
	if err != nil {
		return domain.Ticket{}, err
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdRaw)
	if err != nil {
		return domain.Ticket{}, fmt.Errorf("parse created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedRaw)
	if err != nil {
		return domain.Ticket{}, fmt.Errorf("parse updated_at: %w", err)
	}

	ticket.Type = tType
	ticket.Status = status
	ticket.Assignee = assigneeRaw
	ticket.ParentID = parentRaw
	ticket.CreatedAt = createdAt
	ticket.UpdatedAt = updatedAt
	return ticket, nil
}
