package config

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	publicKey     *rsa.PublicKey
	privateKey    *rsa.PrivateKey
	signingMethod jwt.SigningMethod
	tokenLifetime time.Duration
}

func loadPrivateKey() (*rsa.PrivateKey, error) {
	privateKeyStr, ok := os.LookupEnv("JWT_PRIVATE_KEY")
	if ok {
		return jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyStr))
	}
	privateKeyPath, ok := os.LookupEnv("JWT_PRIVATE_KEY_FILE")
	if !ok {
		return nil, fmt.Errorf("no JWT_PRIVATE_KEY or JWT_PRIVATE_KEY_FILE env variable set")
	}
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read JWT private key: %w", err)
	}
	return jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
}

func loadPublicKey() (*rsa.PublicKey, error) {
	publicKeyStr, ok := os.LookupEnv("JWT_PUBLIC_KEY")
	if ok {
		return jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyStr))
	}
	publicKeyPath, ok := os.LookupEnv("JWT_PUBLIC_KEY_FILE")
	if !ok {
		return nil, fmt.Errorf("no JWT_PUBLIC_KEY or JWT_PUBLIC_KEY_FILE env variable set")
	}
	publicKeyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read JWT public key: %w", err)
	}
	return jwt.ParseRSAPublicKeyFromPEM(publicKeyBytes)
}

func NewJWT() (*JWT, error) {
	privateKey, err := loadPrivateKey()
	if err != nil {
		return nil, err
	}

	publicKey, err := loadPublicKey()
	if err != nil {
		return nil, err
	}

	j := &JWT{
		privateKey:    privateKey,
		publicKey:     publicKey,
		signingMethod: jwt.GetSigningMethod("RS256"),
		tokenLifetime: time.Hour * 24 * 30,
	}

	return j, nil
}

func (j *JWT) KeyFunc(t *jwt.Token) (*rsa.PublicKey, error) {
	return j.publicKey, nil
}

func (j *JWT) Sign(claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(j.signingMethod, claims).SignedString(j.privateKey)
}

func (j *JWT) ParseWithClaims(tokenString string, claims jwt.Claims) (*jwt.Token, error) {
	return jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			return j.publicKey, nil
		},
	)
}
