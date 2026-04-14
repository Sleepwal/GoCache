package cache

import (
	"testing"
	"time"
)

// TestMemoryCache_Stats 测试 MemoryCache 统计指标
func TestMemoryCache_Stats(t *testing.T) {
	c := New()

	// 初始状态
	snapshot := c.Stats.GetSnapshot()
	if snapshot.Hits != 0 || snapshot.Misses != 0 {
		t.Error("expected initial stats to be zero")
	}

	// 设置缓存项
	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	snapshot = c.Stats.GetSnapshot()
	if snapshot.Sets != 2 {
		t.Errorf("expected 2 sets, got %d", snapshot.Sets)
	}

	// 命中
	c.Get("key1")
	c.Get("key2")

	snapshot = c.Stats.GetSnapshot()
	if snapshot.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", snapshot.Hits)
	}

	// 未命中
	c.Get("nonexistent")

	snapshot = c.Stats.GetSnapshot()
	if snapshot.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", snapshot.Misses)
	}

	// 删除
	c.Delete("key1")

	snapshot = c.Stats.GetSnapshot()
	if snapshot.Deletes != 1 {
		t.Errorf("expected 1 delete, got %d", snapshot.Deletes)
	}
}

// TestMemoryCache_HitRate 测试命中率计算
func TestMemoryCache_HitRate(t *testing.T) {
	c := New()

	// 设置并命中
	c.Set("key1", "value1", 0)
	c.Get("key1") // hit
	c.Get("key2") // miss

	snapshot := c.Stats.GetSnapshot()
	expectedRate := 0.5 // 1 hit / 2 total = 0.5 (50%)
	if snapshot.HitRate != expectedRate {
		t.Errorf("expected hit rate %.2f, got %.2f", expectedRate, snapshot.HitRate)
	}
}

// TestMemoryCache_TTLStats 测试 TTL 相关统计
func TestMemoryCache_TTLStats(t *testing.T) {
	c := New()

	c.Set("temp", "value", 50*time.Millisecond)

	// TTL 有效期内命中
	c.Get("temp")

	snapshot := c.Stats.GetSnapshot()
	if snapshot.TTLHits != 1 {
		t.Errorf("expected 1 TTL hit, got %d", snapshot.TTLHits)
	}

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	// TTL 过期后未命中
	c.Get("temp")

	snapshot = c.Stats.GetSnapshot()
	if snapshot.TTLMisses != 1 {
		t.Errorf("expected 1 TTL miss, got %d", snapshot.TTLMisses)
	}

	if snapshot.ExpiredCount < 1 {
		t.Errorf("expected at least 1 expired count, got %d", snapshot.ExpiredCount)
	}
}

// TestMemoryCache_ResetStats 测试重置统计
func TestMemoryCache_ResetStats(t *testing.T) {
	c := New()

	c.Set("key1", "value1", 0)
	c.Get("key1")

	c.Stats.Reset()

	snapshot := c.Stats.GetSnapshot()
	if snapshot.Hits != 0 || snapshot.Sets != 0 {
		t.Error("expected stats to be reset")
	}
}

// TestLRUCache_Stats 测试 LRUCache 统计指标
func TestLRUCache_Stats(t *testing.T) {
	c := NewLRU(10)

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	// 命中
	c.Get("key1")
	c.Get("key2")

	// 未命中
	c.Get("nonexistent")

	snapshot := c.Stats.GetSnapshot()
	if snapshot.Sets != 2 {
		t.Errorf("expected 2 sets, got %d", snapshot.Sets)
	}
	if snapshot.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", snapshot.Hits)
	}
	if snapshot.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", snapshot.Misses)
	}
}

// TestLFUCache_Stats 测试 LFUCache 统计指标
func TestLFUCache_Stats(t *testing.T) {
	c := NewLFU(10)

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	// 命中
	c.Get("key1")
	c.Get("key2")

	// 未命中
	c.Get("nonexistent")

	snapshot := c.Stats.GetSnapshot()
	if snapshot.Sets != 2 {
		t.Errorf("expected 2 sets, got %d", snapshot.Sets)
	}
	if snapshot.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", snapshot.Hits)
	}
	if snapshot.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", snapshot.Misses)
	}
}

// TestStats_HitRateEdgeCases 测试命中率边界情况
func TestStats_HitRateEdgeCases(t *testing.T) {
	stats := &Stats{}

	// 无操作时命中率应为 0
	if stats.HitRate() != 0.0 {
		t.Error("expected 0 hit rate with no operations")
	}

	// 全部命中
	stats.Hits.Add(10)
	if stats.HitRate() != 1.0 {
		t.Error("expected 100% hit rate with all hits")
	}

	// 全部未命中
	stats2 := &Stats{}
	stats2.Misses.Add(10)
	if stats2.HitRate() != 0.0 {
		t.Error("expected 0% hit rate with all misses")
	}

	// 50% 命中率
	stats3 := &Stats{}
	stats3.Hits.Add(5)
	stats3.Misses.Add(5)
	if stats3.HitRate() != 0.5 {
		t.Errorf("expected 50%% hit rate, got %f", stats3.HitRate())
	}
}

// TestStats_TotalOperations 测试总操作数计算
func TestStats_TotalOperations(t *testing.T) {
	stats := &Stats{}
	stats.Hits.Add(10)
	stats.Misses.Add(5)
	stats.Sets.Add(20)
	stats.Deletes.Add(3)

	total := stats.TotalOperations()
	expected := int64(38)
	if total != expected {
		t.Errorf("expected %d total operations, got %d", expected, total)
	}
}
