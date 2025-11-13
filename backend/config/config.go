package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

const DefaultHTTPPort = ":8080"

type Config struct {
	PgConnStr string
	HTTPPort  string
}

func NewConfig(logger *zap.Logger) (Config, error) {
	cfg := Config{}

	if err := godotenv.Load(); err != nil {
		logger.Warn("could not load .env file",
			zap.Error(err),
		)
	}

	pgDsn := os.Getenv("POSTGRES_CONNECTION_STRING")
	if pgDsn == "" {
		logger.Info("POSTGRES_CONNECTION_STRING not found")
		return cfg, fmt.Errorf("POSTGRES_CONNECTION_STRING environment variable is required")
	}
	cfg.PgConnStr = pgDsn

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		cfg.HTTPPort = DefaultHTTPPort
	} else {
		// Ensure port starts with ':' if not already present
		if len(httpPort) > 0 && httpPort[0] != ':' {
			cfg.HTTPPort = ":" + httpPort
		} else {
			cfg.HTTPPort = httpPort
		}
	}
	return cfg, nil
}
