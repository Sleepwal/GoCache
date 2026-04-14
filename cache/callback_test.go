package cache

import (
	"sync"
	"testing"
	"time"
)

// TestMemoryCache_EvictionCallback 测试 MemoryCache 回调函数
func TestMemoryCache_EvictionCallback(t *testing.T) {
	var mu sync.Mutex
	evictedKeys := make([]string, 0)
	evictedValues := make([]any, 0)
	evictedReasons := make([]EvictionReason, 0)

	callback := func(key string, value any, reason EvictionReason) {
		mu.Lock()
		defer mu.Unlock()
		evictedKeys = append(evictedKeys, key)
		evictedValues = append(evictedValues, value)
		evictedReasons = append(evictedReasons, reason)
	}

	c := New(WithEvictionCallback(callback))

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	// 手动删除
	c.Delete("key1")

	mu.Lock()
	if len(evictedKeys) != 1 {
		t.Errorf("expected 1 evicted key, got %d", len(evictedKeys))
	}
	if evictedKeys[0] != "key1" {
		t.Errorf("expected evicted key 'key1', got %s", evictedKeys[0])
	}
	if evictedValues[0] != "value1" {
		t.Errorf("expected evicted value 'value1', got %v", evictedValues[0])
	}
	if evictedReasons[0] != Manual {
		t.Errorf("expected eviction reason 'manual', got %s", evictedReasons[0])
	}
	mu.Unlock()
}

// TestMemoryCache_TTLExpiryCallback 测试 TTL 过期回调
func TestMemoryCache_TTLExpiryCallback(t *testing.T) {
	var mu sync.Mutex
	evictedKeys := make([]string, 0)
	evictedReasons := make([]EvictionReason, 0)

	callback := func(key string, value any, reason EvictionReason) {
		mu.Lock()
		defer mu.Unlock()
		evictedKeys = append(evictedKeys, key)
		evictedReasons = append(evictedReasons, reason)
	}

	c := New(WithEvictionCallback(callback))

	c.Set("temp", "temp_value", 50*time.Millisecond)

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	// 手动触发清理
	c.DeleteExpired()

	mu.Lock()
	if len(evictedKeys) != 1 {
		t.Errorf("expected 1 evicted key, got %d", len(evictedKeys))
	}
	if evictedKeys[0] != "temp" {
		t.Errorf("expected evicted key 'temp', got %s", evictedKeys[0])
	}
	if evictedReasons[0] != TTLExpired {
		t.Errorf("expected eviction reason 'ttl_expired', got %s", evictedReasons[0])
	}
	mu.Unlock()
}

// TestLRUCache_EvictionCallback 测试 LRU 回调函数
func TestLRUCache_EvictionCallback(t *testing.T) {
	var mu sync.Mutex
	evictedKeys := make([]string, 0)
	evictedReasons := make([]EvictionReason, 0)

	callback := func(key string, value any, reason EvictionReason) {
		mu.Lock()
		defer mu.Unlock()
		evictedKeys = append(evictedKeys, key)
		evictedReasons = append(evictedReasons, reason)
	}

	c := NewLRU(2, WithLRUEvictionCallback(callback))

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	// 添加第3个键，应该淘汰 key1（最久未使用）
	c.Set("key3", "value3", 0)

	mu.Lock()
	if len(evictedKeys) != 1 {
		t.Errorf("expected 1 evicted key, got %d", len(evictedKeys))
	}
	if evictedKeys[0] != "key1" {
		t.Errorf("expected evicted key 'key1', got %s", evictedKeys[0])
	}
	if evictedReasons[0] != CapacityEvicted {
		t.Errorf("expected eviction reason 'capacity_evicted', got %s", evictedReasons[0])
	}
	mu.Unlock()
}

