package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"rustymanager/internal/github"
	"rustymanager/internal/store"
)

type Commits struct {
	store *store.Store
}

func NewCommits(s *store.Store) *Commits {
	return &Commits{store: s}
}

func (h *Commits) List(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	project, err := h.store.Queries().GetProject(context.Background(), id)
	if err != nil {
		return echo.ErrNotFound
	}
	repo := project.GithubRepo
	if repo == "" {
		return c.JSON(http.StatusOK, []any{})
	}
	commits, err := github.FetchCommits(repo, 5)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, commits)
}
