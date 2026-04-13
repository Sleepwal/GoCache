package cache

import (
	"container/list"
	"sync"
	"time"
)

// lruItem LRU 缓存项
type lruItem struct {
	key        string
	value      any
	expiration int64 // 过期时间戳(纳秒), 0表示永不过期
}

// LRUCache LRU 缓存实现
type LRUCache struct {
	capacity int
	items    map[string]*list.Element
	lruList  *list.List // 双向链表，前端=最近使用，后端=最久未使用
	mu       sync.RWMutex
}

// NewLRU 创建一个新的 LRU 缓存
// capacity: 缓存容量，0 表示无限制
func NewLRU(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		lruList:  list.New(),
	}
}

// Set 添加或更新缓存项
func (c *LRUCache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果键已存在，更新值并移到前端
	if elem, exists := c.items[key]; exists {
		c.lruList.MoveToFront(elem)
		item := elem.Value.(*lruItem)
		item.value = value
		if ttl > 0 {
			item.expiration = time.Now().Add(ttl).UnixNano()
		}
		return
	}

	// 如果缓存已满，删除最久未使用的项
	if c.capacity > 0 && c.lruList.Len() >= c.capacity {
		c.removeOldest()
	}

	// 创建新项并添加到链表前端
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	newItem := &lruItem{
		key:        key,
		value:      value,
		expiration: expiration,
	}
	elem := c.lruList.PushFront(newItem)
	c.items[key] = elem
}

// Get 获取缓存项
func (c *LRUCache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, exists := c.items[key]
	if !exists {
		return nil, false
	}

	item := elem.Value.(*lruItem)

	// 检查是否过期
	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		c.removeElement(elem)
		return nil, false
	}

	// 移到前端（最近使用）
	c.lruList.MoveToFront(elem)
	return item.value, true
}

// Delete 删除缓存项
func (c *LRUCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, exists := c.items[key]
	if !exists {
		return false
	}

	c.removeElement(elem)
	return true
}

// Exists 检查键是否存在（包括是否过期）
func (c *LRUCache) Exists(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, exists := c.items[key]
	if !exists {
		return false
	}

	item := elem.Value.(*lruItem)
	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		c.removeElement(elem)
		return false
	}

	return true
}

// Keys 返回所有未过期的键
func (c *LRUCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.items))
	now := time.Now().UnixNano()

	for _, elem := range c.items {
		item := elem.Value.(*lruItem)
		if item.expiration == 0 || now <= item.expiration {
			keys = append(keys, item.key)
		}
	}

	return keys
}

// Clear 清空所有缓存
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.lruList = list.New()
}

// Count 返回缓存项数量（包括已过期的）
func (c *LRUCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// removeOldest 删除最久未使用的项（链表尾部）
func (c *LRUCache) removeOldest() {
	elem := c.lruList.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement 删除指定元素
func (c *LRUCache) removeElement(elem *list.Element) {
	c.lruList.Remove(elem)
	item := elem.Value.(*lruItem)
	delete(c.items, item.key)
}
