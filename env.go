package main

import (
	"os"
	"strings"
)

type Env struct {
	isDocker bool
}

func getEnv() Env {
	isDocker := strings.ToLower(os.Getenv("IS_DOCKER")) == "true"

	return Env{
		isDocker,
	}
}
