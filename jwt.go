package main

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSigningMethod = jwt.GetSigningMethod("RS256")

type PlayerClaims struct {
	PlayerId int    `json:"player_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func createPlayerToken(playerId int, username string) (string, error) {
	claims := PlayerClaims{
		playerId,
		username,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.Jwt.TokenLifetime.Duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.NewWithClaims(jwtSigningMethod, claims).
		SignedString(jwtPrivateKey)
	log.Debug("created new token: ", token)
	return token, err
}

func setPlayerCookies(w http.ResponseWriter, token string) {
	parts := strings.Split(token, ".")
	header, payload, signature := parts[0], parts[1], parts[2]
	jsCookie := &http.Cookie{
		Name:     "auth",
		Path:     "/",
		Domain:   config.Domain,
		Value:    header + "." + payload,
		Secure:   config.Production(),
		Expires:  time.Now().Add(config.Jwt.TokenLifetime.Duration),
		SameSite: config.HttpCookieSameSite(),
		// Partitioned: true,
	}
	httpCookie := &http.Cookie{
		Name:     "sign",
		Path:     "/",
		Domain:   config.Domain,
		Value:    signature,
		Secure:   config.Production(),
		HttpOnly: true,
		Expires:  time.Now().Add(config.Jwt.TokenLifetime.Duration),
		SameSite: config.HttpCookieSameSite(),
		// Partitioned: true,
	}
	http.SetCookie(w, jsCookie)
	http.SetCookie(w, httpCookie)
}

func refreshPlayerCookies(w http.ResponseWriter, claims PlayerClaims) error {
	token, err := createPlayerToken(
		claims.PlayerId, claims.Username,
	)
	if err == nil {
		setPlayerCookies(w, token)
	}
	return err
}

func clearPlayerCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Path:     "/",
		Domain:   config.Domain,
		Value:    "delete",
		MaxAge:   -1,
		Secure:   config.Production(),
		SameSite: config.HttpCookieSameSite(),
		// Partitioned: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "sign",
		Path:     "/",
		Domain:   config.Domain,
		Value:    "delete",
		MaxAge:   -1,
		Secure:   config.Production(),
		HttpOnly: true,
		SameSite: config.HttpCookieSameSite(),
		// Partitioned: true,
	})
}

func getJWTFromCookies(r *http.Request) (string, error) {
	authCookie, err := r.Cookie("auth")
	if err != nil {
		return "", err
	}
	signCookie, err := r.Cookie("sign")
	if err != nil {
		return "nil", err
	}
	return authCookie.Value + "." + signCookie.Value, nil
}

func getPublicKey(t *jwt.Token) (interface{}, error) {
	return jwtPublicKey, nil
}

func tryParseJWTCookie(tokenString string) (*PlayerClaims, error) {
	if token, err := jwt.ParseWithClaims(
		tokenString, &PlayerClaims{}, getPublicKey,
	); err != nil {
		return nil, err
	} else if claims, ok := token.Claims.(*PlayerClaims); ok {
		return claims, nil
	} else {
		return nil, errors.New("unknown claims type")
	}
}
