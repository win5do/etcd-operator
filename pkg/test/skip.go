package test

import (
	"os"
)

func ShouldRun() bool {
	return os.Getenv("integration") == "true"
}
