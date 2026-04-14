package cache

import (
	"testing"
	"time"
)

// TestTypedCache_BasicSetAndGet 测试泛型缓存基本操作
func TestTypedCache_BasicSetAndGet(t *testing.T) {
	mc := New()
	c := NewTypedCache[string](mc)

	c.Set("name", "GoCache", 0)

	val, found := c.Get("name")
	if !found {
		t.Error("expected key 'name' to be found")
	}
	if val != "GoCache" {
		t.Errorf("expected 'GoCache', got %s", val)
	}
}

// TestTypedCache_IntValues 测试整数类型
func TestTypedCache_IntValues(t *testing.T) {
	mc := New()
	c := NewTypedCache[int](mc)

	c.Set("counter", 42, 0)

	val, found := c.Get("counter")
	if !found {
		t.Error("expected key 'counter' to be found")
	}
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

// TestTypedCache_CustomStruct 测试自定义结构体
func TestTypedCache_CustomStruct(t *testing.T) {
	type User struct {
		Name string
		Age  int
	}

	mc := New()
	c := NewTypedCache[User](mc)

	user := User{Name: "Alice", Age: 30}
	c.Set("user1", user, 0)

	val, found := c.Get("user1")
	if !found {
		t.Error("expected key 'user1' to be found")
	}
	if val.Name != "Alice" || val.Age != 30 {
		t.Errorf("expected User{Name: 'Alice', Age: 30}, got %+v", val)
	}
}

// TestTypedCache_NotFound 测试未找到返回零值
func TestTypedCache_NotFound(t *testing.T) {
	mc := New()
	c := NewTypedCache[int](mc)

	val, found := c.Get("nonexistent")
	if found {
		t.Error("expected not to find key")
	}
	if val != 0 {
		t.Errorf("expected zero value, got %d", val)
	}
}

// TestTypedCache_TTL 测试 TTL 过期
func TestTypedCache_TTL(t *testing.T) {
	mc := New()
	c := NewTypedCache[string](mc)

	c.Set("temp", "value", 50*time.Millisecond)

	// 立即获取应该成功
	val, found := c.Get("temp")
	if !found || val != "value" {
		t.Errorf("expected 'value', got %s, found=%v", val, found)
	}

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	_, found = c.Get("temp")
	if found {
		t.Error("expected key to be expired")
	}
}

// TestTypedCache_Delete 测试删除
func TestTypedCache_Delete(t *testing.T) {
	mc := New()
	c := NewTypedCache[string](mc)

	c.Set("key1", "value1", 0)
	c.Delete("key1")

	_, found := c.Get("key1")
	if found {
		t.Error("expected key to be deleted")
	}
}

// TestTypedCache_Keys 测试 Keys 方法
func TestTypedCache_Keys(t *testing.T) {
	mc := New()
	c := NewTypedCache[string](mc)

	c.Set("a", "1", 0)
	c.Set("b", "2", 0)
	c.Set("c", "3", 0)

	keys := c.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

// TestTypedCache_Stats 测试统计信息
func TestTypedCache_Stats(t *testing.T) {
	mc := New()
	c := NewTypedCache[string](mc)

	c.Set("key1", "value1", 0)
	c.Get("key1")
	c.Get("nonexistent")

	stats := c.Stats()
	snapshot := stats.GetSnapshot()

	if snapshot.Sets != 1 {
		t.Errorf("expected 1 set, got %d", snapshot.Sets)
	}
	if snapshot.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", snapshot.Hits)
	}
	if snapshot.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", snapshot.Misses)
	}
}

// TestTypedLRUCache_BasicSetAndGet 测试泛型 LRU 缓存
func TestTypedLRUCache_BasicSetAndGet(t *testing.T) {
	c := NewTypedLRUCache[string](10)

	c.Set("name", "GoCache", 0)

	val, found := c.Get("name")
	if !found {
		t.Error("expected key 'name' to be found")
	}
	if val != "GoCache" {
		t.Errorf("expected 'GoCache', got %s", val)
	}
}

// TestTypedLRUCache_CapacityLimit 测试容量限制
func TestTypedLRUCache_CapacityLimit(t *testing.T) {
	c := NewTypedLRUCache[int](3)

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)
	c.Set("d", 4, 0) // 应该淘汰 a

	if c.Count() != 3 {
		t.Errorf("expected 3 items, got %d", c.Count())
	}

	_, found := c.Get("a")
	if found {
		t.Error("expected 'a' to be evicted")
	}
}

// TestTypedLRUCache_Stats 测试 LRU 统计信息
func TestTypedLRUCache_Stats(t *testing.T) {
	c := NewTypedLRUCache[int](10)

	c.Set("key1", 1, 0)
	c.Get("key1")

	stats := c.Stats()
	snapshot := stats.GetSnapshot()

	if snapshot.Sets != 1 {
		t.Errorf("expected 1 set, got %d", snapshot.Sets)
	}
	if snapshot.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", snapshot.Hits)
	}
}

