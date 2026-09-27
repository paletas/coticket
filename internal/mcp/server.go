package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/paletas/coticket/internal/service"
)

type Server struct {
	service  *service.LifecycleService
	protocol *Protocol
	version  string
}

func NewServer(svc *service.LifecycleService, in io.Reader, out io.Writer, version string) *Server {
	return &Server{
		service:  svc,
		protocol: NewProtocol(in, out),
		version:  version,
	}
}

func (s *Server) Run(ctx context.Context) error {
	log.Printf("mcp action=server_start status=ok")
	for {
		msg, err := s.protocol.ReadMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Printf("mcp action=server_stop reason=eof status=ok")
				return nil
			}
			log.Printf("mcp action=read_message status=error error=%q", err.Error())
			return err
		}

		response := s.handleRequest(ctx, msg)
		if response == nil {
			continue
		}
		if err = s.protocol.WriteMessage(response); err != nil {
			log.Printf("mcp action=write_message status=error error=%q", err.Error())
			return err
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, request map[string]any) map[string]any {
	id, hasID := request["id"]
	method, _ := request["method"].(string)
	log.Printf("mcp action=request_received method=%q has_id=%t", method, hasID)

	if method == "notifications/initialized" && !hasID {
		log.Printf("mcp action=notification method=%q status=ignored", method)
		return nil
	}
	if !hasID {
		log.Printf("mcp action=request_ignored reason=missing_id method=%q", method)
		return nil
	}

	switch method {
	case "initialize":
		log.Printf("mcp action=initialize request_id=%q status=ok", fmt.Sprintf("%v", id))
		return successResponse(id, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "coticket-mcp",
				"version": s.version,
			},
		})
	case "tools/list":
		log.Printf("mcp action=tools_list request_id=%q status=ok", fmt.Sprintf("%v", id))
		return successResponse(id, map[string]any{
			"tools": toolSpecs(),
		})
	case "tools/call":
		params, _ := request["params"].(map[string]any)
		if params == nil {
			log.Printf("mcp action=tools_call request_id=%q status=error reason=%q", fmt.Sprintf("%v", id), "missing params")
			return successResponse(id, toolError("missing params"))
		}
		name, _ := params["name"].(string)
		args, _ := params["arguments"].(map[string]any)
		log.Printf("mcp action=tools_call_start request_id=%q tool=%q", fmt.Sprintf("%v", id), name)
		result := s.callTool(ctx, name, args)
		if isToolError(result) {
			log.Printf("mcp action=tools_call_end request_id=%q tool=%q status=error", fmt.Sprintf("%v", id), name)
		} else {
			log.Printf("mcp action=tools_call_end request_id=%q tool=%q status=ok", fmt.Sprintf("%v", id), name)
		}
		return successResponse(id, result)
	default:
		log.Printf("mcp action=request_error request_id=%q method=%q status=error reason=%q", fmt.Sprintf("%v", id), method, "method not found")
		return errorResponse(id, -32601, "method not found")
	}
}

func (s *Server) callTool(ctx context.Context, name string, arguments map[string]any) map[string]any {
	switch name {
	case "ticket_create":
		return s.callCreate(ctx, arguments)
	case "ticket_get":
		return s.callGet(ctx, arguments)
	case "ticket_claim":
		return s.callClaim(ctx, arguments)
	case "ticket_update_status":
		return s.callUpdateStatus(ctx, arguments)
	case "ticket_list_ready":
		return s.callListReady(ctx, arguments)
	default:
		return toolError(fmt.Sprintf("unknown tool: %s", name))
	}
}

func (s *Server) callCreate(ctx context.Context, args map[string]any) map[string]any {
	log.Printf("mcp action=ticket_create_start type=%q created_by=%q", stringArg(args, "type"), stringArg(args, "created_by"))
	ticket, err := s.service.CreateTicket(ctx, service.CreateTicketInput{
		Type:        stringArg(args, "type"),
		Title:       stringArg(args, "title"),
		Description: stringArg(args, "description"),
		CreatedBy:   stringArg(args, "created_by"),
		ParentID:    stringArg(args, "parent_id"),
	})
	if err != nil {
		log.Printf("mcp action=ticket_create_end status=error error=%q", err.Error())
		return toolError(err.Error())
	}
	log.Printf("mcp action=ticket_create_end status=ok ticket_id=%q", ticket.ID)
	return toolSuccess(ticket)
}

func (s *Server) callGet(ctx context.Context, args map[string]any) map[string]any {
	ticketID := stringArg(args, "id")
	log.Printf("mcp action=ticket_get_start ticket_id=%q", ticketID)
	ticket, err := s.service.GetTicket(ctx, ticketID)
	if err != nil {
		log.Printf("mcp action=ticket_get_end status=error ticket_id=%q error=%q", ticketID, err.Error())
		return toolError(err.Error())
	}
	log.Printf("mcp action=ticket_get_end status=ok ticket_id=%q", ticket.ID)
	return toolSuccess(ticket)
}

