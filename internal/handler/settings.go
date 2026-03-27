package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"rustymanager/internal/db"
	authmw "rustymanager/internal/middleware"
	"rustymanager/internal/store"
)

type Settings struct {
	store *store.Store
}

func NewSettings(s *store.Store) *Settings {
	return &Settings{store: s}
}

func (h *Settings) Index(c echo.Context) error {
	return c.Render(http.StatusOK, "settings/index.html", map[string]any{})
}

func (h *Settings) Admin(c echo.Context) error {
	currentUser := c.Get(authmw.CurrentUserKey).(db.User)
	users, err := h.store.Queries().ListUsers(context.Background())
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "settings/admin.html", map[string]any{
		"Users":         users,
		"CurrentUserID": currentUser.ID,
	})
}

func (h *Settings) ResetPassword(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	password := c.FormValue("password")
	if password == "" {
		return echo.ErrBadRequest
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := h.store.Queries().UpdateUserPassword(context.Background(), db.UpdateUserPasswordParams{
		PasswordHash: string(hash),
		ID:           id,
	}); err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/settings/admin")
}

func (h *Settings) DeleteUser(c echo.Context) error {
	currentUser := c.Get(authmw.CurrentUserKey).(db.User)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	if id == currentUser.ID {
		return echo.ErrForbidden
	}
	if err := h.store.Queries().DeleteUser(context.Background(), id); err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/settings/admin")
}

func (h *Settings) GenerateAPIToken(c echo.Context) error {
	user := c.Get(authmw.CurrentUserKey).(db.User)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	token := hex.EncodeToString(b)
	err := h.store.Queries().SetUserAPIToken(context.Background(), db.SetUserAPITokenParams{
		ApiToken: sql.NullString{String: token, Valid: true},
		ID:       user.ID,
	})
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/settings")
}

func (h *Settings) RevokeAPIToken(c echo.Context) error {
	user := c.Get(authmw.CurrentUserKey).(db.User)
	err := h.store.Queries().SetUserAPIToken(context.Background(), db.SetUserAPITokenParams{
		ApiToken: sql.NullString{Valid: false},
		ID:       user.ID,
	})
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/settings")
}
