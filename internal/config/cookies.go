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
	jwt      JWT
}

type PlayerClaims struct {
	PlayerId int64  `json:"player_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewPlayerClaims(playerId int64, username string) *PlayerClaims {
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
	secure := secureStr != "0"

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

func (c *Cookies) Refresh(w http.ResponseWriter, token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("malformed JWT token generated")
	}
	header, payload, signature := parts[0], parts[1], parts[2]
	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Path:     "/",
		Value:    header + "." + payload,
		Expires:  time.Now().Add(c.jwt.tokenLifetime),
		Domain:   c.Domain,
		Secure:   c.Secure,
		SameSite: c.SameSite,
		// Partitioned: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "sign",
		Path:     "/",
		Value:    signature,
		Expires:  time.Now().Add(c.jwt.tokenLifetime),
		HttpOnly: true,
		Domain:   c.Domain,
		Secure:   c.Secure,
		SameSite: c.SameSite,
		// Partitioned: true,
	})
	return nil
}

func (c *Cookies) ParsePlayerClaims(r *http.Request) (*PlayerClaims, error) {
	authCookie, err := r.Cookie("auth")
	if err != nil {
		return nil, err
	}
	signCookie, err := r.Cookie("sign")
	if err != nil {
		return nil, err
	}
	token, err := c.jwt.ParseWithClaims(
		authCookie.Value+"."+signCookie.Value, &PlayerClaims{},
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*PlayerClaims)
	if !ok {
		return nil, fmt.Errorf("malformed claims")
	}
	return claims, nil
}
