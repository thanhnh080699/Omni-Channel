package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesAdapterDirectoryLayoutByDefault(t *testing.T) {
	t.Chdir(t.TempDir())
	unsetEnv(t, "WHATSAPP_ADAPTER_DIR")

	cfg := Load()

	if cfg.WhatsAppAdapterDir != "./adapter/whatsapp" {
		t.Fatalf("expected whatsapp adapter dir ./adapter/whatsapp, got %q", cfg.WhatsAppAdapterDir)
	}
}

func TestLoadReadsSharedAPIEnvForAdapterSettings(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	unsetEnv(t, "WHATSAPP_ADAPTER_DIR")

	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("create api dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, ".env"), []byte("WHATSAPP_ADAPTER_DIR=./adapter/custom\nWHATSAPP_ADAPTER_URL=http://localhost:19191\n"), 0o600); err != nil {
		t.Fatalf("write api .env: %v", err)
	}

	cfg := Load()

	if cfg.WhatsAppAdapterDir != "./adapter/custom" {
		t.Fatalf("expected whatsapp adapter dir from api/.env, got %q", cfg.WhatsAppAdapterDir)
	}
	if cfg.WhatsAppAdapterURL != "http://localhost:19191" {
		t.Fatalf("expected whatsapp adapter url from api/.env, got %q", cfg.WhatsAppAdapterURL)
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	original, hadOriginal := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if hadOriginal {
			_ = os.Setenv(key, original)
			return
		}
		_ = os.Unsetenv(key)
	})
}
