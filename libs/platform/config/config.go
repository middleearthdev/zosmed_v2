// Package config loads environment-derived configuration for Zosmed services.
// All required variables are validated at startup; missing vars surface a clear error.
package config

import (
	"fmt"
	"os"
)

// Config holds all environment-derived configuration for Zosmed services.
type Config struct {
	// DBURL is the PostgreSQL connection URL (DB_URL).
	DBURL string
	// RedisURL is the Redis connection URL (REDIS_URL).
	RedisURL string
	// MetaAppSecret is the Meta app secret for HMAC-SHA256 webhook verification (META_APP_SECRET).
	MetaAppSecret string
	// MetaVerifyToken is the token used in webhook subscription verification (META_VERIFY_TOKEN).
	MetaVerifyToken string
	// WAPhone is the WhatsApp phone number (E.164, e.g. 6281234567890) for wa.me links (WA_PHONE).
	WAPhone string
	// Port is the HTTP server port; defaults to "8080" if not set (PORT).
	Port string
}

// Load reads env vars, validates required fields, and returns a Config.
// Returns a descriptive error naming the first missing variable.
func Load() (*Config, error) {
	c := &Config{
		DBURL:           os.Getenv("DB_URL"),
		RedisURL:        os.Getenv("REDIS_URL"),
		MetaAppSecret:   os.Getenv("META_APP_SECRET"),
		MetaVerifyToken: os.Getenv("META_VERIFY_TOKEN"),
		WAPhone:         os.Getenv("WA_PHONE"),
		Port:            os.Getenv("PORT"),
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	return c, nil
}

// required lists (envName, value) pairs that must be non-empty.
func (c *Config) validate() error {
	pairs := []struct {
		name string
		val  string
	}{
		{"DB_URL", c.DBURL},
		{"REDIS_URL", c.RedisURL},
		{"META_APP_SECRET", c.MetaAppSecret},
		{"META_VERIFY_TOKEN", c.MetaVerifyToken},
		{"WA_PHONE", c.WAPhone},
	}
	for _, p := range pairs {
		if p.val == "" {
			return fmt.Errorf("config: required env var %s is not set", p.name)
		}
	}
	return nil
}
