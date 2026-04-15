package cache

import (
	"container/list"
	"time"
)

// listData 列表数据结构
type listData struct {
	items      *list.List
	expiration int64 // 过期时间戳(纳秒), 0表示永不过期
}

// ListCache List 类型缓存操作
type ListCache struct {
	cache *MemoryCache
}

// NewListCache 创建 List 类型缓存
func NewListCache() *ListCache {
	return &ListCache{
		cache: New(),
	}
}

// NewListCacheWithMemory 创建带共享 MemoryCache 的 List 缓存
func NewListCacheWithMemory(mc *MemoryCache) *ListCache {
	if mc == nil {
		mc = New()
	}
	return &ListCache{
		cache: mc,
	}
}

// getOrCreateList 获取或创建列表数据
func (lc *ListCache) getOrCreateList(key string, ttl time.Duration) *listData {
	lc.cache.mu.Lock()
	defer lc.cache.mu.Unlock()

	item, found := lc.cache.items[key]
	if found && !item.IsExpired() {
		if ld, ok := item.Value.(*listData); ok {
			return ld
		}
	}

	ld := &listData{
		items: list.New(),
	}
	if ttl > 0 {
		ld.expiration = time.Now().Add(ttl).UnixNano()
	}

	lc.cache.items[key] = &Item{
		Value:      ld,
		Expiration: ld.expiration,
	}

	return ld
}

// isExpired 检查列表是否过期
func (ld *listData) isExpired() bool {
	if ld.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > ld.expiration
}

// LPush 从左侧推入一个或多个值
func (lc *ListCache) LPush(key string, ttl time.Duration, values ...any) int {
	lc.cache.mu.Lock()
	defer lc.cache.mu.Unlock()

	lc.cache.Stats.Sets.Add(1)

	ld := lc.getListDataLocked(key, ttl)

	for _, v := range values {
		ld.items.PushFront(v)
	}

	return ld.items.Len()
}

// getListDataLocked 在已持有锁的情况下获取列表数据
func (lc *ListCache) getListDataLocked(key string, ttl time.Duration) *listData {
	item, found := lc.cache.items[key]
	if found && !item.IsExpired() {
		if ld, ok := item.Value.(*listData); ok {
			return ld
		}
	}

	ld := &listData{
		items: list.New(),
	}
	if ttl > 0 {
		ld.expiration = time.Now().Add(ttl).UnixNano()
	}

	lc.cache.items[key] = &Item{
		Value:      ld,
		Expiration: ld.expiration,
	}

	return ld
}

// RPush 从右侧推入一个或多个值
func (lc *ListCache) RPush(key string, ttl time.Duration, values ...any) int {
	lc.cache.mu.Lock()
	defer lc.cache.mu.Unlock()

	lc.cache.Stats.Sets.Add(1)

	ld := lc.getListDataLocked(key, ttl)

	for _, v := range values {
		ld.items.PushBack(v)
	}

	return ld.items.Len()
}

