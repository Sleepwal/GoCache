package cache

import (
	"strings"
	"testing"
)

// TestMemoryCache_MaxMemory 测试内存限制
func TestMemoryCache_MaxMemory(t *testing.T) {
	// 创建限制为 1KB 的缓存
	c := New(WithMaxMemory(1024))

	// 添加一些数据
	c.Set("key1", strings.Repeat("a", 200), 0)
	c.Set("key2", strings.Repeat("b", 200), 0)
	c.Set("key3", strings.Repeat("c", 200), 0)

	// 验证内存使用
	used := c.UsedMemory()
	if used <= 0 {
		t.Errorf("expected used memory > 0, got %d", used)
	}

	// 继续添加直到触发淘汰
	c.Set("key4", strings.Repeat("d", 200), 0)
	c.Set("key5", strings.Repeat("e", 200), 0)

	// 验证有些项被淘汰
	count := c.Count()
	if count > 5 {
		t.Errorf("expected some items evicted, got %d items", count)
	}
}

// TestMemoryCache_UsedMemory 测试 UsedMemory 方法
func TestMemoryCache_UsedMemory(t *testing.T) {
	c := New()

	initial := c.UsedMemory()
	if initial != 0 {
		t.Errorf("expected initial used memory 0, got %d", initial)
	}

	c.Set("key", "value", 0)

	used := c.UsedMemory()
	if used <= 0 {
		t.Errorf("expected used memory > 0 after set, got %d", used)
	}

	c.Clear()

	used = c.UsedMemory()
	if used != 0 {
		t.Errorf("expected used memory 0 after clear, got %d", used)
	}
}

// TestMemoryCache_EvictionCallback_WithMaxMemory 测试内存限制触发回调
func TestMemoryCache_EvictionCallback_WithMaxMemory(t *testing.T) {
	evictedKeys := make([]string, 0)

	callback := func(key string, value any, reason EvictionReason) {
		if reason == CapacityEvicted {
			evictedKeys = append(evictedKeys, key)
		}
	}

	c := New(WithMaxMemory(500), WithEvictionCallback(callback))

	// 添加数据直到触发淘汰
	for i := 0; i < 20; i++ {
		c.Set("key"+string(rune(i)), strings.Repeat("x", 100), 0)
	}

	if len(evictedKeys) == 0 {
		t.Error("expected some evictions due to memory limit")
	}
}
