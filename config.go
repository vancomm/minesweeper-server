package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type PostgresConfig struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DbName   string `json:"db_name"`
}

func (p PostgresConfig) DbUrl() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		p.Host, p.Port, p.User, p.Password, p.DbName,
	)
}

type Duration struct{ time.Duration }

// [Duration] implements [json.Marshaler]
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil

	default:
		return errors.New("invalid duration")
	}
}

type JwtConfig struct {
	TokenLifetime  Duration `json:"token_lifetime"`
	PrivateKeyPath string   `json:"private_key_path"`
	PublicKeyPath  string   `json:"public_key_path"`
}

type Config struct {
	Mode     string         `json:"mode"`
	Addr     string         `json:"addr"`
	Domain   string         `json:"domain"`
	Postgres PostgresConfig `json:"postgres"`
	Jwt      JwtConfig      `json:"jwt"`
}

func (c Config) Fields() logrus.Fields {
	return map[string]any{
		"mode":                 c.Mode,
		"addr":                 c.Addr,
		"domain":               c.Domain,
		"pg_host":              c.Postgres.Host,
		"pg_port":              c.Postgres.Port,
		"pg_user":              c.Postgres.User,
		"pg_db_name":           c.Postgres.DbName,
		"jwt_token_lifetime":   c.Jwt.TokenLifetime.Duration.String(),
		"jwt_private_key_path": c.Jwt.PrivateKeyPath,
		"jwt_public_key_path":  c.Jwt.PublicKeyPath,
	}
}

func (c Config) Production() bool {
	return c.Mode == "production"
}

func (c Config) Development() bool {
	return c.Mode != "production"
}

func (c Config) HttpCookieSameSite() http.SameSite {
	if c.Development() {
		return http.SameSiteNoneMode
	} else {
		return http.SameSiteStrictMode
	}
}

func ReadConfig(path string, config *Config) error {
	if b, err := os.ReadFile(path); err != nil {
		return err
	} else {
		return json.Unmarshal(b, config)
	}
}
