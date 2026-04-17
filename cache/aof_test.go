package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func aofTmpFile(name string) string {
	return filepath.Join(os.TempDir(), name)
}

func TestAOFLogger_LogAndReplay(t *testing.T) {
	path := aofTmpFile("test_aof.log")
	aof, err := NewAOFLogger(path)
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove(path)
	defer aof.Close()

	aof.LogSet("key1", "value1", 0)
	aof.LogSet("key2", "value2", 0)
	aof.LogSet("key3", "value3", 0)
	aof.Flush()

	c := New()
	if err := aof.Replay(c); err != nil {
		t.Fatalf("failed to replay AOF: %v", err)
	}

	val, found := c.Get("key1")
	if !found || val != "value1" {
		t.Errorf("expected 'value1', got %v", val)
	}
	val, found = c.Get("key2")
	if !found || val != "value2" {
		t.Errorf("expected 'value2', got %v", val)
	}
}

func TestAOFLogger_TypePreservation(t *testing.T) {
	path := aofTmpFile("test_aof_types.log")
	aof, err := NewAOFLogger(path)
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove(path)
	defer aof.Close()

	aof.LogSet("string_key", "hello", 0)
	aof.LogSet("int_key", int64(42), 0)
	aof.LogSet("bool_key", true, 0)
	aof.LogSet("float_key", 3.14, 0)
	aof.Flush()

	c := New()
	if err := aof.Replay(c); err != nil {
		t.Fatalf("failed to replay AOF: %v", err)
	}

	val, found := c.Get("string_key")
	if !found || val != "hello" {
		t.Errorf("expected 'hello', got %v", val)
	}
	val, found = c.Get("int_key")
	if !found {
		t.Error("int_key not found")
	} else if intVal, ok := val.(int64); !ok || intVal != 42 {
		t.Errorf("expected int64(42), got %T(%v)", val, val)
	}
	val, found = c.Get("bool_key")
	if !found {
		t.Error("bool_key not found")
	} else if boolVal, ok := val.(bool); !ok || boolVal != true {
		t.Errorf("expected bool(true), got %T(%v)", val, val)
	}
	val, found = c.Get("float_key")
	if !found {
		t.Error("float_key not found")
	} else if floatVal, ok := val.(float64); !ok || floatVal != 3.14 {
		t.Errorf("expected float64(3.14), got %T(%v)", val, val)
	}
}

func TestAOFLogger_Close(t *testing.T) {
	path := aofTmpFile("test_aof_close.log")
	aof, err := NewAOFLogger(path)
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove(path)

	aof.LogSet("key", "value", 0)
	if err := aof.Close(); err != nil {
		t.Errorf("failed to close AOF logger: %v", err)
	}
	if err := aof.LogSet("key2", "value2", 0); err != nil {
		t.Errorf("expected no error after close, got %v", err)
	}
}

func TestAOFLogger_Rewrite(t *testing.T) {
	path := aofTmpFile("test_aof_rewrite.log")
	aof, err := NewAOFLogger(path)
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove(path)
	defer aof.Close()

	c := New()
	c.Set("key1", "value1", 0)
	c.Set("key2", int64(123), 0)
	c.Set("key3", true, 0)

	if err := aof.Rewrite(c); err != nil {
		t.Fatalf("failed to rewrite AOF: %v", err)
	}

	c2 := New()
	if err := aof.Replay(c2); err != nil {
		t.Fatalf("failed to replay after rewrite: %v", err)
	}
	if c2.Count() != 3 {
		t.Errorf("expected 3 items after rewrite, got %d", c2.Count())
	}
	val, _ := c2.Get("key2")
	if _, ok := val.(int64); !ok {
		t.Errorf("expected int64 type after rewrite, got %T", val)
	}
}

func TestAOFLogger_WithTTL(t *testing.T) {
	path := aofTmpFile("test_aof_ttl.log")
	aof, err := NewAOFLogger(path)
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove(path)
	defer aof.Close()

	aof.LogSet("key1", "value1", 0)
	aof.Flush()

	c := New()
	if err := aof.Replay(c); err != nil {
		t.Fatalf("failed to replay AOF with TTL: %v", err)
	}
	if _, found := c.Get("key1"); !found {
		t.Error("expected 'key1' to exist")
	}
}

