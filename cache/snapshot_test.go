package cache

import (
	"os"
	"testing"
	"time"
)

// TestMemoryCache_SaveAndLoadFile 测试 MemoryCache 保存和加载
func TestMemoryCache_SaveAndLoadFile(t *testing.T) {
	c := New()

	// 设置一些缓存项
	c.Set("key1", "value1", 0)
	c.Set("key2", 42, 0)
	c.Set("key3", 3.14, 0)

	// 保存到文件
	err := c.SaveToFile("/tmp/test_cache.json")
	if err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}
	defer os.Remove("/tmp/test_cache.json")

	// 创建新缓存并加载
	c2 := New()
	err = c2.LoadFromFile("/tmp/test_cache.json")
	if err != nil {
		t.Fatalf("failed to load cache: %v", err)
	}

	// 验证数据
	val, found := c2.Get("key1")
	if !found || val != "value1" {
		t.Errorf("expected 'value1', got %v", val)
	}

	val, found = c2.Get("key2")
	if !found {
		t.Errorf("expected 42, got %v", val)
	}
	// JSON 反序列化后数字变为 float64
	if valFloat, ok := val.(float64); !ok || valFloat != 42.0 {
		t.Errorf("expected 42.0, got %v", val)
	}

	val, found = c2.Get("key3")
	if !found {
		t.Errorf("expected 3.14, got %v", val)
	}
	if valFloat, ok := val.(float64); !ok || valFloat != 3.14 {
		t.Errorf("expected 3.14, got %v", val)
	}
}

// TestMemoryCache_SaveAndLoadFileGob 测试 gob 格式保存和加载
func TestMemoryCache_SaveAndLoadFileGob(t *testing.T) {
	c := New()

	c.Set("name", "GoCache", 0)
	c.Set("version", "1.0", 0)

	err := c.SaveToFileGob("/tmp/test_cache.gob")
	if err != nil {
		t.Fatalf("failed to save cache with gob: %v", err)
	}
	defer os.Remove("/tmp/test_cache.gob")

	c2 := New()
	err = c2.LoadFromFileGob("/tmp/test_cache.gob")
	if err != nil {
		t.Fatalf("failed to load cache with gob: %v", err)
	}

	val, found := c2.Get("name")
	if !found || val != "GoCache" {
		t.Errorf("expected 'GoCache', got %v", val)
	}
}

// TestMemoryCache_LoadNonExistentFile 测试加载不存在的文件
func TestMemoryCache_LoadNonExistentFile(t *testing.T) {
	c := New()

	err := c.LoadFromFile("/tmp/nonexistent_cache.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// TestLRUCache_SaveAndLoadFile 测试 LRUCache 保存和加载
func TestLRUCache_SaveAndLoadFile(t *testing.T) {
	c := NewLRU(10)

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)

	err := c.SaveToFile("/tmp/test_lru_cache.json")
	if err != nil {
		t.Fatalf("failed to save LRU cache: %v", err)
	}
	defer os.Remove("/tmp/test_lru_cache.json")

	c2 := NewLRU(10)
	err = c2.LoadFromFile("/tmp/test_lru_cache.json")
	if err != nil {
		t.Fatalf("failed to load LRU cache: %v", err)
	}

	val, found := c2.Get("a")
	if !found {
		t.Errorf("expected 1, got %v", val)
	}
	if valFloat, ok := val.(float64); !ok || valFloat != 1.0 {
		t.Errorf("expected 1.0, got %v", val)
	}

	val, found = c2.Get("b")
	if !found {
		t.Errorf("expected 2, got %v", val)
	}
	if valFloat, ok := val.(float64); !ok || valFloat != 2.0 {
		t.Errorf("expected 2.0, got %v", val)
	}
}

// TestLRUCache_SaveAndLoadOrder 测试 LRU 顺序保存
func TestLRUCache_SaveAndLoadOrder(t *testing.T) {
	c := NewLRU(5)

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)

	// 访问 "a" 使其成为最近使用的
	c.Get("a")

	err := c.SaveToFile("/tmp/test_lru_order.json")
	if err != nil {
		t.Fatalf("failed to save LRU cache: %v", err)
	}
	defer os.Remove("/tmp/test_lru_order.json")

	c2 := NewLRU(5)
	err = c2.LoadFromFile("/tmp/test_lru_order.json")
	if err != nil {
		t.Fatalf("failed to load LRU cache: %v", err)
	}

	// 验证容量
	if c2.capacity != 5 {
		t.Errorf("expected capacity 5, got %d", c2.capacity)
	}
}

