package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"rustymanager/internal/store"
)

const (
	ProjectCookieName = "project_id"
	CurrentProjectKey = "current_project"
)

func RequireProject() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Get(CurrentProjectKey) == nil {
				return c.Redirect(http.StatusSeeOther, "/select-project")
			}
			return next(c)
		}
	}
}

func LoadProject(s *store.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(ProjectCookieName)
			if err != nil {
				return next(c)
			}
			projectID, err := strconv.ParseInt(cookie.Value, 10, 64)
			if err != nil {
				return next(c)
			}
			project, err := s.Queries().GetProject(context.Background(), projectID)
			if err != nil {
				return next(c)
			}
			c.Set(CurrentProjectKey, project)
			return next(c)
		}
	}
}