// TestLRUCache_ManualDeleteCallback 测试 LRU 手动删除回调
func TestLRUCache_ManualDeleteCallback(t *testing.T) {
	var mu sync.Mutex
	evictedKeys := make([]string, 0)
	evictedReasons := make([]EvictionReason, 0)

	callback := func(key string, value any, reason EvictionReason) {
		mu.Lock()
		defer mu.Unlock()
		evictedKeys = append(evictedKeys, key)
		evictedReasons = append(evictedReasons, reason)
	}

	c := NewLRU(10, WithLRUEvictionCallback(callback))

	c.Set("key1", "value1", 0)
	c.Delete("key1")

	mu.Lock()
	if len(evictedKeys) != 1 {
		t.Errorf("expected 1 evicted key, got %d", len(evictedKeys))
	}
	if evictedReasons[0] != Manual {
		t.Errorf("expected eviction reason 'manual', got %s", evictedReasons[0])
	}
	mu.Unlock()
}

// TestLFUCache_EvictionCallback 测试 LFU 回调函数
func TestLFUCache_EvictionCallback(t *testing.T) {
	var mu sync.Mutex
	evictedKeys := make([]string, 0)
	evictedReasons := make([]EvictionReason, 0)

	callback := func(key string, value any, reason EvictionReason) {
		mu.Lock()
		defer mu.Unlock()
		evictedKeys = append(evictedKeys, key)
		evictedReasons = append(evictedReasons, reason)
	}

	c := NewLFU(2, WithLFUEvictionCallback(callback))

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	// 添加第3个键，应该淘汰频率最低的项
	c.Set("key3", "value3", 0)

	mu.Lock()
	if len(evictedKeys) != 1 {
		t.Errorf("expected 1 evicted key, got %d", len(evictedKeys))
	}
	if evictedReasons[0] != CapacityEvicted {
		t.Errorf("expected eviction reason 'capacity_evicted', got %s", evictedReasons[0])
	}
	mu.Unlock()
}

// TestLFUCache_ManualDeleteCallback 测试 LFU 手动删除回调
func TestLFUCache_ManualDeleteCallback(t *testing.T) {
	var mu sync.Mutex
	evictedKeys := make([]string, 0)
	evictedReasons := make([]EvictionReason, 0)

	callback := func(key string, value any, reason EvictionReason) {
		mu.Lock()
		defer mu.Unlock()
		evictedKeys = append(evictedKeys, key)
		evictedReasons = append(evictedReasons, reason)
	}

	c := NewLFU(10, WithLFUEvictionCallback(callback))

	c.Set("key1", "value1", 0)
	c.Delete("key1")

	mu.Lock()
	if len(evictedKeys) != 1 {
		t.Errorf("expected 1 evicted key, got %d", len(evictedKeys))
	}
	if evictedReasons[0] != Manual {
		t.Errorf("expected eviction reason 'manual', got %s", evictedReasons[0])
	}
	mu.Unlock()
}

// TestEvictionCallback_Reasons 测试所有驱逐原因
func TestEvictionCallback_Reasons(t *testing.T) {
	// 测试 String 方法
	if Manual.String() != "manual" {
		t.Errorf("expected 'manual', got %s", Manual.String())
	}
	if TTLExpired.String() != "ttl_expired" {
		t.Errorf("expected 'ttl_expired', got %s", TTLExpired.String())
	}
	if CapacityEvicted.String() != "capacity_evicted" {
		t.Errorf("expected 'capacity_evicted', got %s", CapacityEvicted.String())
	}
}

// TestEvictionCallback_NoCallback 测试无回调时正常工作
func TestEvictionCallback_NoCallback(t *testing.T) {
	// 不设置回调
	c := New()

	c.Set("key1", "value1", 0)
	c.Delete("key1") // 不应该 panic

	if c.Count() != 0 {
		t.Error("expected cache to be empty after delete")
	}
}

// TestEvictionCallback_MultipleEvictions 测试多次驱逐
func TestEvictionCallback_MultipleEvictions(t *testing.T) {
	var mu sync.Mutex
	evictionCount := 0

	callback := func(key string, value any, reason EvictionReason) {
		mu.Lock()
		defer mu.Unlock()
		evictionCount++
	}

	c := NewLRU(2, WithLRUEvictionCallback(callback))

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0) // 驱逐 a
	c.Set("d", 4, 0) // 驱逐 b

	mu.Lock()
	if evictionCount != 2 {
		t.Errorf("expected 2 evictions, got %d", evictionCount)
	}
	mu.Unlock()
}
