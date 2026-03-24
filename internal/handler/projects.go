package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"rustymanager/internal/db"
	"rustymanager/internal/store"
)

type Projects struct {
	store *store.Store
}

func NewProjects(s *store.Store) *Projects {
	return &Projects{store: s}
}

func (h *Projects) Index(c echo.Context) error {
	projects, err := h.store.Queries().ListProjects(context.Background())
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "projects/index.html", map[string]any{
		"Projects": projects,
	})
}

func (h *Projects) New(c echo.Context) error {
	return c.Render(http.StatusOK, "projects/new.html", nil)
}

func (h *Projects) Create(c echo.Context) error {
	params := db.CreateProjectParams{
		Name:        c.FormValue("name"),
		Description: c.FormValue("description"),
		Status:      "active",
	}
	if _, err := h.store.Queries().CreateProject(context.Background(), params); err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/projects")
}

func (h *Projects) Show(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	project, err := h.store.Queries().GetProject(context.Background(), id)
	if err != nil {
		return echo.ErrNotFound
	}
	items, err := h.store.Queries().ListKanbanItemsByProject(context.Background(), id)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "projects/show.html", map[string]any{
		"Project": project,
		"Items":   items,
	})
}

func (h *Projects) Edit(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	project, err := h.store.Queries().GetProject(context.Background(), id)
	if err != nil {
		return echo.ErrNotFound
	}
	return c.Render(http.StatusOK, "projects/edit.html", map[string]any{
		"Project": project,
	})
}

func (h *Projects) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	params := db.UpdateProjectParams{
		ID:          id,
		Name:        c.FormValue("name"),
		Description: c.FormValue("description"),
		Status:      c.FormValue("status"),
	}
	if _, err := h.store.Queries().UpdateProject(context.Background(), params); err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/projects")
}

func (h *Projects) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.store.Queries().DeleteProject(context.Background(), id); err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/projects")
}
