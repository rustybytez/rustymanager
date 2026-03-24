package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"rustymanager/internal/db"
	"rustymanager/internal/store"
)

type Users struct {
	store *store.Store
}

func NewUsers(s *store.Store) *Users {
	return &Users{store: s}
}

func (h *Users) Index(c echo.Context) error {
	users, err := h.store.Queries().ListUsers(context.Background())
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "users/index.html", map[string]any{
		"Users": users,
	})
}

func (h *Users) New(c echo.Context) error {
	return c.Render(http.StatusOK, "users/new.html", nil)
}

func (h *Users) Create(c echo.Context) error {
	name := c.FormValue("name")
	if _, err := h.store.Queries().CreateUser(context.Background(), name); err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/users")
}

func (h *Users) Edit(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	user, err := h.store.Queries().GetUser(context.Background(), id)
	if err != nil {
		return echo.ErrNotFound
	}
	return c.Render(http.StatusOK, "users/edit.html", map[string]any{
		"User": user,
	})
}

func (h *Users) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	params := db.UpdateUserParams{
		ID:   id,
		Name: c.FormValue("name"),
	}
	if _, err := h.store.Queries().UpdateUser(context.Background(), params); err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/users")
}

func (h *Users) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.store.Queries().DeleteUser(context.Background(), id); err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/users")
}
