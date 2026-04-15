package cache

import (
	"math/rand"
	"sync"
	"time"
)

// setData Set 数据结构
type setData struct {
	members    map[any]bool
	expiration int64 // 过期时间戳(纳秒), 0表示永不过期
}

// SetCache Set 类型缓存操作
type SetCache struct {
	cache *MemoryCache
}

// NewSetCache 创建 Set 类型缓存
func NewSetCache() *SetCache {
	return &SetCache{
		cache: New(),
	}
}

// NewSetCacheWithMemory 创建带共享 MemoryCache 的 Set 缓存
func NewSetCacheWithMemory(mc *MemoryCache) *SetCache {
	if mc == nil {
		mc = New()
	}
	return &SetCache{
		cache: mc,
	}
}

// isExpired 检查 Set 是否过期
func (sd *setData) isExpired() bool {
	if sd.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > sd.expiration
}

// getSetDataIfExist 在已持有锁的情况下获取 Set 数据
func (sc *SetCache) getSetDataIfExist(key string) (*setData, bool) {
	item, found := sc.cache.items[key]
	if !found || item.IsExpired() {
		return nil, false
	}

	sd, ok := item.Value.(*setData)
	if !ok {
		return nil, false
	}

	if sd.isExpired() {
		delete(sc.cache.items, key)
		return nil, false
	}

	return sd, true
}

// SAdd 添加一个或多个成员
func (sc *SetCache) SAdd(key string, ttl time.Duration, members ...any) int {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	sc.cache.Stats.Sets.Add(1)

	item, found := sc.cache.items[key]
	var sd *setData

	if found && !item.IsExpired() {
		if s, ok := item.Value.(*setData); ok {
			sd = s
		}
	}

	if sd == nil {
		sd = &setData{members: make(map[any]bool)}
		if ttl > 0 {
			sd.expiration = time.Now().Add(ttl).UnixNano()
		}
		sc.cache.items[key] = &Item{
			Value:      sd,
			Expiration: sd.expiration,
		}
	}

	added := 0
	for _, member := range members {
		if !sd.members[member] {
			sd.members[member] = true
			added++
		}
	}

	return added
}

// SRem 移除一个或多个成员
func (sc *SetCache) SRem(key string, members ...any) int {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	sc.cache.Stats.Sets.Add(1)

	sd, found := sc.getSetDataIfExist(key)
	if !found {
		return 0
	}

	removed := 0
	for _, member := range members {
		if sd.members[member] {
			delete(sd.members, member)
			removed++
		}
	}

	return removed
}

// SIsMember 检查成员是否存在
func (sc *SetCache) SIsMember(key string, member any) bool {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	sd, found := sc.getSetDataIfExist(key)
	if !found {
		return false
	}

	return sd.members[member]
}

// SCard 获取集合基数（大小）
func (sc *SetCache) SCard(key string) (int, bool) {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	sd, found := sc.getSetDataIfExist(key)
	if !found {
		sc.cache.Stats.Misses.Add(1)
		return 0, false
	}

	sc.cache.Stats.Hits.Add(1)
	return len(sd.members), true
}

// SMembers 获取所有成员
func (sc *SetCache) SMembers(key string) ([]any, bool) {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	sd, found := sc.getSetDataIfExist(key)
	if !found {
		sc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	sc.cache.Stats.Hits.Add(1)

	members := make([]any, 0, len(sd.members))
	for member := range sd.members {
		members = append(members, member)
	}

	return members, true
}

// SPop 随机弹出一个成员
func (sc *SetCache) SPop(key string) (any, bool) {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	sd, found := sc.getSetDataIfExist(key)
	if !found {
		sc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	if len(sd.members) == 0 {
		sc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	sc.cache.Stats.Hits.Add(1)

	// 随机选择一个成员
	for member := range sd.members {
		delete(sd.members, member)
		return member, true
	}

	return nil, false
}

// SUnion 获取多个集合的并集
func (sc *SetCache) SUnion(keys ...string) []any {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	union := make(map[any]bool)

	for _, key := range keys {
		sd, found := sc.getSetDataIfExist(key)
		if !found {
			continue
		}

		for member := range sd.members {
			union[member] = true
		}
	}

	result := make([]any, 0, len(union))
	for member := range union {
		result = append(result, member)
	}

	return result
}

// SInter 获取多个集合的交集
func (sc *SetCache) SInter(keys ...string) []any {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	if len(keys) == 0 {
		return []any{}
	}

	// 获取第一个集合
	firstSd, found := sc.getSetDataIfExist(keys[0])
	if !found {
		return []any{}
	}

	intersection := make(map[any]bool)
	for member := range firstSd.members {
		intersection[member] = true
	}

	// 与其他集合求交集
	for _, key := range keys[1:] {
		sd, found := sc.getSetDataIfExist(key)
		if !found {
			return []any{}
		}

		for member := range intersection {
			if !sd.members[member] {
				delete(intersection, member)
			}
		}

		if len(intersection) == 0 {
			break
		}
	}

	result := make([]any, 0, len(intersection))
	for member := range intersection {
		result = append(result, member)
	}

	return result
}

// SDiff 获取两个集合的差集（key1 - key2）
func (sc *SetCache) SDiff(key1, key2 string) []any {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	sd1, found1 := sc.getSetDataIfExist(key1)
	if !found1 {
		return []any{}
	}

	sd2, found2 := sc.getSetDataIfExist(key2)

	diff := make([]any, 0)
	for member := range sd1.members {
		if !found2 || !sd2.members[member] {
			diff = append(diff, member)
		}
	}

	return diff
}

// Delete 删除整个 Set
func (sc *SetCache) Delete(key string) bool {
	return sc.cache.Delete(key)
}

// Exists 检查 Set 是否存在
func (sc *SetCache) Exists(key string) bool {
	return sc.cache.Exists(key)
}

// Keys 返回所有未过期的键
func (sc *SetCache) Keys() []string {
	return sc.cache.Keys()
}

// Clear 清空所有 Set
func (sc *SetCache) Clear() {
	sc.cache.Clear()
}

// Count 返回 Set 数量
func (sc *SetCache) Count() int {
	return sc.cache.Count()
}

// GetCache 获取底层 MemoryCache（用于测试和高级操作）
func (sc *SetCache) GetCache() *MemoryCache {
	return sc.cache
}

// seedRand 初始化随机种子
var seedRand = sync.Once{}

func init() {
	seedRand.Do(func() {
		// Go 1.20+ auto-seeds, so this is a no-op but kept for compatibility
		_ = rand.Intn(1)
	})
}
