package main

import (
	"crypto/rsa"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ssh-keygen -t rsa -m pem -f jwt-private-key.pem
// openssl rsa -in jwt-private-key.pem -pubout -out jwt-public-key.pem

var (
	jwtPrivateKey    *rsa.PrivateKey
	jwtPublicKey     *rsa.PublicKey
	jwtSigningMethod               = jwt.GetSigningMethod("RS256")
	jwtLifetime      time.Duration = 24 * time.Hour
)

func init() {
	pvtKeyBytes, err := os.ReadFile("./secrets/jwt-private-key.pem")
	if err != nil {
		log.Fatal("unable to read JWT private key: ", err)
	}
	jwtPrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM(pvtKeyBytes)
	if err != nil {
		log.Fatal("unable to parse JWT private key: ", err)
	}
	pubKeyBytes, err := os.ReadFile("./secrets/jwt-public-key.pem")
	if err != nil {
		log.Fatal("unable to read JWT public key: ", err)
	}
	jwtPublicKey, err = jwt.ParseRSAPublicKeyFromPEM(pubKeyBytes)
	if err != nil {
		log.Fatal("unable to parse JWT public key: ", err)
	}
}

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
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtLifetime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.NewWithClaims(jwtSigningMethod, claims).
		SignedString(jwtPrivateKey)
	log.Debug("created new token: ", token)
	return token, err
}

var sameSite http.SameSite

func init() {
	if development {
		sameSite = http.SameSiteNoneMode
	} else {
		sameSite = http.SameSiteStrictMode
	}
}

func setPlayerCookies(w http.ResponseWriter, token string) {
	parts := strings.Split(token, ".")
	header, payload, signature := parts[0], parts[1], parts[2]
	jsCookie := &http.Cookie{
		Name:     "auth",
		Path:     "/",
		Domain:   domain,
		Value:    header + "." + payload,
		Secure:   !development,
		Expires:  time.Now().Add(jwtLifetime),
		SameSite: sameSite,
		// Partitioned: true,
	}
	httpCookie := &http.Cookie{
		Name:     "sign",
		Path:     "/",
		Domain:   domain,
		Value:    signature,
		Secure:   !development,
		HttpOnly: true,
		SameSite: sameSite,
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
		Domain:   domain,
		Value:    "delete",
		MaxAge:   -1,
		Secure:   !development,
		SameSite: sameSite,
		// Partitioned: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "sign",
		Path:     "/",
		Domain:   domain,
		Value:    "delete",
		MaxAge:   -1,
		Secure:   !development,
		HttpOnly: true,
		SameSite: sameSite,
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
