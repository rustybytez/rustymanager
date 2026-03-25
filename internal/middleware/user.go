package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"rustymanager/internal/store"
)

const (
	UserCookieName = "user_id"
	CurrentUserKey = "current_user"
)

func RequireUser(s *store.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(UserCookieName)
			if err != nil {
				return c.Redirect(http.StatusSeeOther, "/select-user")
			}
			userID, err := strconv.ParseInt(cookie.Value, 10, 64)
			if err != nil {
				return c.Redirect(http.StatusSeeOther, "/select-user")
			}
			user, err := s.Queries().GetUser(context.Background(), userID)
			if err != nil {
				return c.Redirect(http.StatusSeeOther, "/select-user")
			}
			c.Set(CurrentUserKey, user)
			return next(c)
		}
	}
}
