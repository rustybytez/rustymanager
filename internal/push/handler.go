package push

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"rustymanager/internal/db"
)

// Handler exposes HTTP endpoints for push subscription management.
type Handler struct {
	queries        db.Querier
	vapidPublicKey string
}

func NewHandler(q db.Querier, pubKey string) *Handler {
	return &Handler{queries: q, vapidPublicKey: pubKey}
}

// VAPIDPublicKey returns the server's VAPID public key so the browser can subscribe.
func (h *Handler) VAPIDPublicKey(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"public_key": h.vapidPublicKey})
}

type subscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// Subscribe stores a push subscription sent by the browser.
func (h *Handler) Subscribe(c echo.Context) error {
	var req subscribeRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		return echo.ErrBadRequest
	}
	if err := h.queries.UpsertPushSubscription(c.Request().Context(), db.UpsertPushSubscriptionParams{
		Endpoint: req.Endpoint,
		P256dh:   req.Keys.P256dh,
		Auth:     req.Keys.Auth,
	}); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// Unsubscribe removes a push subscription.
func (h *Handler) Unsubscribe(c echo.Context) error {
	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if err := h.queries.DeletePushSubscription(c.Request().Context(), req.Endpoint); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
