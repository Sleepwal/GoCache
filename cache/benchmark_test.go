package cache

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkMemoryCache_Set 测试 MemoryCache Set 性能
func BenchmarkMemoryCache_Set(b *testing.B) {
	c := New()
	for i := 0; i < b.N; i++ {
		c.Set(fmt.Sprintf("key%d", i), i, 0)
	}
}

// BenchmarkMemoryCache_Get 测试 MemoryCache Get 性能
func BenchmarkMemoryCache_Get(b *testing.B) {
	c := New()
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("key%d", i), i, 0)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(fmt.Sprintf("key%d", i%1000))
	}
}

// BenchmarkMemoryCache_Concurrent 测试并发性能
func BenchmarkMemoryCache_Concurrent(b *testing.B) {
	c := New()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i)
			c.Set(key, i, 0)
			c.Get(key)
			i++
		}
	})
}

// BenchmarkLRUCache_Set 测试 LRU Set 性能
func BenchmarkLRUCache_Set(b *testing.B) {
	c := NewLRU(10000)
	for i := 0; i < b.N; i++ {
		c.Set(fmt.Sprintf("key%d", i), i, 0)
	}
}

// BenchmarkLRUCache_Get 测试 LRU Get 性能
func BenchmarkLRUCache_Get(b *testing.B) {
	c := NewLRU(10000)
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("key%d", i), i, 0)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(fmt.Sprintf("key%d", i%1000))
	}
}

// BenchmarkLFUCache_Set 测试 LFU Set 性能
func BenchmarkLFUCache_Set(b *testing.B) {
	c := NewLFU(10000)
	for i := 0; i < b.N; i++ {
		c.Set(fmt.Sprintf("key%d", i), i, 0)
	}
}

// BenchmarkLFUCache_Get 测试 LFU Get 性能
func BenchmarkLFUCache_Get(b *testing.B) {
	c := NewLFU(10000)
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("key%d", i), i, 0)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(fmt.Sprintf("key%d", i%1000))
	}
}

// BenchmarkMemoryCache_WithTTL 测试带 TTL 的 Set 性能
func BenchmarkMemoryCache_WithTTL(b *testing.B) {
	c := New()
	ttl := 1 * time.Hour
	for i := 0; i < b.N; i++ {
		c.Set(fmt.Sprintf("key%d", i), i, ttl)
	}
}

// BenchmarkMemoryCache_Delete 测试 Delete 性能
func BenchmarkMemoryCache_Delete(b *testing.B) {
	c := New()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		c.Set(key, i, 0)
		c.Delete(key)
	}
}

// BenchmarkMemoryCache_Keys 测试 Keys 性能
func BenchmarkMemoryCache_Keys(b *testing.B) {
	c := New()
	for i := 0; i < 10000; i++ {
		c.Set(fmt.Sprintf("key%d", i), i, 0)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Keys()
	}
}

// BenchmarkComparison 对比不同缓存实现的性能
func BenchmarkComparison(b *testing.B) {
	b.Run("MemoryCache/Set", func(b *testing.B) {
		c := New()
		for i := 0; i < b.N; i++ {
			c.Set(fmt.Sprintf("key%d", i), i, 0)
		}
	})
	b.Run("LRU/Set", func(b *testing.B) {
		c := NewLRU(100000)
		for i := 0; i < b.N; i++ {
			c.Set(fmt.Sprintf("key%d", i), i, 0)
		}
	})
	b.Run("LFU/Set", func(b *testing.B) {
		c := NewLFU(100000)
		for i := 0; i < b.N; i++ {
			c.Set(fmt.Sprintf("key%d", i), i, 0)
		}
	})
}

// BenchmarkLatency 测试延迟分布
func BenchmarkLatency(b *testing.B) {
	c := New()
	
	// 预热
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("key%d", i), i, 0)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		c.Get(fmt.Sprintf("key%d", i%1000))
		elapsed := time.Since(start)
		if elapsed > time.Millisecond {
			b.Logf("Operation took %v", elapsed)
		}
	}
}
