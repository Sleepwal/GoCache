package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"GoCache/logger"
)

type FsyncStrategy int

const (
	FsyncAlways   FsyncStrategy = iota
	FsyncEverySec
	FsyncNone
)

type Config struct {
	mu sync.RWMutex

	HTTPPort     int           `json:"http_port" toml:"http_port"`
	RESPPort     int           `json:"resp_port" toml:"resp_port"`
	Password     string        `json:"password" toml:"password"`
	MaxMemory    int64         `json:"max_memory" toml:"max_memory"`
	EvictionInterval time.Duration `json:"eviction_interval" toml:"eviction_interval"`

	AOFEnabled   bool          `json:"aof_enabled" toml:"aof_enabled"`
	AOFPath      string        `json:"aof_path" toml:"aof_path"`
	AOFFsync     FsyncStrategy `json:"aof_fsync" toml:"aof_fsync"`
	AOFRewriteThreshold float64 `json:"aof_rewrite_threshold" toml:"aof_rewrite_threshold"`

	SnapshotEnabled bool   `json:"snapshot_enabled" toml:"snapshot_enabled"`
	SnapshotPath    string `json:"snapshot_path" toml:"snapshot_path"`
	SnapshotInterval time.Duration `json:"snapshot_interval" toml:"snapshot_interval"`

	TLSEnabled  bool   `json:"tls_enabled" toml:"tls_enabled"`
	TLSCertFile string `json:"tls_cert_file" toml:"tls_cert_file"`
	TLSKeyFile  string `json:"tls_key_file" toml:"tls_key_file"`

	LogLevel string `json:"log_level" toml:"log_level"`
}

func DefaultConfig() *Config {
	return &Config{
		HTTPPort:     8080,
		RESPPort:     6379,
		Password:     "",
		MaxMemory:    0,
		EvictionInterval: 10 * time.Second,

		AOFEnabled:   false,
		AOFPath:      "appendonly.aof",
		AOFFsync:     FsyncEverySec,
		AOFRewriteThreshold: 2.0,

		SnapshotEnabled: false,
		SnapshotPath:    "dump.json",
		SnapshotInterval: 5 * time.Minute,

		TLSEnabled:  false,
		TLSCertFile: "",
		TLSKeyFile:  "",

		LogLevel: "INFO",
	}
}

func (c *Config) RequireAuth() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Password != ""
}

func (c *Config) CheckPassword(password string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Password == "" {
		return true
	}

	if password == c.Password {
		return true
	}

	hash := sha256.Sum256([]byte(password))
	hashedHex := hex.EncodeToString(hash[:])
	if hashedHex == c.Password {
		return true
	}

	return false
}

func (c *Config) SetPassword(password string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Password = password
	logger.Info("password updated")
}

func (c *Config) GetHTTPPort() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.HTTPPort
}

func (c *Config) GetRESPPort() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.RESPPort
}

func (c *Config) GetMaxMemory() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.MaxMemory
}

func (c *Config) GetAOFFsync() FsyncStrategy {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AOFFsync
}

func (c *Config) GetAOFRewriteThreshold() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AOFRewriteThreshold
}

func (c *Config) IsTLSEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.TLSEnabled
}

func (c *Config) GetTLSCertFile() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.TLSCertFile
}

func (c *Config) GetTLSKeyFile() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.TLSKeyFile
}

func LoadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := DefaultConfig()

	if len(data) == 0 {
		return cfg, nil
	}

	logger.Info("loading config file", "path", path)

	ext := filepathExt(path)
	switch ext {
	case ".json":
		if err := jsonUnmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s (use .json)", ext)
	}

	logger.Info("config loaded", "path", path, "http_port", cfg.HTTPPort, "resp_port", cfg.RESPPort, "auth_enabled", cfg.RequireAuth())
	return cfg, nil
}

func SaveConfigToFile(cfg *Config, path string) error {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	data, err := jsonMarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.Info("config saved", "path", path)
	return nil
}

func filepathExt(path string) string {
	for i := len(path) - 1; i >= 0 && path[i] != '/' && path[i] != '\\'; i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func jsonMarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}
