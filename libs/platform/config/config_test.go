package config_test

import (
	"strings"
	"testing"

	"github.com/zosmed/zosmed/libs/platform/config"
)

func TestLoad_MissingRequired(t *testing.T) {
	// No env vars set → Load must return a non-nil error naming a missing var.
	t.Setenv("DB_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("META_APP_SECRET", "")
	t.Setenv("META_VERIFY_TOKEN", "")
	t.Setenv("WA_PHONE", "")
	t.Setenv("PORT", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing env vars, got nil")
	}
	if !strings.Contains(err.Error(), "config:") {
		t.Errorf("error message should contain 'config:', got: %v", err)
	}
}

func TestLoad_AllSet(t *testing.T) {
	t.Setenv("DB_URL", "postgresql://user:pass@localhost:5432/zosmed")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("META_APP_SECRET", "test-secret")
	t.Setenv("META_VERIFY_TOKEN", "test-verify-token")
	t.Setenv("WA_PHONE", "6281234567890")
	t.Setenv("PORT", "9090")

	c, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Port != "9090" {
		t.Errorf("expected Port=9090, got %s", c.Port)
	}
	if c.WAPhone != "6281234567890" {
		t.Errorf("expected WAPhone=6281234567890, got %s", c.WAPhone)
	}
}

func TestLoad_DefaultPort(t *testing.T) {
	t.Setenv("DB_URL", "postgresql://localhost/zosmed")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("META_APP_SECRET", "s")
	t.Setenv("META_VERIFY_TOKEN", "v")
	t.Setenv("WA_PHONE", "628111")
	t.Setenv("PORT", "") // not set

	c, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Port != "8080" {
		t.Errorf("expected default Port=8080, got %s", c.Port)
	}
}

func TestLoad_MissingSpecific(t *testing.T) {
	cases := []struct {
		name    string
		envKey  string
		wantMsg string
	}{
		{"missing DB_URL", "DB_URL", "DB_URL"},
		{"missing REDIS_URL", "REDIS_URL", "REDIS_URL"},
		{"missing META_APP_SECRET", "META_APP_SECRET", "META_APP_SECRET"},
		{"missing META_VERIFY_TOKEN", "META_VERIFY_TOKEN", "META_VERIFY_TOKEN"},
		{"missing WA_PHONE", "WA_PHONE", "WA_PHONE"},
	}

	// Base env — all set.
	base := map[string]string{
		"DB_URL":            "postgresql://localhost/z",
		"REDIS_URL":         "redis://localhost:6379",
		"META_APP_SECRET":   "s",
		"META_VERIFY_TOKEN": "v",
		"WA_PHONE":          "628111",
		"PORT":              "",
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range base {
				t.Setenv(k, v)
			}
			t.Setenv(tc.envKey, "") // unset the one under test

			_, err := config.Load()
			if err == nil {
				t.Fatalf("expected error for missing %s", tc.envKey)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("expected error to mention %q, got: %v", tc.wantMsg, err)
			}
		})
	}
}
