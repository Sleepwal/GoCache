package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tmpFile(name string) string {
	return filepath.Join(os.TempDir(), name)
}

func TestMemoryCache_SaveAndLoadFile(t *testing.T) {
	c := New()
	c.Set("key1", "value1", 0)
	c.Set("key2", 42, 0)
	c.Set("key3", 3.14, 0)

	path := tmpFile("test_cache.json")
	err := c.SaveToFile(path)
	if err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}
	defer os.Remove(path)

	c2 := New()
	if err := c2.LoadFromFile(path); err != nil {
		t.Fatalf("failed to load cache: %v", err)
	}

	val, found := c2.Get("key1")
	if !found || val != "value1" {
		t.Errorf("expected 'value1', got %v", val)
	}
	val, found = c2.Get("key2")
	if !found {
		t.Errorf("expected 42, got %v", val)
	}
	if vf, ok := val.(float64); !ok || vf != 42.0 {
		t.Errorf("expected 42.0, got %v", val)
	}
}

func TestMemoryCache_SaveAndLoadFileGob(t *testing.T) {
	c := New()
	c.Set("name", "GoCache", 0)
	c.Set("version", "1.0", 0)

	path := tmpFile("test_cache.gob")
	err := c.SaveToFileGob(path)
	if err != nil {
		t.Fatalf("failed to save cache with gob: %v", err)
	}
	defer os.Remove(path)

	c2 := New()
	if err := c2.LoadFromFileGob(path); err != nil {
		t.Fatalf("failed to load cache with gob: %v", err)
	}

	val, found := c2.Get("name")
	if !found || val != "GoCache" {
		t.Errorf("expected 'GoCache', got %v", val)
	}
}

func TestMemoryCache_LoadNonExistentFile(t *testing.T) {
	c := New()
	err := c.LoadFromFile(tmpFile("nonexistent_cache.json"))
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLRUCache_SaveAndLoadFile(t *testing.T) {
	c := NewLRU(10)
	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)

	path := tmpFile("test_lru_cache.json")
	err := c.SaveToFile(path)
	if err != nil {
		t.Fatalf("failed to save LRU cache: %v", err)
	}
	defer os.Remove(path)

	c2 := NewLRU(10)
	if err := c2.LoadFromFile(path); err != nil {
		t.Fatalf("failed to load LRU cache: %v", err)
	}

	val, found := c2.Get("a")
	if !found {
		t.Errorf("expected 1, got %v", val)
	}
	if vf, ok := val.(float64); !ok || vf != 1.0 {
		t.Errorf("expected 1.0, got %v", val)
	}
}

func TestLRUCache_SaveAndLoadOrder(t *testing.T) {
	c := NewLRU(5)
	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)
	c.Get("a")

	path := tmpFile("test_lru_order.json")
	err := c.SaveToFile(path)
	if err != nil {
		t.Fatalf("failed to save LRU cache: %v", err)
	}
	defer os.Remove(path)

	c2 := NewLRU(5)
	if err := c2.LoadFromFile(path); err != nil {
		t.Fatalf("failed to load LRU cache: %v", err)
	}
	if c2.capacity != 5 {
		t.Errorf("expected capacity 5, got %d", c2.capacity)
	}
}

func TestLFUCache_SaveAndLoadFile(t *testing.T) {
	c := NewLFU(10)
	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)
	c.Get("a")
	c.Get("a")

	path := tmpFile("test_lfu_cache.json")
	err := c.SaveToFile(path)
	if err != nil {
		t.Fatalf("failed to save LFU cache: %v", err)
	}
	defer os.Remove(path)

	c2 := NewLFU(10)
	if err := c2.LoadFromFile(path); err != nil {
		t.Fatalf("failed to load LFU cache: %v", err)
	}

	val, found := c2.Get("a")
	if !found {
		t.Errorf("expected 1, got %v", val)
	}
	if vf, ok := val.(float64); !ok || vf != 1.0 {
		t.Errorf("expected 1.0, got %v", val)
	}
	freqs := c2.GetFrequencies()
	if freqs["a"] != 4.0 {
		t.Errorf("expected frequency 4.0 for 'a', got %f", freqs["a"])
	}
}

func TestLFUCache_SaveAndLoadDecayFactor(t *testing.T) {
	c := NewLFU(10)
	c.SetDecayFactor(0.8)
	c.Set("key", "value", 0)

	path := tmpFile("test_lfu_decay.json")
	err := c.SaveToFile(path)
	if err != nil {
		t.Fatalf("failed to save LFU cache: %v", err)
	}
	defer os.Remove(path)

	c2 := NewLFU(10)
	if err := c2.LoadFromFile(path); err != nil {
		t.Fatalf("failed to load LFU cache: %v", err)
	}
	if c2.decayFactor != 0.8 {
		t.Errorf("expected decay factor 0.8, got %f", c2.decayFactor)
	}
}

func TestMemoryCache_SaveEmptyCache(t *testing.T) {
	c := New()

	path := tmpFile("test_empty_cache.json")
	err := c.SaveToFile(path)
	if err != nil {
		t.Fatalf("failed to save empty cache: %v", err)
	}
	defer os.Remove(path)

	c2 := New()
	if err := c2.LoadFromFile(path); err != nil {
		t.Fatalf("failed to load empty cache: %v", err)
	}
	if c2.Count() != 0 {
		t.Errorf("expected 0 items, got %d", c2.Count())
	}
}

func TestMemoryCache_SaveWithTTL(t *testing.T) {
	c := New()
	c.Set("permanent", "value", 0)
	c.Set("temporary", "value", 1*time.Hour)

	path := tmpFile("test_ttl_cache.json")
	err := c.SaveToFile(path)
	if err != nil {
		t.Fatalf("failed to save cache with TTL: %v", err)
	}
	defer os.Remove(path)

	c2 := New()
	if err := c2.LoadFromFile(path); err != nil {
		t.Fatalf("failed to load cache with TTL: %v", err)
	}

	if _, found := c2.Get("permanent"); !found {
		t.Error("expected 'permanent' key to exist")
	}
	if _, found := c2.Get("temporary"); !found {
		t.Error("expected 'temporary' key to exist")
	}
}

func TestMemoryCache_SaveInvalidJSON(t *testing.T) {
	path := tmpFile("test_invalid.json")
	err := os.WriteFile(path, []byte("invalid json{"), 0644)
	if err != nil {
		t.Fatalf("failed to create invalid JSON file: %v", err)
	}
	defer os.Remove(path)

	c := New()
	err = c.LoadFromFile(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
