package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port             string
	DBDriver         string
	DBDSN            string
	DBAutoMigrate    bool
	JWTSecret        string
	CORSOrigins      []string
	MaxTracePoints   int
	LogLevel         string
	ShutdownTimeout  time.Duration
	AccessTokenTTL   time.Duration
	LoginRateLimit   int
	ImportRateLimit  int
	AnalyzeRateLimit int
}

func Load() (Config, error) {
	c := Config{
		Port:             env("PORT", "8080"),
		DBDriver:         strings.ToLower(env("DB_DRIVER", "postgres")),
		DBDSN:            env("DB_DSN", "host=localhost user=otdr_app password=otdr_local_password dbname=fiber_otdr port=5432 sslmode=disable"),
		DBAutoMigrate:    envBool("DB_AUTO_MIGRATE", true),
		JWTSecret:        env("JWT_SECRET", "development-secret-change-me-at-least-32-bytes"),
		CORSOrigins:      split(env("CORS_ORIGINS", "http://localhost:18522")),
		MaxTracePoints:   envInt("MAX_TRACE_POINTS", 20000),
		LogLevel:         env("LOG_LEVEL", "info"),
		ShutdownTimeout:  10 * time.Second,
		AccessTokenTTL:   8 * time.Hour,
		LoginRateLimit:   envInt("LOGIN_RATE_LIMIT", 30),
		ImportRateLimit:  envInt("IMPORT_RATE_LIMIT", 40),
		AnalyzeRateLimit: envInt("ANALYZE_RATE_LIMIT", 60),
	}
	if len(c.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	if c.DBDriver != "postgres" && c.DBDriver != "sqlite" {
		return Config{}, fmt.Errorf("unsupported DB_DRIVER %q", c.DBDriver)
	}
	if c.MaxTracePoints < 64 || c.MaxTracePoints > 200000 {
		return Config{}, fmt.Errorf("MAX_TRACE_POINTS must be between 64 and 200000")
	}
	return c, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v, err := strconv.Atoi(env(key, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return v
}

func envBool(key string, fallback bool) bool {
	v, err := strconv.ParseBool(env(key, strconv.FormatBool(fallback)))
	if err != nil {
		return fallback
	}
	return v
}

func split(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