func TestAOFLogger_Delete(t *testing.T) {
	path := aofTmpFile("test_aof_delete.log")
	aof, err := NewAOFLogger(path)
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove(path)
	defer aof.Close()

	aof.LogSet("key1", "value1", 0)
	aof.LogDelete("key1")
	aof.Flush()

	c := New()
	if err := aof.Replay(c); err != nil {
		t.Fatalf("failed to replay AOF: %v", err)
	}
	if _, found := c.Get("key1"); found {
		t.Error("expected 'key1' to be deleted")
	}
}

func TestAOFLogger_EmptyFile(t *testing.T) {
	path := aofTmpFile("test_aof_empty.log")
	aof, err := NewAOFLogger(path)
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove(path)
	defer aof.Close()

	c := New()
	if err := aof.Replay(c); err != nil {
		t.Errorf("failed to replay empty AOF: %v", err)
	}
	if c.Count() != 0 {
		t.Errorf("expected 0 items, got %d", c.Count())
	}
}

func TestAOFLogger_ExpiredTTL(t *testing.T) {
	path := aofTmpFile("test_aof_expired.log")
	aof, err := NewAOFLogger(path)
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove(path)
	defer aof.Close()

	pastExpiration := time.Now().Add(-1 * time.Second).UnixNano()
	aof.LogSet("expired_key", "old_value", pastExpiration)
	aof.Flush()

	c := New()
	if err := aof.Replay(c); err != nil {
		t.Fatalf("failed to replay AOF: %v", err)
	}
	if _, found := c.Get("expired_key"); found {
		t.Error("expected expired key to not exist")
	}
	if c.Count() != 0 {
		t.Errorf("expected 0 items, got %d", c.Count())
	}
}

func TestAOFLogger_WithConfig(t *testing.T) {
	path := aofTmpFile("test_aof_config.log")
	c := New()
	c.Set("k1", "v1", 0)

	aof, err := NewAOFLoggerWithConfig(path, FsyncAlways, 2.0, c)
	if err != nil {
		t.Fatalf("failed to create AOF logger with config: %v", err)
	}
	defer os.Remove(path)
	defer aof.Close()

	if aof.fsyncStrategy != FsyncAlways {
		t.Errorf("expected FsyncAlways, got %d", aof.fsyncStrategy)
	}
	if aof.rewriteThreshold != 2.0 {
		t.Errorf("expected threshold 2.0, got %f", aof.rewriteThreshold)
	}
	if aof.cache == nil {
		t.Error("expected cache reference to be set")
	}
}

func TestAOFLogger_FsyncEverySec(t *testing.T) {
	path := aofTmpFile("test_aof_fsync.log")
	c := New()

	aof, err := NewAOFLoggerWithConfig(path, FsyncEverySec, 2.0, c)
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove(path)
	defer aof.Close()

	aof.LogSet("key", "value", 0)
	aof.Flush()

	c2 := New()
	if err := aof.Replay(c2); err != nil {
		t.Fatalf("failed to replay: %v", err)
	}
	if val, found := c2.Get("key"); !found || val != "value" {
		t.Errorf("expected 'value', got %v, found %v", val, found)
	}
}

func TestAOFLogger_FsyncNone(t *testing.T) {
	path := aofTmpFile("test_aof_fsync_none.log")
	c := New()

	aof, err := NewAOFLoggerWithConfig(path, FsyncNone, 2.0, c)
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove(path)
	defer aof.Close()

	aof.LogSet("key", "value", 0)
	aof.Flush()

	c2 := New()
	if err := aof.Replay(c2); err != nil {
		t.Fatalf("failed to replay: %v", err)
	}
	if val, found := c2.Get("key"); !found || val != "value" {
		t.Errorf("expected 'value', got %v, found %v", val, found)
	}
}
