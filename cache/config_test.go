package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.HTTPPort != 8080 {
		t.Errorf("expected HTTPPort 8080, got %d", cfg.HTTPPort)
	}
	if cfg.RESPPort != 6379 {
		t.Errorf("expected RESPPort 6379, got %d", cfg.RESPPort)
	}
	if cfg.Password != "" {
		t.Errorf("expected empty Password, got %s", cfg.Password)
	}
	if cfg.AOFEnabled {
		t.Error("expected AOFEnabled false")
	}
	if cfg.TLSEnabled {
		t.Error("expected TLSEnabled false")
	}
	if cfg.AOFFsync != FsyncEverySec {
		t.Errorf("expected FsyncEverySec, got %d", cfg.AOFFsync)
	}
	if cfg.AOFRewriteThreshold != 2.0 {
		t.Errorf("expected rewrite threshold 2.0, got %f", cfg.AOFRewriteThreshold)
	}
}

func TestConfig_RequireAuth(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.RequireAuth() {
		t.Error("should not require auth with empty password")
	}

	cfg.Password = "secret"
	if !cfg.RequireAuth() {
		t.Error("should require auth with non-empty password")
	}
}

func TestConfig_CheckPassword_Plaintext(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Password = "mypassword"

	if !cfg.CheckPassword("mypassword") {
		t.Error("should accept correct plaintext password")
	}
	if cfg.CheckPassword("wrongpassword") {
		t.Error("should reject wrong password")
	}
}

func TestConfig_CheckPassword_SHA256(t *testing.T) {
	cfg := DefaultConfig()
	password := "mypassword"
	hash := sha256.Sum256([]byte(password))
	cfg.Password = hex.EncodeToString(hash[:])

	if !cfg.CheckPassword(password) {
		t.Error("should accept correct SHA256 hashed password")
	}
	if cfg.CheckPassword("wrongpassword") {
		t.Error("should reject wrong password")
	}
}

func TestConfig_SetPassword(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetPassword("newpass")

	if cfg.Password != "newpass" {
		t.Errorf("expected password 'newpass', got %s", cfg.Password)
	}
}

func TestConfig_TLS(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.IsTLSEnabled() {
		t.Error("TLS should not be enabled by default")
	}

	cfg.TLSEnabled = true
	cfg.TLSCertFile = "cert.pem"
	cfg.TLSKeyFile = "key.pem"

	if !cfg.IsTLSEnabled() {
		t.Error("TLS should be enabled")
	}
	if cfg.GetTLSCertFile() != "cert.pem" {
		t.Errorf("expected cert.pem, got %s", cfg.GetTLSCertFile())
	}
	if cfg.GetTLSKeyFile() != "key.pem" {
		t.Errorf("expected key.pem, got %s", cfg.GetTLSKeyFile())
	}
}

func TestConfig_GetPorts(t *testing.T) {
	cfg := DefaultConfig()
	cfg.HTTPPort = 9090
	cfg.RESPPort = 6380

	if cfg.GetHTTPPort() != 9090 {
		t.Errorf("expected HTTP port 9090, got %d", cfg.GetHTTPPort())
	}
	if cfg.GetRESPPort() != 6380 {
		t.Errorf("expected RESP port 6380, got %d", cfg.GetRESPPort())
	}
}

func TestConfig_GetMaxMemory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxMemory = 1024 * 1024 * 100

	if cfg.GetMaxMemory() != 1024*1024*100 {
		t.Errorf("expected 100MB, got %d", cfg.GetMaxMemory())
	}
}

