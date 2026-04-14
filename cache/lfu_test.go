package cache

import (
	"sync"
	"testing"
	"time"
)

// TestLFUCache_BasicSetAndGet 测试基本的设置和获取
func TestLFUCache_BasicSetAndGet(t *testing.T) {
	c := NewLFU(10)

	c.Set("name", "GoCache", 0)

	val, found := c.Get("name")
	if !found {
		t.Error("expected key 'name' to be found")
	}
	if val != "GoCache" {
		t.Errorf("expected 'GoCache', got %v", val)
	}
}

// TestLFUCache_GetNotFound 测试获取不存在的键
func TestLFUCache_GetNotFound(t *testing.T) {
	c := NewLFU(10)

	_, found := c.Get("nonexistent")
	if found {
		t.Error("expected key 'nonexistent' to not be found")
	}
}

// TestLFUCache_Delete 测试删除缓存
func TestLFUCache_Delete(t *testing.T) {
	c := NewLFU(10)

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

// TestLFUCache_Exists 测试 Exists 方法
func TestLFUCache_Exists(t *testing.T) {
	c := NewLFU(10)

	c.Set("key1", "value1", 0)

	if !c.Exists("key1") {
		t.Error("expected key1 to exist")
	}

	if c.Exists("nonexistent") {
		t.Error("expected nonexistent key to not exist")
	}
}

// TestLFUCache_Keys 测试 Keys 方法
func TestLFUCache_Keys(t *testing.T) {
	c := NewLFU(10)

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

// TestLFUCache_Clear 测试清空缓存
func TestLFUCache_Clear(t *testing.T) {
	c := NewLFU(10)

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

// TestLFUCache_Count 测试 Count 方法
func TestLFUCache_Count(t *testing.T) {
	c := NewLFU(10)

	if c.Count() != 0 {
		t.Errorf("expected 0 items initially, got %d", c.Count())
	}

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	if c.Count() != 2 {
		t.Errorf("expected 2 items, got %d", c.Count())
	}
}

// TestLFUCache_CapacityLimit 测试容量限制
func TestLFUCache_CapacityLimit(t *testing.T) {
	c := NewLFU(3)

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)
	c.Set("key3", "value3", 0)

	// 添加第4个键，应该淘汰频率最低的项
	c.Set("key4", "value4", 0)

	if c.Count() != 3 {
		t.Errorf("expected 3 items, got %d", c.Count())
	}
}

// TestLFUCache_EvictionPolicy 测试 LFU 淘汰策略
func TestLFUCache_EvictionPolicy(t *testing.T) {
	c := NewLFU(3)

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)

	// 多次访问 "a"，使其频率最高
	c.Get("a")
	c.Get("a")
	c.Get("a")

	// 访问 "b" 一次
	c.Get("b")

	// 添加新键，应该淘汰频率最低的 "c"（只设置了1次，没访问过）
	c.Set("d", 4, 0)

	if c.Count() != 3 {
		t.Errorf("expected 3 items, got %d", c.Count())
	}

	_, found := c.Get("c")
	if found {
		t.Error("expected 'c' to be evicted (lowest frequency)")
	}

	_, found = c.Get("a")
	if !found {
		t.Error("expected 'a' to exist (highest frequency)")
	}

	_, found = c.Get("d")
	if !found {
		t.Error("expected 'd' to exist")
	}
}

// TestLFUCache_FrequencyUpdate 测试频率更新
func TestLFUCache_FrequencyUpdate(t *testing.T) {
	c := NewLFU(3)

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)

	// 只访问 "b"
	c.Get("b")
	c.Get("b")

	// 现在频率: a=1, b=3, c=1
	// 添加新键，应该淘汰 a 或 c 中的一个
	c.Set("d", 4, 0)

	if c.Count() != 3 {
		t.Errorf("expected 3 items, got %d", c.Count())
	}

	// b 应该肯定存在
	_, found := c.Get("b")
	if !found {
		t.Error("expected 'b' to exist (highest frequency)")
	}
}

