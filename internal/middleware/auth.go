package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"rustymanager/internal/store"
)

const (
	CookieName     = "user_id"
	CurrentUserKey = "current_user"
)

func RequireAuth(s *store.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(CookieName)
			if err != nil {
				return c.Redirect(http.StatusSeeOther, "/login")
			}
			userID, err := strconv.ParseInt(cookie.Value, 10, 64)
			if err != nil {
				return c.Redirect(http.StatusSeeOther, "/login")
			}
			user, err := s.Queries().GetUser(context.Background(), userID)
			if err != nil {
				return c.Redirect(http.StatusSeeOther, "/login")
			}
			c.Set(CurrentUserKey, user)
			return next(c)
		}
	}
}
