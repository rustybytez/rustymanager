package handler

import (
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"

	authmw "rustymanager/internal/middleware"
)

type Auth struct{}

func NewAuth() *Auth { return &Auth{} }

func (h *Auth) LoginPage(c echo.Context) error {
	return c.Render(http.StatusOK, "auth/login.html", nil)
}

func (h *Auth) Login(c echo.Context) error {
	token := c.FormValue("token")
	if token != os.Getenv("AUTH_TOKEN") {
		return c.Render(http.StatusUnauthorized, "auth/login.html", map[string]any{
			"Error": "Invalid token.",
		})
	}

	cookie := new(http.Cookie)
	cookie.Name = authmw.CookieName
	cookie.Value = token
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteStrictMode
	cookie.Expires = time.Now().Add(24 * time.Hour)
	c.SetCookie(cookie)

	return c.Redirect(http.StatusSeeOther, "/projects")
}

func (h *Auth) Logout(c echo.Context) error {
	for _, name := range []string{authmw.CookieName, authmw.UserCookieName} {
		c.SetCookie(&http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1})
	}
	return c.Redirect(http.StatusSeeOther, "/login")
}
