package cache

import (
	"testing"
	"time"
)

// TestLRUCache_BasicSetAndGet 测试基本的设置和获取
func TestLRUCache_BasicSetAndGet(t *testing.T) {
	c := NewLRU(10)

	// 设置缓存
	c.Set("name", "GoCache", 0)

	// 获取缓存
	val, found := c.Get("name")
	if !found {
		t.Error("expected key 'name' to be found")
	}
	if val != "GoCache" {
		t.Errorf("expected 'GoCache', got %v", val)
	}
}

// TestLRUCache_GetNotFound 测试获取不存在的键
func TestLRUCache_GetNotFound(t *testing.T) {
	c := NewLRU(10)

	_, found := c.Get("nonexistent")
	if found {
		t.Error("expected key 'nonexistent' to not be found")
	}
}

// TestLRUCache_Delete 测试删除缓存
func TestLRUCache_Delete(t *testing.T) {
	c := NewLRU(10)

	c.Set("key1", "value1", 0)
	deleted := c.Delete("key1")
	if !deleted {
		t.Error("expected delete to succeed")
	}

	_, found := c.Get("key1")
	if found {
		t.Error("expected key to be deleted")
	}

	// 删除不存在的键
	deleted = c.Delete("nonexistent")
	if deleted {
		t.Error("expected delete to fail for nonexistent key")
	}
}

// TestLRUCache_Exists 测试 Exists 方法
func TestLRUCache_Exists(t *testing.T) {
	c := NewLRU(10)

	c.Set("key1", "value1", 0)

	if !c.Exists("key1") {
		t.Error("expected key1 to exist")
	}

	if c.Exists("nonexistent") {
		t.Error("expected nonexistent key to not exist")
	}
}

// TestLRUCache_Keys 测试 Keys 方法
func TestLRUCache_Keys(t *testing.T) {
	c := NewLRU(10)

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)

	keys := c.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	// 检查键是否存在
	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[key] = true
	}

	if !keySet["a"] || !keySet["b"] || !keySet["c"] {
		t.Errorf("expected keys [a, b, c], got %v", keys)
	}
}

// TestLRUCache_Clear 测试清空缓存
func TestLRUCache_Clear(t *testing.T) {
	c := NewLRU(10)

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)
	c.Clear()

	if c.Count() != 0 {
		t.Errorf("expected 0 items after clear, got %d", c.Count())
	}

	_, found := c.Get("key1")
	if found {
		t.Error("expected cache to be cleared")
	}
}

// TestLRUCache_Count 测试 Count 方法
func TestLRUCache_Count(t *testing.T) {
	c := NewLRU(10)

	if c.Count() != 0 {
		t.Errorf("expected 0 items initially, got %d", c.Count())
	}

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	if c.Count() != 2 {
		t.Errorf("expected 2 items, got %d", c.Count())
	}
}

// TestLRUCache_CapacityLimit 测试容量限制
func TestLRUCache_CapacityLimit(t *testing.T) {
	c := NewLRU(3)

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)
	c.Set("key3", "value3", 0)

	// 添加第4个键，应该淘汰最久未使用的 key1
	c.Set("key4", "value4", 0)

	if c.Count() != 3 {
		t.Errorf("expected 3 items, got %d", c.Count())
	}

	_, found := c.Get("key1")
	if found {
		t.Error("expected key1 to be evicted")
	}

	_, found = c.Get("key4")
	if !found {
		t.Error("expected key4 to exist")
	}
}

// TestLRUCache_EvictionPolicy 测试 LRU 淘汰策略
func TestLRUCache_EvictionPolicy(t *testing.T) {
	c := NewLRU(3)

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)

	// 访问 "a"，使其成为最近使用的
	c.Get("a")

	// 添加新键，应该淘汰最久未使用的 "b"
	c.Set("d", 4, 0)

	if c.Count() != 3 {
		t.Errorf("expected 3 items, got %d", c.Count())
	}

	_, found := c.Get("b")
	if found {
		t.Error("expected 'b' to be evicted (least recently used)")
	}

	_, found = c.Get("a")
	if !found {
		t.Error("expected 'a' to exist (was recently used)")
	}

	_, found = c.Get("d")
	if !found {
		t.Error("expected 'd' to exist")
	}
}