func TestLoadConfigFromFile_JSON(t *testing.T) {
	tmpDir := os.TempDir()
	path := filepath.Join(tmpDir, "gocache_test_config.json")
	defer os.Remove(path)

	cfgData := map[string]any{
		"http_port":            9090,
		"resp_port":            6380,
		"password":             "testpass",
		"aof_enabled":          true,
		"aof_path":             "test.aof",
		"aof_fsync":            0,
		"aof_rewrite_threshold": 3.0,
		"tls_enabled":          false,
		"log_level":            "DEBUG",
	}

	data, _ := json.MarshalIndent(cfgData, "", "  ")
	os.WriteFile(path, data, 0644)

	cfg, err := LoadConfigFromFile(path)
	if err != nil {
		t.Fatalf("LoadConfigFromFile failed: %v", err)
	}

	if cfg.HTTPPort != 9090 {
		t.Errorf("expected HTTPPort 9090, got %d", cfg.HTTPPort)
	}
	if cfg.RESPPort != 6380 {
		t.Errorf("expected RESPPort 6380, got %d", cfg.RESPPort)
	}
	if cfg.Password != "testpass" {
		t.Errorf("expected password 'testpass', got %s", cfg.Password)
	}
	if !cfg.AOFEnabled {
		t.Error("expected AOFEnabled true")
	}
	if cfg.AOFPath != "test.aof" {
		t.Errorf("expected AOFPath 'test.aof', got %s", cfg.AOFPath)
	}
	if cfg.LogLevel != "DEBUG" {
		t.Errorf("expected LogLevel DEBUG, got %s", cfg.LogLevel)
	}
}

func TestLoadConfigFromFile_Empty(t *testing.T) {
	tmpDir := os.TempDir()
	path := filepath.Join(tmpDir, "gocache_test_empty.json")
	defer os.Remove(path)

	os.WriteFile(path, []byte{}, 0644)

	cfg, err := LoadConfigFromFile(path)
	if err != nil {
		t.Fatalf("LoadConfigFromFile failed: %v", err)
	}

	if cfg.HTTPPort != 8080 {
		t.Errorf("expected default HTTPPort 8080, got %d", cfg.HTTPPort)
	}
}

func TestLoadConfigFromFile_NotFound(t *testing.T) {
	_, err := LoadConfigFromFile("nonexistent_config.json")
	if err == nil {
		t.Error("expected error for nonexistent config file")
	}
}

func TestSaveConfigToFile(t *testing.T) {
	tmpDir := os.TempDir()
	path := filepath.Join(tmpDir, "gocache_test_save.json")
	defer os.Remove(path)

	cfg := DefaultConfig()
	cfg.HTTPPort = 7070
	cfg.Password = "savedpass"

	err := SaveConfigToFile(cfg, path)
	if err != nil {
		t.Fatalf("SaveConfigToFile failed: %v", err)
	}

	loaded, err := LoadConfigFromFile(path)
	if err != nil {
		t.Fatalf("LoadConfigFromFile failed: %v", err)
	}

	if loaded.HTTPPort != 7070 {
		t.Errorf("expected HTTPPort 7070, got %d", loaded.HTTPPort)
	}
	if loaded.Password != "savedpass" {
		t.Errorf("expected password 'savedpass', got %s", loaded.Password)
	}
}

func TestFsyncStrategy(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.GetAOFFsync() != FsyncEverySec {
		t.Errorf("expected FsyncEverySec, got %d", cfg.GetAOFFsync())
	}

	cfg.AOFFsync = FsyncAlways
	if cfg.GetAOFFsync() != FsyncAlways {
		t.Errorf("expected FsyncAlways, got %d", cfg.GetAOFFsync())
	}

	cfg.AOFFsync = FsyncNone
	if cfg.GetAOFFsync() != FsyncNone {
		t.Errorf("expected FsyncNone, got %d", cfg.GetAOFFsync())
	}
}

func TestConfig_AOFRewriteThreshold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AOFRewriteThreshold = 5.0

	if cfg.GetAOFRewriteThreshold() != 5.0 {
		t.Errorf("expected 5.0, got %f", cfg.GetAOFRewriteThreshold())
	}
}

func TestConfig_EvictionInterval(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.EvictionInterval != 10*time.Second {
		t.Errorf("expected 10s eviction interval, got %v", cfg.EvictionInterval)
	}
}

func TestConfig_Snapshot(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.SnapshotEnabled {
		t.Error("snapshot should be disabled by default")
	}
	if cfg.SnapshotInterval != 5*time.Minute {
		t.Errorf("expected 5m snapshot interval, got %v", cfg.SnapshotInterval)
	}
}
