package cache

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// NamespaceCache 缓存命名空间包装器
// 提供隔离的缓存分区
type NamespaceCache struct {
	cache   *MemoryCache
	mu      sync.RWMutex
	prefix  string
}

// NewNamespaceCache 创建缓存命名空间
func NewNamespaceCache(cache *MemoryCache, namespace string) *NamespaceCache {
	return &NamespaceCache{
		cache:  cache,
		prefix: namespace + ":",
	}
}

// qualifyKey 为键添加命名空间前缀
func (nc *NamespaceCache) qualifyKey(key string) string {
	return nc.prefix + key
}

// Set 添加或更新缓存项
func (nc *NamespaceCache) Set(key string, value any, ttl time.Duration) {
	nc.cache.Set(nc.qualifyKey(key), value, ttl)
}

// Get 获取缓存项
func (nc *NamespaceCache) Get(key string) (any, bool) {
	return nc.cache.Get(nc.qualifyKey(key))
}

// Delete 删除缓存项
func (nc *NamespaceCache) Delete(key string) bool {
	return nc.cache.Delete(nc.qualifyKey(key))
}

// Exists 检查键是否存在
func (nc *NamespaceCache) Exists(key string) bool {
	return nc.cache.Exists(nc.qualifyKey(key))
}

// Keys 返回命名空间下的所有键
func (nc *NamespaceCache) Keys() []string {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	allKeys := nc.cache.Keys()
	nsKeys := make([]string, 0)

	for _, key := range allKeys {
		if strings.HasPrefix(key, nc.prefix) {
			nsKeys = append(nsKeys, strings.TrimPrefix(key, nc.prefix))
		}
	}

	return nsKeys
}

// Clear 清空命名空间下的所有缓存
func (nc *NamespaceCache) Clear() int {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	allKeys := nc.cache.Keys()
	deleted := 0

	for _, key := range allKeys {
		if strings.HasPrefix(key, nc.prefix) {
			nc.cache.Delete(key)
			deleted++
		}
	}

	return deleted
}

// Count 返回命名空间下的缓存项数量
func (nc *NamespaceCache) Count() int {
	return len(nc.Keys())
}

// NamespacedKey 获取带命名空间的完整键
func (nc *NamespaceCache) NamespacedKey(key string) string {
	return nc.qualifyKey(key)
}

// MultiNamespaceCache 多命名空间缓存
type MultiNamespaceCache struct {
	cache      *MemoryCache
	namespaces map[string]*NamespaceCache
	mu         sync.RWMutex
}

// NewMultiNamespaceCache 创建多命名空间缓存
func NewMultiNamespaceCache(cache *MemoryCache) *MultiNamespaceCache {
	return &MultiNamespaceCache{
		cache:      cache,
		namespaces: make(map[string]*NamespaceCache),
	}
}

// Namespace 获取或创建命名空间
func (mnc *MultiNamespaceCache) Namespace(name string) *NamespaceCache {
	mnc.mu.Lock()
	defer mnc.mu.Unlock()

	if ns, exists := mnc.namespaces[name]; exists {
		return ns
	}

	ns := NewNamespaceCache(mnc.cache, name)
	mnc.namespaces[name] = ns
	return ns
}

// ListNamespaces 列出所有命名空间
func (mnc *MultiNamespaceCache) ListNamespaces() []string {
	mnc.mu.RLock()
	defer mnc.mu.RUnlock()

	names := make([]string, 0, len(mnc.namespaces))
	for name := range mnc.namespaces {
		names = append(names, name)
	}
	return names
}

// FormatKey 格式化带命名空间的键
func FormatKey(namespace, key string) string {
	return fmt.Sprintf("%s:%s", namespace, key)
}

// ParseKey 解析带命名空间的键
func ParseKey(fullKey string) (namespace, key string, ok bool) {
	parts := strings.SplitN(fullKey, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}
