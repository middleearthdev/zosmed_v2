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
	// IGAppID is the Instagram Login app's client_id, used for OAuth authorize/exchange (IG_APP_ID).
	IGAppID string
	// IGAppSecret is the Instagram Login app secret. One value serves two purposes
	// (DRY §12a-1 — one App, one secret): (a) OAuth client_secret, (b) HMAC-SHA256
	// webhook signature verification, (c) signing the connect-flow CSRF state (IG_APP_SECRET).
	IGAppSecret string
	// IGVerifyToken is the token used in webhook subscription verification (IG_VERIFY_TOKEN).
	IGVerifyToken string
	// IGRedirectURI is the OAuth redirect_uri; must match exactly what is
	// registered in the Meta App Dashboard (IG_REDIRECT_URI).
	IGRedirectURI string
	// WAPhone is the WhatsApp phone number (E.164, e.g. 6281234567890) for wa.me links (WA_PHONE).
	WAPhone string
	// Port is the HTTP server port; defaults to "8080" if not set (PORT).
	Port string
	// AppEnv is the deployment environment (APP_ENV): "dev" (default) or "prod".
	// Controls the session cookie's Secure flag (ADR-003 §0 decision 1) and
	// guards apps/api/cmd/seed against running against production data.
	AppEnv string
	// WebBaseURL is the frontend's base URL (WEB_BASE_URL), used for absolute
	// redirects if a handler ever needs one instead of a relative path
	// (ADR-003 §6). Optional — most redirects in this codebase are relative.
	WebBaseURL string
}

// IsProd reports whether AppEnv is "prod".
func (c *Config) IsProd() bool {
	return c.AppEnv == "prod"
}

// Load reads env vars, validates required fields, and returns a Config.
// Returns a descriptive error naming the first missing variable.
func Load() (*Config, error) {
	c := &Config{
		DBURL:         os.Getenv("DB_URL"),
		RedisURL:      os.Getenv("REDIS_URL"),
		IGAppID:       os.Getenv("IG_APP_ID"),
		IGAppSecret:   os.Getenv("IG_APP_SECRET"),
		IGVerifyToken: os.Getenv("IG_VERIFY_TOKEN"),
		IGRedirectURI: os.Getenv("IG_REDIRECT_URI"),
		WAPhone:       os.Getenv("WA_PHONE"),
		Port:          os.Getenv("PORT"),
		AppEnv:        os.Getenv("APP_ENV"),
		WebBaseURL:    os.Getenv("WEB_BASE_URL"),
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	if c.AppEnv == "" {
		c.AppEnv = "dev"
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
		{"IG_APP_ID", c.IGAppID},
		{"IG_APP_SECRET", c.IGAppSecret},
		{"IG_VERIFY_TOKEN", c.IGVerifyToken},
		{"IG_REDIRECT_URI", c.IGRedirectURI},
		{"WA_PHONE", c.WAPhone},
	}
	for _, p := range pairs {
		if p.val == "" {
			return fmt.Errorf("config: required env var %s is not set", p.name)
		}
	}
	return nil
}
