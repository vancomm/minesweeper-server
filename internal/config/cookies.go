package config

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Cookies struct {
	Domain   string
	Secure   bool
	SameSite http.SameSite
}

type PlayerClaims struct {
	PlayerId int    `json:"player_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewPlayerClaims(playerId int, username string) *PlayerClaims {
	return &PlayerClaims{
		PlayerId: playerId,
		Username: username,
	}
}

func NewCookies() (*Cookies, error) {
	domain, ok := os.LookupEnv("COOKIES_DOMAIN")
	if !ok {
		return nil, fmt.Errorf("COOKIES_DOMAIN env variable is not set")
	}

	secureStr, ok := os.LookupEnv("COOKIES_SECURE")
	if !ok {
		return nil, fmt.Errorf("COOKIES_SECURE env variable is not set")
	}
	secure := secureStr != "0" && secureStr != ""

	sameSiteStr, ok := os.LookupEnv("COOKIES_SAMESITE")
	if !ok {
		return nil, fmt.Errorf("COOKIES_SAMESITE env variable is not set")
	}
	sameSite := http.SameSiteStrictMode
	switch strings.ToUpper(sameSiteStr) {
	case "DEFAULT":
		sameSite = http.SameSiteDefaultMode
	case "LAX":
		sameSite = http.SameSiteLaxMode
	case "STRICT":
		sameSite = http.SameSiteStrictMode
	case "NONE":
		sameSite = http.SameSiteNoneMode
	}

	cookies := &Cookies{
		Domain:   domain,
		Secure:   secure,
		SameSite: sameSite,
	}

	return cookies, nil
}

func (c *Cookies) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Path:     "/",
		Value:    "delete",
		MaxAge:   -1,
		Domain:   c.Domain,
		Secure:   c.Secure,
		SameSite: c.SameSite,
		// Partitioned: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "sign",
		Path:     "/",
		Value:    "delete",
		MaxAge:   -1,
		HttpOnly: true,
		Domain:   c.Domain,
		Secure:   c.Secure,
		SameSite: c.SameSite,
		// Partitioned: true,
	})
}

func (c *Cookies) Refresh(w http.ResponseWriter, token string, expires time.Time) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("malformed JWT token generated")
	}
	header, payload, signature := parts[0], parts[1], parts[2]
	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Path:     "/",
		Value:    header + "." + payload,
		Expires:  expires,
		Domain:   c.Domain,
		Secure:   c.Secure,
		SameSite: c.SameSite,
		// Partitioned: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "sign",
		Path:     "/",
		Value:    signature,
		Expires:  expires,
		HttpOnly: true,
		Domain:   c.Domain,
		Secure:   c.Secure,
		SameSite: c.SameSite,
		// Partitioned: true,
	})
	return nil
}
