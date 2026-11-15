package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddress            string
	DatabaseURL            string
	DatabaseMaxConns       int32
	DatabaseMinConns       int32
	DatabaseConnectTimeout time.Duration
	DatabaseQueryTimeout   time.Duration
	DatabaseMaxLifetime    time.Duration
	ShutdownTimeout        time.Duration
	HTTPReadTimeout        time.Duration
	HTTPReadHeaderTimeout  time.Duration
	HTTPWriteTimeout       time.Duration
	HTTPIdleTimeout        time.Duration
	LogLevel               string
}

func Read() (Config, error) {
	var cfg Config
	var err error

	if cfg.HTTPAddress, err = required("HTTP_ADDR"); err != nil {
		return cfg, err
	}
	if cfg.DatabaseURL, err = required("DATABASE_URL"); err != nil {
		return cfg, err
	}
	if cfg.LogLevel, err = required("LOG_LEVEL"); err != nil {
		return cfg, err
	}
	if cfg.DatabaseMaxConns, err = integer("DATABASE_MAX_CONNS"); err != nil {
		return cfg, err
	}
	if cfg.DatabaseMinConns, err = integer("DATABASE_MIN_CONNS"); err != nil {
		return cfg, err
	}
	if cfg.DatabaseMaxConns < 1 || cfg.DatabaseMinConns < 0 || cfg.DatabaseMinConns > cfg.DatabaseMaxConns {
		return cfg, errors.New("invalid database pool size")
	}
	if cfg.DatabaseConnectTimeout, err = duration("DATABASE_CONNECT_TIMEOUT"); err != nil {
		return cfg, err
	}
	if cfg.DatabaseQueryTimeout, err = duration("DATABASE_QUERY_TIMEOUT"); err != nil {
		return cfg, err
	}
	if cfg.DatabaseMaxLifetime, err = duration("DATABASE_MAX_CONN_LIFETIME"); err != nil {
		return cfg, err
	}
	if cfg.ShutdownTimeout, err = duration("SHUTDOWN_TIMEOUT"); err != nil {
		return cfg, err
	}
	if cfg.HTTPReadTimeout, err = duration("HTTP_READ_TIMEOUT"); err != nil {
		return cfg, err
	}
	if cfg.HTTPReadHeaderTimeout, err = duration("HTTP_READ_HEADER_TIMEOUT"); err != nil {
		return cfg, err
	}
	if cfg.HTTPWriteTimeout, err = duration("HTTP_WRITE_TIMEOUT"); err != nil {
		return cfg, err
	}
	if cfg.HTTPIdleTimeout, err = duration("HTTP_IDLE_TIMEOUT"); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func required(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return value, nil
}

func integer(name string) (int32, error) {
	value, err := required(name)
	if err != nil {
		return 0, err
	}

	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}

	return int32(number), nil
}

func duration(name string) (time.Duration, error) {
	value, err := required(name)
	if err != nil {
		return 0, err
	}

	result, err := time.ParseDuration(value)
	if err != nil || result <= 0 {
		return 0, fmt.Errorf("invalid %s: %q", name, value)
	}

	return result, nil
}
