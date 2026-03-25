package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func Settings(c echo.Context) error {
	return c.Render(http.StatusOK, "settings/index.html", map[string]any{})
}