// TestLFUCache_SaveAndLoadFile 测试 LFUCache 保存和加载
func TestLFUCache_SaveAndLoadFile(t *testing.T) {
	c := NewLFU(10)

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)

	// 增加 "a" 的频率
	c.Get("a")
	c.Get("a")

	err := c.SaveToFile("/tmp/test_lfu_cache.json")
	if err != nil {
		t.Fatalf("failed to save LFU cache: %v", err)
	}
	defer os.Remove("/tmp/test_lfu_cache.json")

	c2 := NewLFU(10)
	err = c2.LoadFromFile("/tmp/test_lfu_cache.json")
	if err != nil {
		t.Fatalf("failed to load LFU cache: %v", err)
	}

	val, found := c2.Get("a")
	if !found {
		t.Errorf("expected 1, got %v", val)
	}
	if valFloat, ok := val.(float64); !ok || valFloat != 1.0 {
		t.Errorf("expected 1.0, got %v", val)
	}

	// 验证频率被保存
	freqs := c2.GetFrequencies()
	if freqs["a"] != 4.0 { // 初始 1 + 2 次 Get + 1 次 Set 更新 = 4
		t.Errorf("expected frequency 4.0 for 'a', got %f", freqs["a"])
	}
}

// TestLFUCache_SaveAndLoadDecayFactor 测试衰减系数保存
func TestLFUCache_SaveAndLoadDecayFactor(t *testing.T) {
	c := NewLFU(10)
	c.SetDecayFactor(0.8)

	c.Set("key", "value", 0)

	err := c.SaveToFile("/tmp/test_lfu_decay.json")
	if err != nil {
		t.Fatalf("failed to save LFU cache: %v", err)
	}
	defer os.Remove("/tmp/test_lfu_decay.json")

	c2 := NewLFU(10)
	err = c2.LoadFromFile("/tmp/test_lfu_decay.json")
	if err != nil {
		t.Fatalf("failed to load LFU cache: %v", err)
	}

	if c2.decayFactor != 0.8 {
		t.Errorf("expected decay factor 0.8, got %f", c2.decayFactor)
	}
}

// TestMemoryCache_SaveEmptyCache 测试保存空缓存
func TestMemoryCache_SaveEmptyCache(t *testing.T) {
	c := New()

	err := c.SaveToFile("/tmp/test_empty_cache.json")
	if err != nil {
		t.Fatalf("failed to save empty cache: %v", err)
	}
	defer os.Remove("/tmp/test_empty_cache.json")

	c2 := New()
	err = c2.LoadFromFile("/tmp/test_empty_cache.json")
	if err != nil {
		t.Fatalf("failed to load empty cache: %v", err)
	}

	if c2.Count() != 0 {
		t.Errorf("expected 0 items, got %d", c2.Count())
	}
}

// TestMemoryCache_SaveWithTTL 测试保存带 TTL 的缓存项
func TestMemoryCache_SaveWithTTL(t *testing.T) {
	c := New()

	c.Set("permanent", "value", 0)
	c.Set("temporary", "value", 1*time.Hour)

	err := c.SaveToFile("/tmp/test_ttl_cache.json")
	if err != nil {
		t.Fatalf("failed to save cache with TTL: %v", err)
	}
	defer os.Remove("/tmp/test_ttl_cache.json")

	c2 := New()
	err = c2.LoadFromFile("/tmp/test_ttl_cache.json")
	if err != nil {
		t.Fatalf("failed to load cache with TTL: %v", err)
	}

	// 永久项应该存在
	_, found := c2.Get("permanent")
	if !found {
		t.Error("expected 'permanent' key to exist")
	}

	// 临时项也应该存在（过期时间被保存）
	_, found = c2.Get("temporary")
	if !found {
		t.Error("expected 'temporary' key to exist")
	}
}

// TestMemoryCache_SaveInvalidJSON 测试加载无效 JSON 文件
func TestMemoryCache_SaveInvalidJSON(t *testing.T) {
	// 创建无效 JSON 文件
	err := os.WriteFile("/tmp/test_invalid.json", []byte("invalid json{"), 0644)
	if err != nil {
		t.Fatalf("failed to create invalid JSON file: %v", err)
	}
	defer os.Remove("/tmp/test_invalid.json")

	c := New()
	err = c.LoadFromFile("/tmp/test_invalid.json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
