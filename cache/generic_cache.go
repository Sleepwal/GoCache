package cache

import (
	"time"
)

// TypedCache 泛型缓存包装器
// 提供类型安全的 Get/Set 操作，避免运行时类型断言
type TypedCache[T any] struct {
	cache *MemoryCache
}

// NewTypedCache 创建一个新的泛型缓存
// 注意：这需要传入一个已有的 MemoryCache 实例
func NewTypedCache[T any](cache *MemoryCache) *TypedCache[T] {
	return &TypedCache[T]{
		cache: cache,
	}
}

// Set 添加或更新缓存项
func (c *TypedCache[T]) Set(key string, value T, ttl time.Duration) {
	c.cache.Set(key, value, ttl)
}

// Get 获取缓存项
// 返回值是指定类型 T，如果未找到则返回零值
func (c *TypedCache[T]) Get(key string) (T, bool) {
	val, found := c.cache.Get(key)
	if !found {
		var zero T
		return zero, false
	}

	typedVal, ok := val.(T)
	if !ok {
		var zero T
		return zero, false
	}

	return typedVal, true
}

// Delete 删除缓存项
func (c *TypedCache[T]) Delete(key string) bool {
	return c.cache.Delete(key)
}

// Exists 检查键是否存在（包括是否过期）
func (c *TypedCache[T]) Exists(key string) bool {
	return c.cache.Exists(key)
}

// Keys 返回所有未过期的键
func (c *TypedCache[T]) Keys() []string {
	return c.cache.Keys()
}

// Clear 清空所有缓存
func (c *TypedCache[T]) Clear() {
	c.cache.Clear()
}

// Count 返回缓存项数量（包括已过期的）
func (c *TypedCache[T]) Count() int {
	return c.cache.Count()
}

// Stats 获取统计信息
func (c *TypedCache[T]) Stats() *Stats {
	return c.cache.Stats
}

// TypedLRUCache 泛型 LRU 缓存包装器
type TypedLRUCache[T any] struct {
	cache *LRUCache
}

// NewTypedLRUCache 创建一个新的泛型 LRU 缓存
func NewTypedLRUCache[T any](capacity int, opts ...LRUCacheOption) *TypedLRUCache[T] {
	lruCache := NewLRU(capacity)
	for _, opt := range opts {
		opt(lruCache)
	}
	return &TypedLRUCache[T]{
		cache: lruCache,
	}
}

// Set 添加或更新缓存项
func (c *TypedLRUCache[T]) Set(key string, value T, ttl time.Duration) {
	c.cache.Set(key, value, ttl)
}

// Get 获取缓存项
func (c *TypedLRUCache[T]) Get(key string) (T, bool) {
	val, found := c.cache.Get(key)
	if !found {
		var zero T
		return zero, false
	}

	typedVal, ok := val.(T)
	if !ok {
		var zero T
		return zero, false
	}

	return typedVal, true
}

// Delete 删除缓存项
func (c *TypedLRUCache[T]) Delete(key string) bool {
	return c.cache.Delete(key)
}

// Exists 检查键是否存在（包括是否过期）
func (c *TypedLRUCache[T]) Exists(key string) bool {
	return c.cache.Exists(key)
}

// Keys 返回所有未过期的键
func (c *TypedLRUCache[T]) Keys() []string {
	return c.cache.Keys()
}

// Clear 清空所有缓存
func (c *TypedLRUCache[T]) Clear() {
	c.cache.Clear()
}

// Count 返回缓存项数量（包括已过期的）
func (c *TypedLRUCache[T]) Count() int {
	return c.cache.Count()
}

// Stats 获取统计信息
func (c *TypedLRUCache[T]) Stats() *Stats {
	return c.cache.Stats
}

// GetInternalCache 获取内部的 LRU 缓存实例（用于高级操作）
func (c *TypedLRUCache[T]) GetInternalCache() *LRUCache {
	return c.cache
}

// TypedLFUCache 泛型 LFU 缓存包装器
type TypedLFUCache[T any] struct {
	cache *LFUCache
}

// NewTypedLFUCache 创建一个新的泛型 LFU 缓存
func NewTypedLFUCache[T any](capacity int, opts ...LFUCacheOption) *TypedLFUCache[T] {
	lfuCache := NewLFU(capacity)
	for _, opt := range opts {
		opt(lfuCache)
	}
	return &TypedLFUCache[T]{
		cache: lfuCache,
	}
}

// Set 添加或更新缓存项
func (c *TypedLFUCache[T]) Set(key string, value T, ttl time.Duration) {
	c.cache.Set(key, value, ttl)
}

// Get 获取缓存项
func (c *TypedLFUCache[T]) Get(key string) (T, bool) {
	val, found := c.cache.Get(key)
	if !found {
		var zero T
		return zero, false
	}

	typedVal, ok := val.(T)
	if !ok {
		var zero T
		return zero, false
	}

	return typedVal, true
}

// Delete 删除缓存项
func (c *TypedLFUCache[T]) Delete(key string) bool {
	return c.cache.Delete(key)
}

// Exists 检查键是否存在（包括是否过期）
func (c *TypedLFUCache[T]) Exists(key string) bool {
	return c.cache.Exists(key)
}

// Keys 返回所有未过期的键
func (c *TypedLFUCache[T]) Keys() []string {
	return c.cache.Keys()
}

// Clear 清空所有缓存
func (c *TypedLFUCache[T]) Clear() {
	c.cache.Clear()
}

// Count 返回缓存项数量（包括已过期的）
func (c *TypedLFUCache[T]) Count() int {
	return c.cache.Count()
}

// Stats 获取统计信息
func (c *TypedLFUCache[T]) Stats() *Stats {
	return c.cache.Stats
}

// GetInternalCache 获取内部的 LFU 缓存实例（用于高级操作）
func (c *TypedLFUCache[T]) GetInternalCache() *LFUCache {
	return c.cache
}