// LPop 从左侧弹出一个值
func (lc *ListCache) LPop(key string) (any, bool) {
	lc.cache.mu.Lock()
	defer lc.cache.mu.Unlock()

	ld, found := lc.getListDataIfExist(key)
	if !found {
		lc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	elem := ld.items.Front()
	if elem == nil {
		lc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	ld.items.Remove(elem)
	lc.cache.Stats.Hits.Add(1)
	return elem.Value, true
}

// getListDataIfExist 在已持有锁的情况下获取列表数据
func (lc *ListCache) getListDataIfExist(key string) (*listData, bool) {
	item, found := lc.cache.items[key]
	if !found || item.IsExpired() {
		return nil, false
	}

	ld, ok := item.Value.(*listData)
	if !ok {
		return nil, false
	}

	if ld.isExpired() {
		delete(lc.cache.items, key)
		return nil, false
	}

	return ld, true
}

// RPop 从右侧弹出一个值
func (lc *ListCache) RPop(key string) (any, bool) {
	lc.cache.mu.Lock()
	defer lc.cache.mu.Unlock()

	ld, found := lc.getListDataIfExist(key)
	if !found {
		lc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	elem := ld.items.Back()
	if elem == nil {
		lc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	ld.items.Remove(elem)
	lc.cache.Stats.Hits.Add(1)
	return elem.Value, true
}

// LRange 获取指定范围的元素
func (lc *ListCache) LRange(key string, start, stop int) ([]any, bool) {
	lc.cache.mu.Lock()
	defer lc.cache.mu.Unlock()

	ld, found := lc.getListDataIfExist(key)
	if !found {
		lc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	lc.cache.Stats.Hits.Add(1)

	length := ld.items.Len()
	if length == 0 {
		return []any{}, true
	}

	// 处理负数索引
	if start < 0 {
		start = length + start
		if start < 0 {
			start = 0
		}
	}

	if stop < 0 {
		stop = length + stop
	}

	// 边界检查
	if start >= length {
		return []any{}, true
	}

	if stop >= length {
		stop = length - 1
	}

	if start > stop {
		return []any{}, true
	}

	// 收集范围内的元素
	result := make([]any, 0, stop-start+1)
	elem := ld.items.Front()
	for i := 0; elem != nil && i <= stop; i++ {
		if i >= start {
			result = append(result, elem.Value)
		}
		elem = elem.Next()
	}

	return result, true
}

// LIndex 获取指定索引的元素
func (lc *ListCache) LIndex(key string, index int) (any, bool) {
	lc.cache.mu.Lock()
	defer lc.cache.mu.Unlock()

	ld, found := lc.getListDataIfExist(key)
	if !found {
		lc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	lc.cache.Stats.Hits.Add(1)

	length := ld.items.Len()
	if length == 0 {
		return nil, false
	}

	// 处理负数索引
	if index < 0 {
		index = length + index
	}

	if index < 0 || index >= length {
		return nil, false
	}

	elem := ld.items.Front()
	for i := 0; elem != nil; i++ {
		if i == index {
			return elem.Value, true
		}
		elem = elem.Next()
	}

	return nil, false
}

// LLen 获取列表长度
func (lc *ListCache) LLen(key string) (int, bool) {
	lc.cache.mu.Lock()
	defer lc.cache.mu.Unlock()

	ld, found := lc.getListDataIfExist(key)
	if !found {
		lc.cache.Stats.Misses.Add(1)
		return 0, false
	}

	lc.cache.Stats.Hits.Add(1)
	return ld.items.Len(), true
}

// LTrim 修剪列表到指定范围
func (lc *ListCache) LTrim(key string, start, stop int) bool {
	lc.cache.mu.Lock()
	defer lc.cache.mu.Unlock()

	lc.cache.Stats.Sets.Add(1)

	ld, found := lc.getListDataIfExist(key)
	if !found {
		return false
	}

	length := ld.items.Len()
	if length == 0 {
		return true
	}

	// 处理负数索引
	if start < 0 {
		start = length + start
		if start < 0 {
			start = 0
		}
	}

	if stop < 0 {
		stop = length + stop
	}

	// 边界检查
	if start >= length {
		lc.cache.items[key] = &Item{
			Value:      &listData{items: list.New(), expiration: ld.expiration},
			Expiration: ld.expiration,
		}
		return true
	}

	if stop >= length {
		stop = length - 1
	}

	if start > stop {
		lc.cache.items[key] = &Item{
			Value:      &listData{items: list.New(), expiration: ld.expiration},
			Expiration: ld.expiration,
		}
		return true
	}

	// 收集保留的元素
	var keep []any
	for i, elem := 0, ld.items.Front(); elem != nil; i, elem = i+1, elem.Next() {
		if i >= start && i <= stop {
			keep = append(keep, elem.Value)
		}
	}

	// 重建列表
	newLd := &listData{
		items:      list.New(),
		expiration: ld.expiration,
	}
	for _, v := range keep {
		newLd.items.PushBack(v)
	}
	lc.cache.items[key] = &Item{
		Value:      newLd,
		Expiration: ld.expiration,
	}

	return true
}

// LRem 从列表中删除指定值的元素
func (lc *ListCache) LRem(key string, count int, value any) int {
	lc.cache.mu.Lock()
	defer lc.cache.mu.Unlock()

	lc.cache.Stats.Sets.Add(1)

	ld, found := lc.getListDataIfExist(key)
	if !found {
		return 0
	}

	removed := 0

	if count == 0 {
		// 删除所有匹配值
		var next *list.Element
		for elem := ld.items.Front(); elem != nil; elem = next {
			next = elem.Next()
			if elem.Value == value {
				ld.items.Remove(elem)
				removed++
			}
		}
	} else if count > 0 {
		// 从头开始删除
		var next *list.Element
		for elem := ld.items.Front(); elem != nil && removed < count; elem = next {
			next = elem.Next()
			if elem.Value == value {
				ld.items.Remove(elem)
				removed++
			}
		}
	} else {
		// 从尾开始删除
		count = -count
		var prev *list.Element
		for elem := ld.items.Back(); elem != nil && removed < count; elem = prev {
			prev = elem.Prev()
			if elem.Value == value {
				ld.items.Remove(elem)
				removed++
			}
		}
	}

	return removed
}

// Delete 删除整个列表
func (lc *ListCache) Delete(key string) bool {
	return lc.cache.Delete(key)
}

// Exists 检查列表是否存在
func (lc *ListCache) Exists(key string) bool {
	return lc.cache.Exists(key)
}

// Keys 返回所有未过期的键
func (lc *ListCache) Keys() []string {
	return lc.cache.Keys()
}

// Clear 清空所有列表
func (lc *ListCache) Clear() {
	lc.cache.Clear()
}

// Count 返回列表数量
func (lc *ListCache) Count() int {
	return lc.cache.Count()
}

// GetCache 获取底层 MemoryCache（用于测试和高级操作）
func (lc *ListCache) GetCache() *MemoryCache {
	return lc.cache
}
