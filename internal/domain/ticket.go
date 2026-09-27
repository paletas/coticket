package domain

import (
	"fmt"
	"strings"
	"time"
)

type TicketType string

const (
	TicketTypeEpic  TicketType = "epic"
	TicketTypeStory TicketType = "story"
	TicketTypeBug   TicketType = "bug"
)

type TicketStatus string

const (
	StatusTodo       TicketStatus = "todo"
	StatusInProgress TicketStatus = "in_progress"
	StatusBlocked    TicketStatus = "blocked"
	StatusDone       TicketStatus = "done"
)

type Ticket struct {
	ID          string       `json:"id"`
	Type        TicketType   `json:"type"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      TicketStatus `json:"status"`
	Assignee    string       `json:"assignee,omitempty"`
	ParentID    string       `json:"parent_id,omitempty"`
	CreatedBy   string       `json:"created_by"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func ParseTicketType(value string) (TicketType, error) {
	switch TicketType(strings.ToLower(strings.TrimSpace(value))) {
	case TicketTypeEpic, TicketTypeStory, TicketTypeBug:
		return TicketType(strings.ToLower(strings.TrimSpace(value))), nil
	default:
		return "", fmt.Errorf("invalid ticket type: %s", value)
	}
}

func ParseStatus(value string) (TicketStatus, error) {
	switch TicketStatus(strings.ToLower(strings.TrimSpace(value))) {
	case StatusTodo, StatusInProgress, StatusBlocked, StatusDone:
		return TicketStatus(strings.ToLower(strings.TrimSpace(value))), nil
	default:
		return "", fmt.Errorf("invalid status: %s", value)
	}
}

var AllowedTransitions = map[TicketStatus]map[TicketStatus]bool{
	StatusTodo: {
		StatusInProgress: true,
		StatusBlocked:    true,
		StatusDone:       true,
	},
	StatusInProgress: {
		StatusTodo:    true,
		StatusBlocked: true,
		StatusDone:    true,
	},
	StatusBlocked: {
		StatusTodo:       true,
		StatusInProgress: true,
		StatusDone:       true,
	},
	StatusDone: {
		StatusDone: true,
	},
}

func CanTransition(from, to TicketStatus) bool {
	return AllowedTransitions[from][to]
}
