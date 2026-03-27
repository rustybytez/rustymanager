package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"rustymanager/internal/db"
	"rustymanager/internal/store"
)

type contextKey string

const userContextKey contextKey = "mcp_user"

// Handler returns an http.Handler for the MCP server, authenticated by Bearer token.
func Handler(s *store.Store) http.Handler {
	srv := mcpserver.NewMCPServer("rustymanager", "1.0.0")
	registerTools(srv, s)
	h := mcpserver.NewStreamableHTTPServer(srv)
	return withBearerAuth(h, s)
}

func withBearerAuth(next http.Handler, s *store.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		user, err := s.Queries().GetUserByAPIToken(r.Context(), sql.NullString{String: token, Valid: true})
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func registerTools(srv *mcpserver.MCPServer, s *store.Store) {
	srv.AddTool(
		mcp.NewTool("list_projects",
			mcp.WithDescription("List all projects"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projects, err := s.Queries().ListProjects(ctx)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			type projectSummary struct {
				ID     int64  `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status"`
			}
			out := make([]projectSummary, len(projects))
			for i, p := range projects {
				out[i] = projectSummary{ID: p.ID, Name: p.Name, Status: p.Status}
			}
			return jsonResult(out)
		},
	)

	srv.AddTool(
		mcp.NewTool("list_kanban_items",
			mcp.WithDescription("List kanban items for a project"),
			mcp.WithNumber("project_id", mcp.Required(), mcp.Description("Project ID")),
			mcp.WithString("status", mcp.Description("Filter by status: todo, in_progress, done. Omit for all.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := int64(req.GetInt("project_id", 0))
			if projectID == 0 {
				return mcp.NewToolResultError("project_id is required"), nil
			}
			items, err := s.Queries().ListKanbanItemsByProject(ctx, projectID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			filterStatus := req.GetString("status", "")
			type itemSummary struct {
				ID           int64  `json:"id"`
				Title        string `json:"title"`
				Status       string `json:"status"`
				AssigneeName string `json:"assignee_name,omitempty"`
			}
			var out []itemSummary
			for _, item := range items {
				if filterStatus != "" && item.Status != filterStatus {
					continue
				}
				s := itemSummary{ID: item.ID, Title: item.Title, Status: item.Status}
				if item.AssigneeName.Valid {
					s.AssigneeName = item.AssigneeName.String
				}
				out = append(out, s)
			}
			if out == nil {
				out = []itemSummary{}
			}
			return jsonResult(out)
		},
	)

	srv.AddTool(
		mcp.NewTool("create_kanban_item",
			mcp.WithDescription("Create a new kanban item in a project"),
			mcp.WithNumber("project_id", mcp.Required(), mcp.Description("Project ID")),
			mcp.WithString("title", mcp.Required(), mcp.Description("Item title")),
			mcp.WithString("status", mcp.Description("Status: todo (default), in_progress, done")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := int64(req.GetInt("project_id", 0))
			if projectID == 0 {
				return mcp.NewToolResultError("project_id is required"), nil
			}
			title := req.GetString("title", "")
			if title == "" {
				return mcp.NewToolResultError("title is required"), nil
			}
			status := req.GetString("status", "todo")
			if status != "todo" && status != "in_progress" && status != "done" {
				return mcp.NewToolResultError("status must be todo, in_progress, or done"), nil
			}
			item, err := s.Queries().CreateKanbanItem(ctx, db.CreateKanbanItemParams{
				ProjectID: projectID,
				Title:     title,
				Status:    status,
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return jsonResult(map[string]any{"id": item.ID, "title": item.Title, "status": item.Status})
		},
	)

	srv.AddTool(
		mcp.NewTool("update_kanban_status",
			mcp.WithDescription("Update the status of a kanban item"),
			mcp.WithNumber("item_id", mcp.Required(), mcp.Description("Kanban item ID")),
			mcp.WithNumber("project_id", mcp.Required(), mcp.Description("Project ID (used to verify ownership)")),
			mcp.WithString("status", mcp.Required(), mcp.Description("New status: todo, in_progress, done")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			itemID := int64(req.GetInt("item_id", 0))
			projectID := int64(req.GetInt("project_id", 0))
			status := req.GetString("status", "")
			if itemID == 0 || projectID == 0 || status == "" {
				return mcp.NewToolResultError("item_id, project_id, and status are required"), nil
			}
			if status != "todo" && status != "in_progress" && status != "done" {
				return mcp.NewToolResultError("status must be todo, in_progress, or done"), nil
			}
			item, err := s.Queries().GetKanbanItem(ctx, itemID)
			if err != nil {
				return mcp.NewToolResultError("item not found"), nil
			}
			if item.ProjectID != projectID {
				return mcp.NewToolResultError("item does not belong to the specified project"), nil
			}
			updated, err := s.Queries().UpdateKanbanItemStatus(ctx, db.UpdateKanbanItemStatusParams{
				ID:     itemID,
				Status: status,
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return jsonResult(map[string]any{"id": updated.ID, "title": updated.Title, "status": updated.Status})
		},
	)

	srv.AddTool(
		mcp.NewTool("delete_kanban_item",
			mcp.WithDescription("Delete a kanban item"),
			mcp.WithNumber("item_id", mcp.Required(), mcp.Description("Kanban item ID")),
			mcp.WithNumber("project_id", mcp.Required(), mcp.Description("Project ID (used to verify ownership)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			itemID := int64(req.GetInt("item_id", 0))
			projectID := int64(req.GetInt("project_id", 0))
			if itemID == 0 || projectID == 0 {
				return mcp.NewToolResultError("item_id and project_id are required"), nil
			}
			item, err := s.Queries().GetKanbanItem(ctx, itemID)
			if err != nil {
				return mcp.NewToolResultError("item not found"), nil
			}
			if item.ProjectID != projectID {
				return mcp.NewToolResultError("item does not belong to the specified project"), nil
			}
			if err := s.Queries().DeleteKanbanItem(ctx, itemID); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("item %d deleted", itemID)), nil
		},
	)
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
