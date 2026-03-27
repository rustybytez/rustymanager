package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"

	"github.com/labstack/echo/v4"

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
