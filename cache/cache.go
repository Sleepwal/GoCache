package cache

import (
	"sync"
	"time"
)

// EvictionReason 缓存项被移除的原因
type EvictionReason int

const (
	Manual          EvictionReason = iota // 手动删除
	TTLExpired                            // TTL 过期
	CapacityEvicted                       // 容量淘汰
)

func (r EvictionReason) String() string {
	switch r {
	case Manual:
		return "manual"
	case TTLExpired:
		return "ttl_expired"
	case CapacityEvicted:
		return "capacity_evicted"
	default:
		return "unknown"
	}
}

// EvictionCallback 缓存项被移除时的回调函数
type EvictionCallback func(key string, value any, reason EvictionReason)

// Item 缓存项
type Item struct {
	Value      any
	Expiration int64 // 过期时间戳(纳秒),0表示永不过期
}

// IsExpired 检查缓存项是否过期
func (item *Item) IsExpired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

// Cache 缓存接口
type Cache interface {
	Set(key string, value any, ttl time.Duration)
	Get(key string) (any, bool)
	Delete(key string) bool
	Exists(key string) bool
	Keys() []string
	Clear()
	Count() int
}

// MemoryCache 内存缓存实现
type MemoryCache struct {
	items    map[string]*Item
	mu       sync.RWMutex
	Stats    *Stats           // 统计指标
	onEvict  EvictionCallback // 移除回调
}

// Option 缓存配置选项
type Option func(*MemoryCache)

// WithEvictionCallback 设置移除回调选项
func WithEvictionCallback(callback EvictionCallback) Option {
	return func(c *MemoryCache) {
		c.onEvict = callback
	}
}

// New 创建一个新的内存缓存
func New(opts ...Option) *MemoryCache {
	c := &MemoryCache{
		items: make(map[string]*Item),
		Stats: &Stats{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Set 添加或更新缓存项
func (c *MemoryCache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Stats.Sets.Add(1)

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	c.items[key] = &Item{
		Value:      value,
		Expiration: expiration,
	}
}

// Get 获取缓存项
func (c *MemoryCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		c.Stats.Misses.Add(1)
		c.Stats.TTLMisses.Add(1)
		return nil, false
	}

	if item.IsExpired() {
		c.Stats.Misses.Add(1)
		c.Stats.TTLMisses.Add(1)
		c.Stats.ExpiredCount.Add(1)
		return nil, false
	}

	c.Stats.Hits.Add(1)
	c.Stats.TTLHits.Add(1)
	return item.Value, true
}

// Delete 删除缓存项
func (c *MemoryCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Stats.Deletes.Add(1)

	item, found := c.items[key]
	if !found {
		return false
	}

	delete(c.items, key)

	// 触发回调
	if c.onEvict != nil {
		c.onEvict(key, item.Value, Manual)
	}

	return true
}

// Exists 检查键是否存在(包括是否过期)
func (c *MemoryCache) Exists(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return false
	}

	return !item.IsExpired()
}

// Keys 返回所有未过期的键
func (c *MemoryCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.items))
	now := time.Now().UnixNano()

	for key, item := range c.items {
		if item.Expiration == 0 || now <= item.Expiration {
			keys = append(keys, key)
		}
	}

	return keys
}

// Clear 清空所有缓存
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*Item)
}

// Count 返回缓存项数量(包括已过期的)
func (c *MemoryCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}
