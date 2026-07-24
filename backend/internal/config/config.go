package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultPort = 8080

type Config struct {
	Address string
}

func FromEnv() (Config, error) {
	value := strings.TrimSpace(os.Getenv("PORT"))
	if value == "" {
		value = strconv.Itoa(defaultPort)
	}

	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be an integer between 1 and 65535")
	}

	return Config{Address: ":" + strconv.Itoa(port)}, nil
}
