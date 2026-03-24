package middleware

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

const CookieName = "auth_token"

func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie(CookieName)
		if err != nil || cookie.Value != os.Getenv("AUTH_TOKEN") {
			return c.Redirect(http.StatusSeeOther, "/login")
		}
		return next(c)
	}
}
