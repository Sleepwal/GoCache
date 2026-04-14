package cache

import (
	"container/heap"
	"sync"
	"time"
)

// lfuItem LFU 缓存项
type lfuItem struct {
	key        string
	value      any
	expiration int64   // 过期时间戳(纳秒), 0表示永不过期
	frequency  float64 // 访问频率
	index      int     // 在堆中的索引位置
}

// frequencyHeap 频率小顶堆
type frequencyHeap []*lfuItem

func (h frequencyHeap) Len() int { return len(h) }
func (h frequencyHeap) Less(i, j int) bool {
	return h[i].frequency < h[j].frequency
}
func (h frequencyHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *frequencyHeap) Push(x any) {
	n := len(*h)
	item := x.(*lfuItem)
	item.index = n
	*h = append(*h, item)
}
func (h *frequencyHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // 避免内存泄漏
	item.index = -1 // 标记为已移除
	*h = old[0 : n-1]
	return item
}

// LFUCache LFU (Least Frequently Used) 缓存实现
// 使用时间衰减机制：访问频率会随时间衰减，更精确地反映访问热度
type LFUCache struct {
	capacity    int
	items       map[string]*lfuItem
	freqHeap    *frequencyHeap
	mu          sync.RWMutex
	Stats       *Stats           // 统计指标
	onEvict     EvictionCallback // 移除回调
	decayFactor float64          // 衰减系数 (默认 0.5)
	decayTicker *time.Ticker     // 衰减定时器
	stopCh      chan struct{}    // 停止信号
}

// LFUCacheOption LFU 缓存配置选项
type LFUCacheOption func(*LFUCache)

// WithLFUEvictionCallback 设置移除回调选项
func WithLFUEvictionCallback(callback EvictionCallback) LFUCacheOption {
	return func(c *LFUCache) {
		c.onEvict = callback
	}
}

// NewLFU 创建一个新的 LFU 缓存
// capacity: 缓存容量，0 表示无限制
func NewLFU(capacity int, opts ...LFUCacheOption) *LFUCache {
	c := &LFUCache{
		capacity:    capacity,
		items:       make(map[string]*lfuItem),
		freqHeap:    &frequencyHeap{},
		Stats:       &Stats{},
		decayFactor: 0.5,
		stopCh:      make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	heap.Init(c.freqHeap)
	return c
}

// Set 添加或更新缓存项
func (c *LFUCache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Stats.Sets.Add(1)

	// 如果键已存在，更新值并增加频率
	if item, exists := c.items[key]; exists {
		item.value = value
		item.frequency += 1.0
		if ttl > 0 {
			item.expiration = time.Now().Add(ttl).UnixNano()
		}
		heap.Fix(c.freqHeap, item.index)
		return
	}

	// 如果缓存已满，删除频率最低的项
	if c.capacity > 0 && len(c.items) >= c.capacity {
		c.removeLowestFrequency()
	}

	// 创建新项
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	newItem := &lfuItem{
		key:        key,
		value:      value,
		expiration: expiration,
		frequency:  1.0,
	}
	c.items[key] = newItem
	heap.Push(c.freqHeap, newItem)
}

// Get 获取缓存项
func (c *LFUCache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if !exists {
		c.Stats.Misses.Add(1)
		c.Stats.TTLMisses.Add(1)
		return nil, false
	}

	// 检查是否过期
	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		c.removeItemWithoutCallback(item)
		c.Stats.Misses.Add(1)
		c.Stats.TTLMisses.Add(1)
		c.Stats.ExpiredCount.Add(1)

		// 触发回调
		if c.onEvict != nil {
			c.onEvict(item.key, item.value, TTLExpired)
		}

		return nil, false
	}

	// 增加访问频率
	item.frequency += 1.0
	heap.Fix(c.freqHeap, item.index)

	c.Stats.Hits.Add(1)
	c.Stats.TTLHits.Add(1)
	return item.value, true
}

// Delete 删除缓存项
func (c *LFUCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Stats.Deletes.Add(1)

	item, exists := c.items[key]
	if !exists {
		return false
	}

	c.removeItemWithoutCallback(item)

	// 触发回调
	if c.onEvict != nil {
		c.onEvict(key, item.value, Manual)
	}

	return true
}

// removeItemWithoutCallback 从缓存和堆中移除指定项（不触发回调，用于内部调用）
func (c *LFUCache) removeItemWithoutCallback(item *lfuItem) {
	delete(c.items, item.key)
	if item.index >= 0 && item.index < c.freqHeap.Len() {
		heap.Remove(c.freqHeap, item.index)
	}
}

// Exists 检查键是否存在（包括是否过期）
func (c *LFUCache) Exists(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if !exists {
		return false
	}

	// 检查是否过期
	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		c.removeItemWithoutCallback(item)

		// 触发回调
		if c.onEvict != nil {
			c.onEvict(item.key, item.value, TTLExpired)
		}

		return false
	}

	return true
}

// Keys 返回所有未过期的键
func (c *LFUCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.items))
	now := time.Now().UnixNano()

	for key, item := range c.items {
		if item.expiration == 0 || now <= item.expiration {
			keys = append(keys, key)
		}
	}

	return keys
}

// Clear 清空所有缓存
func (c *LFUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*lfuItem)
	c.freqHeap = &frequencyHeap{}
	heap.Init(c.freqHeap)
}

// Count 返回缓存项数量（包括已过期的）
func (c *LFUCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// removeItem 从缓存和堆中移除指定项（触发回调）
func (c *LFUCache) removeItem(item *lfuItem) {
	delete(c.items, item.key)
	if item.index >= 0 && item.index < c.freqHeap.Len() {
		heap.Remove(c.freqHeap, item.index)
	}

	// 触发回调
	if c.onEvict != nil {
		c.onEvict(item.key, item.value, CapacityEvicted)
	}
}

// removeLowestFrequency 删除频率最低的项
func (c *LFUCache) removeLowestFrequency() {
	if c.freqHeap.Len() == 0 {
		return
	}
	item := heap.Pop(c.freqHeap).(*lfuItem)
	delete(c.items, item.key)

	// 触发回调
	if c.onEvict != nil {
		c.onEvict(item.key, item.value, CapacityEvicted)
	}
}

// StartDecay 启动定期频率衰减
// interval: 衰减间隔时间
// 返回一个停止函数，调用该函数可以停止衰减
func (c *LFUCache) StartDecay(interval time.Duration) func() {
	c.stopCh = make(chan struct{})
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.applyDecay()
			case <-c.stopCh:
				return
			}
		}
	}()

	return func() {
		close(c.stopCh)
	}
}

// applyDecay 对所有缓存项应用频率衰减
func (c *LFUCache) applyDecay() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, item := range c.items {
		item.frequency *= c.decayFactor
		// 确保频率不会低于 0
		if item.frequency < 0 {
			item.frequency = 0
		}
	}
	// 重新构建堆
	heap.Init(c.freqHeap)
}

// SetDecayFactor 设置衰减系数 (0.0 - 1.0)
// 默认值为 0.5，值越大衰减越慢
func (c *LFUCache) SetDecayFactor(factor float64) {
	if factor >= 0.0 && factor <= 1.0 {
		c.decayFactor = factor
	}
}

// GetFrequencies 获取所有键的频率信息（用于调试和监控）
func (c *LFUCache) GetFrequencies() map[string]float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	freqs := make(map[string]float64, len(c.items))
	for key, item := range c.items {
		freqs[key] = item.frequency
	}
	return freqs
}
