package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/paletas/coticket/internal/domain"
	"github.com/paletas/coticket/internal/store"
)

type TicketStore interface {
	CreateTicket(ctx context.Context, ticket domain.Ticket) error
	GetTicket(ctx context.Context, ticketID string) (domain.Ticket, error)
	UpdateTicket(ctx context.Context, ticket domain.Ticket) error
	ListReady(ctx context.Context, limit int, typeFilter *domain.TicketType) ([]domain.Ticket, error)
	AppendEvent(ctx context.Context, ticketID, eventType, actor string, payload any) error
}

type LifecycleService struct {
	store TicketStore
}

func NewLifecycleService(ticketStore TicketStore) *LifecycleService {
	return &LifecycleService{store: ticketStore}
}

type CreateTicketInput struct {
	Type        string
	Title       string
	Description string
	CreatedBy   string
	ParentID    string
}

func (s *LifecycleService) CreateTicket(ctx context.Context, input CreateTicketInput) (domain.Ticket, error) {
	tType, err := domain.ParseTicketType(input.Type)
	if err != nil {
		return domain.Ticket{}, err
	}
	if strings.TrimSpace(input.Title) == "" {
		return domain.Ticket{}, errors.New("title is required")
	}
	if strings.TrimSpace(input.CreatedBy) == "" {
		return domain.Ticket{}, errors.New("created_by is required")
	}

	now := time.Now().UTC()
	ticket := domain.Ticket{
		ID:          uuid.NewString(),
		Type:        tType,
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		Status:      domain.StatusTodo,
		Assignee:    "",
		ParentID:    strings.TrimSpace(input.ParentID),
		CreatedBy:   strings.TrimSpace(input.CreatedBy),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err = s.store.CreateTicket(ctx, ticket); err != nil {
		return domain.Ticket{}, err
	}
	if err = s.store.AppendEvent(ctx, ticket.ID, "ticket_created", ticket.CreatedBy, map[string]any{
		"type":   ticket.Type,
		"status": ticket.Status,
	}); err != nil {
		return domain.Ticket{}, err
	}
	return ticket, nil
}

func (s *LifecycleService) GetTicket(ctx context.Context, ticketID string) (domain.Ticket, error) {
	if strings.TrimSpace(ticketID) == "" {
		return domain.Ticket{}, errors.New("id is required")
	}
	return s.store.GetTicket(ctx, strings.TrimSpace(ticketID))
}

func (s *LifecycleService) ListReady(ctx context.Context, limit int, typeFilter string) ([]domain.Ticket, error) {
	var filter *domain.TicketType
	if strings.TrimSpace(typeFilter) != "" {
		tType, err := domain.ParseTicketType(typeFilter)
		if err != nil {
			return nil, err
		}
		filter = &tType
	}
	return s.store.ListReady(ctx, limit, filter)
}

func (s *LifecycleService) ClaimTicket(ctx context.Context, ticketID, owner string, force bool) (domain.Ticket, error) {
	if strings.TrimSpace(owner) == "" {
		return domain.Ticket{}, errors.New("owner is required")
	}

	ticket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if ticket.Status == domain.StatusDone {
		return domain.Ticket{}, errors.New("cannot claim a done ticket")
	}
	if ticket.Assignee != "" && ticket.Assignee != owner && !force {
		return domain.Ticket{}, fmt.Errorf("ticket already assigned to %s", ticket.Assignee)
	}

	ticket.Assignee = strings.TrimSpace(owner)
	ticket.UpdatedAt = time.Now().UTC()
	if err = s.store.UpdateTicket(ctx, ticket); err != nil {
		return domain.Ticket{}, err
	}
	if err = s.store.AppendEvent(ctx, ticket.ID, "ticket_claimed", owner, map[string]any{
		"force": force,
	}); err != nil {
		return domain.Ticket{}, err
	}
	return ticket, nil
}

func (s *LifecycleService) UpdateStatus(ctx context.Context, ticketID, statusValue, actor string) (domain.Ticket, error) {
	if strings.TrimSpace(actor) == "" {
		return domain.Ticket{}, errors.New("updated_by is required")
	}
	newStatus, err := domain.ParseStatus(statusValue)
	if err != nil {
		return domain.Ticket{}, err
	}

	ticket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if !domain.CanTransition(ticket.Status, newStatus) {
		return domain.Ticket{}, fmt.Errorf("invalid status transition: %s -> %s", ticket.Status, newStatus)
	}

	ticket.Status = newStatus
	ticket.UpdatedAt = time.Now().UTC()
	if err = s.store.UpdateTicket(ctx, ticket); err != nil {
		return domain.Ticket{}, err
	}
	if err = s.store.AppendEvent(ctx, ticket.ID, "status_updated", actor, map[string]any{
		"status": newStatus,
	}); err != nil {
		return domain.Ticket{}, err
	}
	return ticket, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, store.ErrNotFound)
}
