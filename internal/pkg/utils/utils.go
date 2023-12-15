package utils

import "os"

// check for the environment variable with a given key.
// if key is not set the fallback value is returned
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