// TestTypedLFUCache_BasicSetAndGet 测试泛型 LFU 缓存
func TestTypedLFUCache_BasicSetAndGet(t *testing.T) {
	c := NewTypedLFUCache[string](10)

	c.Set("name", "GoCache", 0)

	val, found := c.Get("name")
	if !found {
		t.Error("expected key 'name' to be found")
	}
	if val != "GoCache" {
		t.Errorf("expected 'GoCache', got %s", val)
	}
}

// TestTypedLFUCache_CapacityLimit 测试容量限制
func TestTypedLFUCache_CapacityLimit(t *testing.T) {
	c := NewTypedLFUCache[int](3)

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)
	c.Set("d", 4, 0) // 应该淘汰频率最低的项

	if c.Count() != 3 {
		t.Errorf("expected 3 items, got %d", c.Count())
	}
}

// TestTypedLFUCache_Stats 测试 LFU 统计信息
func TestTypedLFUCache_Stats(t *testing.T) {
	c := NewTypedLFUCache[int](10)

	c.Set("key1", 1, 0)
	c.Get("key1")

	stats := c.Stats()
	snapshot := stats.GetSnapshot()

	if snapshot.Sets != 1 {
		t.Errorf("expected 1 set, got %d", snapshot.Sets)
	}
	if snapshot.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", snapshot.Hits)
	}
}

// TestTypedCache_TypeSafety 测试类型安全
func TestTypedCache_TypeSafety(t *testing.T) {
	mc := New()
	c := NewTypedCache[int](mc)

	c.Set("key1", 123, 0)

	// 直接通过内部缓存获取值，验证类型
	val, found := mc.Get("key1")
	if !found {
		t.Error("expected key to be found")
	}

	intVal, ok := val.(int)
	if !ok {
		t.Error("expected value to be int")
	}
	if intVal != 123 {
		t.Errorf("expected 123, got %d", intVal)
	}
}

// TestTypedCache_Clear 测试清空
func TestTypedCache_Clear(t *testing.T) {
	mc := New()
	c := NewTypedCache[string](mc)

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	c.Clear()

	if c.Count() != 0 {
		t.Errorf("expected 0 items after clear, got %d", c.Count())
	}
}

// TestTypedCache_Exists 测试 Exists
func TestTypedCache_Exists(t *testing.T) {
	mc := New()
	c := NewTypedCache[string](mc)

	c.Set("key1", "value1", 0)

	if !c.Exists("key1") {
		t.Error("expected key1 to exist")
	}

	if c.Exists("nonexistent") {
		t.Error("expected nonexistent key to not exist")
	}
}

// TestTypedLRUCache_WithCallback 测试带回调的泛型 LRU
func TestTypedLRUCache_WithCallback(t *testing.T) {
	evictedKeys := make([]string, 0)

	callback := func(key string, value any, reason EvictionReason) {
		evictedKeys = append(evictedKeys, key)
	}

	c := NewTypedLRUCache[int](2, WithLRUEvictionCallback(callback))

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0) // 应该淘汰 a

	if len(evictedKeys) != 1 {
		t.Errorf("expected 1 evicted key, got %d", len(evictedKeys))
	}
	if evictedKeys[0] != "a" {
		t.Errorf("expected evicted key 'a', got %s", evictedKeys[0])
	}
}

// TestTypedLFUCache_WithCallback 测试带回调的泛型 LFU
func TestTypedLFUCache_WithCallback(t *testing.T) {
	evictedKeys := make([]string, 0)

	callback := func(key string, value any, reason EvictionReason) {
		evictedKeys = append(evictedKeys, key)
	}

	c := NewTypedLFUCache[int](2, WithLFUEvictionCallback(callback))

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0) // 应该淘汰频率最低的项

	if len(evictedKeys) != 1 {
		t.Errorf("expected 1 evicted key, got %d", len(evictedKeys))
	}
}

// TestTypedCache_SliceType 测试切片类型
func TestTypedCache_SliceType(t *testing.T) {
	mc := New()
	c := NewTypedCache[[]int](mc)

	c.Set("numbers", []int{1, 2, 3}, 0)

	val, found := c.Get("numbers")
	if !found {
		t.Error("expected key 'numbers' to be found")
	}

	if len(val) != 3 || val[0] != 1 || val[1] != 2 || val[2] != 3 {
		t.Errorf("expected [1, 2, 3], got %v", val)
	}
}

// TestTypedCache_MapType 测试 map 类型
func TestTypedCache_MapType(t *testing.T) {
	mc := New()
	c := NewTypedCache[map[string]int](mc)

	data := map[string]int{"a": 1, "b": 2}
	c.Set("data", data, 0)

	val, found := c.Get("data")
	if !found {
		t.Error("expected key 'data' to be found")
	}

	if val["a"] != 1 || val["b"] != 2 {
		t.Errorf("expected map{a:1, b:2}, got %v", val)
	}
}
