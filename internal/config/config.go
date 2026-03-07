package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	AuthDisabled = "disabled"
	AuthJWT      = "jwt"
)

type Config struct {
	ListenAddr      string
	AuthMode        string
	JWTPublicKey    string
	LogLevel        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	MaxHeaderBytes  int
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddr:      getEnv("LISTEN_ADDR", ":8080"),
		AuthMode:        strings.ToLower(getEnv("AUTH_MODE", AuthDisabled)),
		JWTPublicKey:    getEnv("JWT_PUBLIC_KEY", ""),
		LogLevel:        strings.ToLower(getEnv("LOG_LEVEL", "info")),
		ReadTimeout:     getDurationEnv("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    getDurationEnv("WRITE_TIMEOUT", 60*time.Second),
		IdleTimeout:     getDurationEnv("IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		MaxHeaderBytes:  getIntEnv("MAX_HEADER_BYTES", 1<<20),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	switch c.AuthMode {
	case AuthDisabled:
		return nil
	case AuthJWT:
		if c.JWTPublicKey == "" {
			return errors.New("JWT_PUBLIC_KEY is required when AUTH_MODE=jwt")
		}
		return nil
	default:
		return fmt.Errorf("unsupported AUTH_MODE %q", c.AuthMode)
	}
}

func (c Config) Address() string {
	return c.ListenAddr
}

func getEnv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getIntEnv(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
