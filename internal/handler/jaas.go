package handler

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// JaaSHandler issues short-lived JaaS JWTs for the browser to use when joining calls.
type JaaSHandler struct {
	appID      string
	keyID      string
	privateKey *rsa.PrivateKey
}

func NewJaaSHandler() (*JaaSHandler, error) {
	appID := os.Getenv("JAAS_APP_ID")
	keyID := os.Getenv("JAAS_KEY_ID")
	keyB64 := os.Getenv("JAAS_PRIVATE_KEY_BASE64")

	if appID == "" || keyID == "" || keyB64 == "" {
		return nil, fmt.Errorf("JAAS_APP_ID, JAAS_KEY_ID and JAAS_PRIVATE_KEY_BASE64 must all be set")
	}

	keyPEM, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, fmt.Errorf("JAAS_PRIVATE_KEY_BASE64 is not valid base64: %w", err)
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("JAAS_PRIVATE_KEY_BASE64 does not contain valid PEM after decoding")
	}

	rsaKey, err := parseRSAKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return &JaaSHandler{appID: appID, keyID: keyID, privateKey: rsaKey}, nil
}

func parseRSAKey(der []byte) (*rsa.PrivateKey, error) {
	// Try PKCS8 first (JaaS default), fall back to PKCS1.
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, fmt.Errorf("JAAS_PRIVATE_KEY_BASE64 must be an RSA key")
	}
	if rsaKey, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return rsaKey, nil
	}
	return nil, fmt.Errorf("could not parse JAAS_PRIVATE_KEY_BASE64 as PKCS8 or PKCS1 RSA key")
}

// Token issues a JWT for the given room and returns it as JSON.
func (h *JaaSHandler) Token(c echo.Context) error {
	room := c.QueryParam("room")
	if room == "" {
		return echo.ErrBadRequest
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":  "chat",
		"sub":  h.appID,
		"aud":  "jitsi",
		"iat":  now.Unix(),
		"nbf":  now.Unix(),
		"exp":  now.Add(2 * time.Hour).Unix(),
		"room": "*",
		"context": map[string]any{
			"features": map[string]bool{
				"recording":     false,
				"livestreaming": false,
				"transcription": false,
				"outbound-call": false,
			},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = h.keyID

	signed, err := token.SignedString(h.privateKey)
	if err != nil {
		return fmt.Errorf("sign JWT: %w", err)
	}

	return c.JSON(http.StatusOK, map[string]string{"token": signed})
}
