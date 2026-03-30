package filestore

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

const maxUploadBytes = 10 << 20 // 10 MB

var allowedMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// Handler exposes HTTP endpoints for file uploads.
type Handler struct {
	store *Store
}

// NewHandler creates a Handler backed by the given Store.
func NewHandler(s *Store) *Handler {
	return &Handler{store: s}
}

// Upload handles POST /projects/:id/chat/upload.
// Accepts a multipart "file" field containing an image and returns {"url": "..."}.
func (h *Handler) Upload(c echo.Context) error {
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxUploadBytes)

	file, err := c.FormFile("file")
	if err != nil {
		return echo.ErrBadRequest
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "file too large")
	}

	contentType := http.DetectContentType(data)
	ext, ok := allowedMIME[contentType]
	if !ok {
		return echo.NewHTTPError(http.StatusUnsupportedMediaType, "only images are allowed")
	}

	url, err := h.store.Save(data, ext)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]string{"url": url})
}
