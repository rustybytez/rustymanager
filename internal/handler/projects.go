package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"rustymanager/internal/db"
	authmw "rustymanager/internal/middleware"
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
	setProjectCookie(c, id)
	return c.Render(http.StatusOK, "projects/show.html", map[string]any{
		"Project":   project,
		"Items":     items,
		"HasKanban": true,
	})
}

func (h *Projects) SelectProjectPage(c echo.Context) error {
	projects, err := h.store.Queries().ListProjects(context.Background())
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "select-project.html", map[string]any{
		"Projects": projects,
	})
}

func (h *Projects) SelectProject(c echo.Context) error {
	idStr := c.FormValue("project_id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	if _, err := h.store.Queries().GetProject(context.Background(), id); err != nil {
		return echo.ErrBadRequest
	}
	setProjectCookie(c, id)
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/projects/%d", id))
}

func (h *Projects) SwitchProject(c echo.Context) error {
	cookie := new(http.Cookie)
	cookie.Name = authmw.ProjectCookieName
	cookie.Value = ""
	cookie.Path = "/"
	cookie.MaxAge = -1
	c.SetCookie(cookie)
	return c.Redirect(http.StatusSeeOther, "/select-project")
}

func setProjectCookie(c echo.Context, id int64) {
	cookie := new(http.Cookie)
	cookie.Name = authmw.ProjectCookieName
	cookie.Value = strconv.FormatInt(id, 10)
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteStrictMode
	cookie.MaxAge = 365 * 24 * 60 * 60
	c.SetCookie(cookie)
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
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/projects/%d", id))
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