// TestLFUCache_UpdateExisting 测试更新已存在的键
func TestLFUCache_UpdateExisting(t *testing.T) {
	c := NewLFU(3)

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

// TestLFUCache_TTLExpiration 测试 TTL 过期
func TestLFUCache_TTLExpiration(t *testing.T) {
	c := NewLFU(10)

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

// TestLFUCache_TTLNeverExpire 测试永不过期
func TestLFUCache_TTLNeverExpire(t *testing.T) {
	c := NewLFU(10)

	c.Set("permanent", "value", 0)

	time.Sleep(50 * time.Millisecond)

	val, found := c.Get("permanent")
	if !found || val != "value" {
		t.Errorf("expected permanent key to not expire")
	}
}

// TestLFUCache_ZeroCapacity 测试无容量限制
func TestLFUCache_ZeroCapacity(t *testing.T) {
	c := NewLFU(0)

	// 添加大量数据
	for i := 0; i < 100; i++ {
		key := "key" + string(rune(i))
		c.Set(key, i, 0)
	}

	if c.Count() != 100 {
		t.Errorf("expected 100 items with unlimited capacity, got %d", c.Count())
	}
}

// TestLFUCache_ConcurrentAccess 测试并发访问
func TestLFUCache_ConcurrentAccess(t *testing.T) {
	c := NewLFU(100)

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

// TestLFUCache_Decay 测试频率衰减
func TestLFUCache_Decay(t *testing.T) {
	c := NewLFU(10)

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)

	// 增加 "a" 的频率
	for i := 0; i < 10; i++ {
		c.Get("a")
	}

	// 应用衰减
	c.applyDecay()

	freqs := c.GetFrequencies()
	if freqs["a"] <= 0 {
		t.Error("expected 'a' frequency to be positive after decay")
	}

	// 衰减后频率应该降低
	if freqs["a"] >= 11.0 {
		t.Errorf("expected 'a' frequency to be reduced, got %f", freqs["a"])
	}
}

// TestLFUCache_SetDecayFactor 测试设置衰减系数
func TestLFUCache_SetDecayFactor(t *testing.T) {
	c := NewLFU(10)

	// 有效值
	c.SetDecayFactor(0.8)
	if c.decayFactor != 0.8 {
		t.Errorf("expected decay factor 0.8, got %f", c.decayFactor)
	}

	// 无效值应该被忽略
	c.SetDecayFactor(1.5)
	if c.decayFactor == 1.5 {
		t.Error("expected invalid decay factor to be ignored")
	}

	c.SetDecayFactor(-0.1)
	if c.decayFactor == -0.1 {
		t.Error("expected invalid decay factor to be ignored")
	}
}

// TestLFUCache_StartDecay 测试启动定期衰减
func TestLFUCache_StartDecay(t *testing.T) {
	c := NewLFU(10)

	c.Set("a", 1, 0)

	// 增加频率
	for i := 0; i < 10; i++ {
		c.Get("a")
	}

	// 启动衰减（100ms 间隔）
	stop := c.StartDecay(100 * time.Millisecond)

	// 等待衰减发生
	time.Sleep(250 * time.Millisecond)

	freqs := c.GetFrequencies()
	if freqs["a"] >= 11.0 {
		t.Errorf("expected frequency to decay, got %f", freqs["a"])
	}

	// 停止衰减
	stop()
}

// TestLFUCache_TableDriven 表格驱动测试
func TestLFUCache_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		capacity  int
		ops       []func(*LFUCache)
		wantCount int
	}{
		{
			name:     "基本操作",
			capacity: 10,
			ops: []func(*LFUCache){
				func(c *LFUCache) { c.Set("a", 1, 0) },
				func(c *LFUCache) { c.Set("b", 2, 0) },
				func(c *LFUCache) { c.Set("c", 3, 0) },
			},
			wantCount: 3,
		},
		{
			name:     "淘汰测试",
			capacity: 2,
			ops: []func(*LFUCache){
				func(c *LFUCache) { c.Set("a", 1, 0) },
				func(c *LFUCache) { c.Set("b", 2, 0) },
				func(c *LFUCache) { c.Set("c", 3, 0) }, // 应该淘汰 "a" 或 "b"
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewLFU(tt.capacity)
			for _, op := range tt.ops {
				op(c)
			}
			if c.Count() != tt.wantCount {
				t.Errorf("expected %d items, got %d", tt.wantCount, c.Count())
			}
		})
	}
}

// TestLFUCache_FrequencyOrder 测试频率顺序淘汰
func TestLFUCache_FrequencyOrder(t *testing.T) {
	c := NewLFU(3)

	c.Set("low", 1, 0)   // frequency = 1
	c.Set("medium", 2, 0) // frequency = 1
	c.Set("high", 3, 0)   // frequency = 1

	// 增加 "high" 的频率
	for i := 0; i < 5; i++ {
		c.Get("high")
	}

	// 增加 "medium" 的频率
	for i := 0; i < 3; i++ {
		c.Get("medium")
	}

	// 添加新键，应该淘汰 "low"（频率最低）
	c.Set("new", 4, 0)

	if c.Count() != 3 {
		t.Errorf("expected 3 items, got %d", c.Count())
	}

	_, found := c.Get("low")
	if found {
		t.Error("expected 'low' to be evicted (lowest frequency)")
	}

	_, found = c.Get("high")
	if !found {
		t.Error("expected 'high' to exist (highest frequency)")
	}

	_, found = c.Get("medium")
	if !found {
		t.Error("expected 'medium' to exist")
	}

	_, found = c.Get("new")
	if !found {
		t.Error("expected 'new' to exist")
	}
}

// TestLFUCache_ConcurrentWithDecay 测试并发访问与衰减
func TestLFUCache_ConcurrentWithDecay(t *testing.T) {
	c := NewLFU(100)

	stop := c.StartDecay(50 * time.Millisecond)
	defer stop()

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := "key" + string(rune(id*50+j))
				c.Set(key, j, 0)
				c.Get(key)
			}
		}(i)
	}

	wg.Wait()

	if c.Count() != 100 {
		// 由于容量限制，数量可能少于 500
		t.Logf("final count: %d", c.Count())
	}
}
