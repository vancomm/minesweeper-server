package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Username string
	Password string
	Host     string
	Port     uint16
	DBName   string
	SSLMode  string
}

func loadPassword() (string, error) {
	password, ok := os.LookupEnv("POSTGRES_PASSWORD")
	if ok {
		return password, nil
	}

	passwordFile, ok := os.LookupEnv("POSTGRES_PASSWORD_FILE")
	if !ok {
		return "", fmt.Errorf("no POSTGRES_PASSWORD or POSTGRES_PASSWORD_FILE env variable set")
	}

	data, err := os.ReadFile(passwordFile)
	if err != nil {
		return "", fmt.Errorf("unable to read from password file: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

func NewDatabase() (*Database, error) {
	username, ok := os.LookupEnv("POSTGRES_USER")
	if !ok {
		return nil, fmt.Errorf("no POSTGRES_USER env variable set")
	}

	password, err := loadPassword()
	if err != nil {
		return nil, fmt.Errorf("unable to load password: %w", err)
	}

	portStr, ok := os.LookupEnv("POSTGRES_PORT")
	if !ok {
		return nil, fmt.Errorf("no POSTGRES_PORT env variable set")
	}

	host, ok := os.LookupEnv("POSTGRES_HOST")
	if !ok {
		return nil, fmt.Errorf("no POSTGRES_HOST env variable set")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("unable to convert port to int: %w", err)
	}

	dbName, ok := os.LookupEnv("POSTGRES_DB")
	if !ok {
		return nil, fmt.Errorf("no POSTGRES_DB env variable set")
	}

	sslMode, ok := os.LookupEnv("POSTGRES_SSLMODE")
	if !ok {
		return nil, fmt.Errorf("no POSTGRES_SSLMODE env variable set")
	}

	config := &Database{
		Username: username,
		Password: password,
		Host:     host,
		Port:     uint16(port),
		DBName:   dbName,
		SSLMode:  sslMode,
	}

	return config, nil
}

func (c Database) URL() string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username,
		url.QueryEscape(c.Password),
		c.Host,
		c.Port,
		c.DBName,
		c.SSLMode,
	)
}

func (c Database) DSN() string {
	return fmt.Sprintf(
		"user=%s password=%s host=%s port=%d dbname=%s sslmode=%s",
		c.Username, c.Password, c.Host, c.Port, c.DBName, c.SSLMode,
	)
}

func DbURL() (string, error) {
	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if ok {
		return dbURL, nil
	}

	cfg, err := NewDatabase()
	if err == nil {
		return cfg.URL(), nil
	}

	return "", fmt.Errorf("no DATABASE_URL set; %w", err)
}

func NewPgxpoolConfig() (*pgxpool.Config, error) {
	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if ok {
		return pgxpool.ParseConfig(dbURL)
	}

	cfg, err := NewDatabase()
	if err == nil {
		return pgxpool.ParseConfig(cfg.DSN())
	}

	return nil, fmt.Errorf("no DATABASE_URL set; %w", err)
}