// TestLRUCache_RecentAccess 测试最近访问更新
func TestLRUCache_RecentAccess(t *testing.T) {
	c := NewLRU(3)

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)

	// 依次访问 a, b, c
	c.Get("a")
	c.Get("b")
	c.Get("c")

	// 现在最久未使用的是 "a"
	c.Set("d", 4, 0)

	_, found := c.Get("a")
	if found {
		t.Error("expected 'a' to be evicted")
	}
}

// TestLRUCache_UpdateExisting 测试更新已存在的键
func TestLRUCache_UpdateExisting(t *testing.T) {
	c := NewLRU(3)

	c.Set("key1", "value1", 0)
	c.Set("key1", "value2", 0)

	val, found := c.Get("key1")
	if !found {
		t.Error("expected key1 to exist")
	}
	if val != "value2" {
		t.Errorf("expected 'value2', got %v", val)
	}

	// 更新不应该改变容量
	if c.Count() != 1 {
		t.Errorf("expected 1 item after update, got %d", c.Count())
	}
}

// TestLRUCache_TTLExpiration 测试 TTL 过期
func TestLRUCache_TTLExpiration(t *testing.T) {
	c := NewLRU(10)

	c.Set("temp", "value", 100*time.Millisecond)

	// 立即获取应该成功
	val, found := c.Get("temp")
	if !found || val != "value" {
		t.Errorf("expected 'value', got %v, found=%v", val, found)
	}

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 过期后获取应失败
	_, found = c.Get("temp")
	if found {
		t.Error("expected key to be expired")
	}
}

// TestLRUCache_TTLNeverExpire 测试永不过期
func TestLRUCache_TTLNeverExpire(t *testing.T) {
	c := NewLRU(10)

	c.Set("permanent", "value", 0)

	time.Sleep(50 * time.Millisecond)

	val, found := c.Get("permanent")
	if !found || val != "value" {
		t.Errorf("expected permanent key to not expire")
	}
}

// TestLRUCache_TTLUpdate 测试更新 TTL
func TestLRUCache_TTLUpdate(t *testing.T) {
	c := NewLRU(10)

	c.Set("key", "value", 100*time.Millisecond)
	time.Sleep(50 * time.Millisecond)

	// 更新值并延长 TTL
	c.Set("key", "newvalue", 1*time.Second)

	val, found := c.Get("key")
	if !found || val != "newvalue" {
		t.Errorf("expected 'newvalue', got %v", val)
	}
}

// TestLRUCache_ZeroCapacity 测试无容量限制
func TestLRUCache_ZeroCapacity(t *testing.T) {
	c := NewLRU(0)

	// 添加大量数据
	for i := 0; i < 1000; i++ {
		key := "key" + string(rune(i))
		c.Set(key, i, 0)
	}

	if c.Count() != 1000 {
		t.Errorf("expected 1000 items with unlimited capacity, got %d", c.Count())
	}
}

// TestLRUCache_ConcurrentAccess 测试并发访问
func TestLRUCache_ConcurrentAccess(t *testing.T) {
	c := NewLRU(100)

	done := make(chan bool)

	// 启动多个 goroutine
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := "key" + string(rune(id*100+j))
				c.Set(key, j, 0)
				c.Get(key)
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestLRUCache_TableDriven 表格驱动测试
func TestLRUCache_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		capacity  int
		ops       []func(*LRUCache)
		wantCount int
	}{
		{
			name:     "基本操作",
			capacity: 10,
			ops: []func(*LRUCache){
				func(c *LRUCache) { c.Set("a", 1, 0) },
				func(c *LRUCache) { c.Set("b", 2, 0) },
				func(c *LRUCache) { c.Set("c", 3, 0) },
			},
			wantCount: 3,
		},
		{
			name:     "淘汰测试",
			capacity: 2,
			ops: []func(*LRUCache){
				func(c *LRUCache) { c.Set("a", 1, 0) },
				func(c *LRUCache) { c.Set("b", 2, 0) },
				func(c *LRUCache) { c.Set("c", 3, 0) }, // 应该淘汰 "a"
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewLRU(tt.capacity)
			for _, op := range tt.ops {
				op(c)
			}
			if c.Count() != tt.wantCount {
				t.Errorf("expected %d items, got %d", tt.wantCount, c.Count())
			}
		})
	}
}
