package cache

import (
	"math"
	"sync"
	"time"

	"GoCache/logger"
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
	LastAccess int64 // 最后访问时间戳(纳秒)
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
	items          map[string]*Item
	mu             sync.RWMutex
	Stats          *Stats           // 统计指标
	onEvict        EvictionCallback // 移除回调
	MaxMemoryBytes int              // 最大内存限制（字节），0表示无限制
	currentBytes   int              // 当前使用的内存（字节）
}

// Option 缓存配置选项
type Option func(*MemoryCache)

// WithEvictionCallback 设置移除回调选项
func WithEvictionCallback(callback EvictionCallback) Option {
	return func(c *MemoryCache) {
		c.onEvict = callback
	}
}

// WithMaxMemory 设置最大内存限制（字节）
func WithMaxMemory(bytes int) Option {
	return func(c *MemoryCache) {
		c.MaxMemoryBytes = bytes
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

	// 计算新项的内存占用
	newItemSize := estimateItemSize(key, value, expiration)

	// 如果已存在，先减去旧项的大小
	if oldItem, exists := c.items[key]; exists {
		c.currentBytes -= estimateItemSize(key, oldItem.Value, oldItem.Expiration)
	}

	// 检查内存限制
	if c.MaxMemoryBytes > 0 {
		for c.currentBytes+newItemSize > c.MaxMemoryBytes && len(c.items) > 0 {
			c.evictOne()
		}
		if c.currentBytes+newItemSize > c.MaxMemoryBytes {
			logger.Warn("cache memory limit reached, cannot store item", "key", key, "current_bytes", c.currentBytes, "max_bytes", c.MaxMemoryBytes, "item_size", newItemSize)
		}
	}

	c.items[key] = &Item{
		Value:      value,
		Expiration: expiration,
		LastAccess: time.Now().UnixNano(),
	}
	c.currentBytes += newItemSize
}

// evictOne 删除最久未访问的项（LRU 淘汰策略）
func (c *MemoryCache) evictOne() {
	var oldestKey string
	var oldestItem *Item
	oldestTime := int64(math.MaxInt64)

	for key, item := range c.items {
		if item.LastAccess < oldestTime {
			oldestTime = item.LastAccess
			oldestKey = key
			oldestItem = item
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
		oldestSize := estimateItemSize(oldestKey, oldestItem.Value, oldestItem.Expiration)
		c.currentBytes -= oldestSize
		logger.Warn("cache item evicted (LRU)", "key", oldestKey, "memory_freed", oldestSize, "current_bytes", c.currentBytes, "max_bytes", c.MaxMemoryBytes)
		if c.onEvict != nil {
			c.onEvict(oldestKey, oldestItem.Value, CapacityEvicted)
		}
	}
}

// estimateItemSize 估算缓存项占用的内存（字节）
func estimateItemSize(key string, value any, expiration int64) int {
	// 基础结构大小
	size := 48 // Item 结构体基础大小
	size += len(key)
	size += 8 // expiration

	// 估算值的大小
	switch v := value.(type) {
	case string:
		size += len(v)
	case int, int64, uint64, float64:
		size += 8
	case bool:
		size += 1
	case []byte:
		size += len(v)
	default:
		// 其他类型估算为基础值
		size += 64
	}

	return size
}

// UsedMemory 返回当前使用的内存（字节）
func (c *MemoryCache) UsedMemory() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentBytes
}

// Clear 清空所有缓存
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*Item)
	c.currentBytes = 0
}

// Get 获取缓存项
func (c *MemoryCache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

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

	item.LastAccess = time.Now().UnixNano()
	c.Stats.Hits.Add(1)
	c.Stats.TTLHits.Add(1)
	return item.Value, true
}

// GetDel 原子地获取值并删除键
func (c *MemoryCache) GetDel(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[key]
	if !found {
		c.Stats.Misses.Add(1)
		c.Stats.TTLMisses.Add(1)
		return nil, false
	}

	if item.IsExpired() {
		delete(c.items, key)
		c.Stats.Misses.Add(1)
		c.Stats.TTLMisses.Add(1)
		c.Stats.ExpiredCount.Add(1)
		if c.onEvict != nil {
			c.onEvict(key, item.Value, TTLExpired)
		}
		return nil, false
	}

	delete(c.items, key)
	c.Stats.Deletes.Add(1)
	if c.onEvict != nil {
		c.onEvict(key, item.Value, Manual)
	}

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

// Scan 游标式迭代键，类似 Redis SCAN
// cursor: 起始游标(从 0 开始)，返回的 nextCursor 为 0 表示迭代完成
// count: 建议返回键数量提示(非严格限制)
// 返回 (nextCursor, keys)
func (c *MemoryCache) Scan(cursor uint64, count int) (uint64, []string) {
	if count <= 0 {
		count = 10
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// 获取所有键(排序以确保游标一致性)
	allKeys := make([]string, 0, len(c.items))
	now := time.Now().UnixNano()
	for key, item := range c.items {
		if item.Expiration == 0 || now <= item.Expiration {
			allKeys = append(allKeys, key)
		}
	}

	// 排序
	for i := 0; i < len(allKeys); i++ {
		for j := i + 1; j < len(allKeys); j++ {
			if allKeys[i] > allKeys[j] {
				allKeys[i], allKeys[j] = allKeys[j], allKeys[i]
			}
		}
	}

	total := uint64(len(allKeys))
	if cursor >= total {
		return 0, []string{}
	}

	// 计算结束位置
	end := cursor + uint64(count)
	if end >= total {
		end = total
	}

	keys := allKeys[cursor:end]
	nextCursor := end

	// 如果已到达末尾，返回 0
	if nextCursor >= total {
		nextCursor = 0
	}

	return nextCursor, keys
}

// Count 返回缓存项数量(包括已过期的)
func (c *MemoryCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

func (c *MemoryCache) Type(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found || item.IsExpired() {
		return "none"
	}

	switch item.Value.(type) {
	case *setData:
		return "set"
	case *listData:
		return "list"
	case *hashData:
		return "hash"
	case *sortedSetData:
		return "zset"
	default:
		return "string"
	}
}

func (c *MemoryCache) Items() map[string]*Item {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*Item, len(c.items))
	for k, v := range c.items {
		result[k] = v
	}
	return result
}
