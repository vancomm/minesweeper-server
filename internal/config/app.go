package config

import "os"

func BasePath() string {
	return os.Getenv("APP_BASE_PATH")
}

func Port() string {
	return os.Getenv("APP_PORT")
}
