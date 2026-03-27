package handler

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"rustymanager/internal/db"
	authmw "rustymanager/internal/middleware"
	"rustymanager/internal/store"
)

type Auth struct {
	store *store.Store
}

func NewAuth(s *store.Store) *Auth { return &Auth{store: s} }

func (h *Auth) LoginPage(c echo.Context) error {
	return c.Render(http.StatusOK, "auth/login.html", map[string]any{})
}

func (h *Auth) Login(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	user, err := h.store.Queries().GetUserByUsername(context.Background(), username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return c.Render(http.StatusUnauthorized, "auth/login.html", map[string]any{
			"Error": "Invalid username or password.",
		})
	}

	cookie := new(http.Cookie)
	cookie.Name = authmw.CookieName
	cookie.Value = strconv.FormatInt(user.ID, 10)
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteStrictMode
	cookie.Expires = time.Now().Add(30 * 24 * time.Hour)
	c.SetCookie(cookie)

	return c.Redirect(http.StatusSeeOther, "/projects")
}

func (h *Auth) Logout(c echo.Context) error {
	c.SetCookie(&http.Cookie{Name: authmw.CookieName, Value: "", Path: "/", MaxAge: -1})
	return c.Redirect(http.StatusSeeOther, "/login")
}

func (h *Auth) RegisterPage(c echo.Context) error {
	return c.Render(http.StatusOK, "auth/register.html", map[string]any{})
}

func (h *Auth) Register(c echo.Context) error {
	authToken := c.FormValue("auth_token")
	if authToken != os.Getenv("AUTH_TOKEN") {
		return c.Render(http.StatusUnauthorized, "auth/register.html", map[string]any{
			"Error": "Invalid auth token.",
		})
	}

	username := c.FormValue("username")
	password := c.FormValue("password")
	name := c.FormValue("name")
	if username == "" || password == "" || name == "" {
		return c.Render(http.StatusBadRequest, "auth/register.html", map[string]any{
			"Error": "All fields are required.",
		})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if _, err := h.store.Queries().CreateUser(context.Background(), db.CreateUserParams{
		Name:         name,
		Username:     username,
		PasswordHash: string(hash),
	}); err != nil {
		return c.Render(http.StatusBadRequest, "auth/register.html", map[string]any{
			"Error": "Username already taken.",
		})
	}

	return c.Redirect(http.StatusSeeOther, "/login")
}
