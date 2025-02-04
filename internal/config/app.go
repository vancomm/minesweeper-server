package config

import "os"

func Mount() string {
	return os.Getenv("APP_MOUNT")
}

func Port() string {
	return os.Getenv("APP_PORT")
}
