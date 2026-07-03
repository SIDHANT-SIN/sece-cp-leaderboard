package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_Success(t *testing.T) {
	t.Setenv("ADMIN_USERNAME", "test_admin")
	t.Setenv("ADMIN_PASSWORD", "hashed_password_string")
	t.Setenv("MAINTAINER_PASSWORD", "maintainer_secret")
	t.Setenv("PORT", "3000")
	t.Setenv("LOGO_URL", "https://example.com/logo.png")
	t.Setenv("TURSO_DATABASE_URL", "libsql://my-cluster.turso.io")
	t.Setenv("TURSO_AUTH_TOKEN", "ey-sample-token")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	t.Setenv("CRON_SECRET", "cron-token-123")
	t.Setenv("SUPABASE_URL", "https://supabase.co")
	t.Setenv("FOLDER", "production_assets")

	cfg := LoadConfig()

	assert.Equal(t, "test_admin", cfg.AdminUsername)
	assert.Equal(t, "hashed_password_string", cfg.AdminPasswordHash)
	assert.Equal(t, "maintainer_secret", cfg.MaintainerPassword)
	assert.Equal(t, "3000", cfg.Port)
	assert.Equal(t, "https://example.com/logo.png", cfg.Logo)
	assert.Equal(t, "libsql://my-cluster.turso.io", cfg.DBUrl)
	assert.Equal(t, "ey-sample-token", cfg.AuthToken)
	assert.Equal(t, "redis://127.0.0.1:6379/0", cfg.RedisURL)
	assert.Equal(t, "cron-token-123", cfg.CronSecret)
	assert.Equal(t, "https://supabase.co", cfg.SupaBase)
	assert.Equal(t, "production_assets", cfg.FolderName)
}

func TestLoadConfig_FallbackEmpty(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("TURSO_DATABASE_URL", "")

	cfg := LoadConfig()

	assert.Empty(t, cfg.Port)
	assert.Empty(t, cfg.DBUrl)
}
