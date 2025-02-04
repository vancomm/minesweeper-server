package config

import "os"

func Development() bool {
	development, ok := os.LookupEnv("DEVELOPMENT")
	if !ok {
		return false
	}
	return development != "0"
}
