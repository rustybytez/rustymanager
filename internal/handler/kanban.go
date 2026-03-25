package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"rustymanager/internal/db"
	"rustymanager/internal/store"
)

type Kanban struct {
	store *store.Store
}

func NewKanban(s *store.Store) *Kanban {
	return &Kanban{store: s}
}

func (h *Kanban) New(c echo.Context) error {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	users, err := h.store.Queries().ListUsers(context.Background())
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "kanban/new.html", map[string]any{
		"ProjectID": projectID,
		"Users":     users,
	})
}

func (h *Kanban) Create(c echo.Context) error {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}

	status := c.FormValue("status")
	if status == "" {
		status = "todo"
	}

	var assigneeID sql.NullInt64
	if v := c.FormValue("assignee_id"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			assigneeID = sql.NullInt64{Int64: n, Valid: true}
		}
	}

	params := db.CreateKanbanItemParams{
		ProjectID:  projectID,
		Title:      c.FormValue("title"),
		AssigneeID: assigneeID,
		Status:     status,
	}
	if _, err := h.store.Queries().CreateKanbanItem(context.Background(), params); err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/projects/%d", projectID))
}

func (h *Kanban) UpdateStatus(c echo.Context) error {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	itemID, err := strconv.ParseInt(c.Param("itemID"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}

	params := db.UpdateKanbanItemStatusParams{
		ID:     itemID,
		Status: c.FormValue("status"),
	}
	if _, err := h.store.Queries().UpdateKanbanItemStatus(context.Background(), params); err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/projects/%d", projectID))
}

func (h *Kanban) Delete(c echo.Context) error {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	itemID, err := strconv.ParseInt(c.Param("itemID"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}

	if err := h.store.Queries().DeleteKanbanItem(context.Background(), itemID); err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/projects/%d", projectID))
}

func (h *Kanban) DeleteAllDone(c echo.Context) error {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.store.Queries().SoftDeleteDoneKanbanItems(context.Background(), projectID); err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/projects/%d", projectID))
}