func (s *Server) callClaim(ctx context.Context, args map[string]any) map[string]any {
	ticketID := stringArg(args, "id")
	owner := stringArg(args, "owner")
	force := boolArg(args, "force")
	log.Printf("mcp action=ticket_claim_start ticket_id=%q owner=%q force=%t", ticketID, owner, force)
	ticket, err := s.service.ClaimTicket(ctx, ticketID, owner, force)
	if err != nil {
		log.Printf("mcp action=ticket_claim_end status=error ticket_id=%q error=%q", ticketID, err.Error())
		return toolError(err.Error())
	}
	log.Printf("mcp action=ticket_claim_end status=ok ticket_id=%q owner=%q", ticket.ID, ticket.Assignee)
	return toolSuccess(ticket)
}

func (s *Server) callUpdateStatus(ctx context.Context, args map[string]any) map[string]any {
	ticketID := stringArg(args, "id")
	statusValue := stringArg(args, "status")
	actor := stringArg(args, "updated_by")
	log.Printf("mcp action=ticket_update_status_start ticket_id=%q status=%q updated_by=%q", ticketID, statusValue, actor)
	ticket, err := s.service.UpdateStatus(ctx, ticketID, statusValue, actor)
	if err != nil {
		log.Printf("mcp action=ticket_update_status_end status=error ticket_id=%q error=%q", ticketID, err.Error())
		return toolError(err.Error())
	}
	log.Printf("mcp action=ticket_update_status_end status=ok ticket_id=%q status=%q", ticket.ID, ticket.Status)
	return toolSuccess(ticket)
}

func (s *Server) callListReady(ctx context.Context, args map[string]any) map[string]any {
	limit := intArg(args, "limit", 25)
	typeFilter := stringArg(args, "type")
	log.Printf("mcp action=ticket_list_ready_start limit=%d type=%q", limit, typeFilter)
	tickets, err := s.service.ListReady(ctx, limit, typeFilter)
	if err != nil {
		log.Printf("mcp action=ticket_list_ready_end status=error error=%q", err.Error())
		return toolError(err.Error())
	}
	log.Printf("mcp action=ticket_list_ready_end status=ok count=%d", len(tickets))
	return toolSuccess(map[string]any{"tickets": tickets})
}

func toolSpecs() []map[string]any {
	return []map[string]any{
		{
			"name":        "ticket_create",
			"description": "Create an epic, story, or bug ticket",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type":        map[string]any{"type": "string", "enum": []string{"epic", "story", "bug"}},
					"title":       map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"created_by":  map[string]any{"type": "string"},
					"parent_id":   map[string]any{"type": "string"},
				},
				"required": []string{"type", "title", "created_by"},
			},
		},
		{
			"name":        "ticket_get",
			"description": "Get a ticket by ID",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "ticket_claim",
			"description": "Claim or reassign ownership of a ticket",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":    map[string]any{"type": "string"},
					"owner": map[string]any{"type": "string"},
					"force": map[string]any{"type": "boolean"},
				},
				"required": []string{"id", "owner"},
			},
		},
		{
			"name":        "ticket_update_status",
			"description": "Update ticket status",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":         map[string]any{"type": "string"},
					"status":     map[string]any{"type": "string", "enum": []string{"todo", "in_progress", "blocked", "done"}},
					"updated_by": map[string]any{"type": "string"},
				},
				"required": []string{"id", "status", "updated_by"},
			},
		},
		{
			"name":        "ticket_list_ready",
			"description": "List ready, unassigned tickets",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 200},
					"type":  map[string]any{"type": "string", "enum": []string{"epic", "story", "bug"}},
				},
			},
		},
	}
}

func successResponse(id any, result map[string]any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
}

func errorResponse(id any, code int, message string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

func toolSuccess(payload any) map[string]any {
	pretty, _ := json.MarshalIndent(payload, "", "  ")
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(pretty)},
		},
		"structuredContent": payload,
	}
}

func toolError(message string) map[string]any {
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": message},
		},
		"isError": true,
	}
}

func isToolError(result map[string]any) bool {
	value, _ := result["isError"].(bool)
	return value
}

func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	value, _ := args[key].(string)
	return value
}

func boolArg(args map[string]any, key string) bool {
	if args == nil {
		return false
	}
	value, _ := args[key].(bool)
	return value
}

func intArg(args map[string]any, key string, fallback int) int {
	if args == nil {
		return fallback
	}
	switch typed := args[key].(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	default:
		return fallback
	}
}
