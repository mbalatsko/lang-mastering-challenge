package utils

import (
	"fmt"
	"os"
)

func MustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("%s env variable must be set to non equal value", key))
	}
	return v
}
