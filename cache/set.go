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
	cache map[string]*setData
	mu    sync.RWMutex
}

// NewSetCache 创建 Set 类型缓存
func NewSetCache() *SetCache {
	return &SetCache{
		cache: make(map[string]*setData),
	}
}

// isExpired 检查 Set 是否过期
func (sd *setData) isExpired() bool {
	if sd.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > sd.expiration
}

// getOrCreate 获取或创建 Set 数据
func (sc *SetCache) getOrCreate(key string, ttl time.Duration) *setData {
	sd, found := sc.cache[key]
	if !found || sd.isExpired() {
		sd = &setData{
			members: make(map[any]bool),
		}
		if ttl > 0 {
			sd.expiration = time.Now().Add(ttl).UnixNano()
		}
		sc.cache[key] = sd
	}
	return sd
}

// SAdd 添加一个或多个成员
func (sc *SetCache) SAdd(key string, ttl time.Duration, members ...any) int {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sd := sc.getOrCreate(key, ttl)

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
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sd, found := sc.cache[key]
	if !found || sd.isExpired() {
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
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	sd, found := sc.cache[key]
	if !found || sd.isExpired() {
		return false
	}

	return sd.members[member]
}

// SCard 获取集合基数（大小）
func (sc *SetCache) SCard(key string) (int, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	sd, found := sc.cache[key]
	if !found || sd.isExpired() {
		return 0, false
	}

	return len(sd.members), true
}

// SMembers 获取所有成员
func (sc *SetCache) SMembers(key string) ([]any, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	sd, found := sc.cache[key]
	if !found || sd.isExpired() {
		return nil, false
	}

	members := make([]any, 0, len(sd.members))
	for member := range sd.members {
		members = append(members, member)
	}

	return members, true
}

// SPop 随机弹出一个成员
func (sc *SetCache) SPop(key string) (any, bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sd, found := sc.cache[key]
	if !found || sd.isExpired() {
		return nil, false
	}

	if len(sd.members) == 0 {
		return nil, false
	}

	// 随机选择一个成员
	for member := range sd.members {
		delete(sd.members, member)
		return member, true
	}

	return nil, false
}

// SUnion 获取多个集合的并集
func (sc *SetCache) SUnion(keys ...string) []any {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	union := make(map[any]bool)

	for _, key := range keys {
		sd, found := sc.cache[key]
		if !found || sd.isExpired() {
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
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if len(keys) == 0 {
		return []any{}
	}

	// 获取第一个集合
	firstSd, found := sc.cache[keys[0]]
	if !found || firstSd.isExpired() {
		return []any{}
	}

	intersection := make(map[any]bool)
	for member := range firstSd.members {
		intersection[member] = true
	}

	// 与其他集合求交集
	for _, key := range keys[1:] {
		sd, found := sc.cache[key]
		if !found || sd.isExpired() {
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
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	sd1, found1 := sc.cache[key1]
	if !found1 || sd1.isExpired() {
		return []any{}
	}

	sd2, found2 := sc.cache[key2]

	diff := make([]any, 0)
	for member := range sd1.members {
		if !found2 || sd2.isExpired() || !sd2.members[member] {
			diff = append(diff, member)
		}
	}

	return diff
}

// Delete 删除整个 Set
func (sc *SetCache) Delete(key string) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	_, found := sc.cache[key]
	if !found {
		return false
	}

	delete(sc.cache, key)
	return true
}

// Exists 检查 Set 是否存在
func (sc *SetCache) Exists(key string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	sd, found := sc.cache[key]
	if !found {
		return false
	}

	return !sd.isExpired()
}

// Keys 返回所有未过期的键
func (sc *SetCache) Keys() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	keys := make([]string, 0, len(sc.cache))
	for key, sd := range sc.cache {
		if !sd.isExpired() {
			keys = append(keys, key)
		}
	}

	return keys
}

// Clear 清空所有 Set
func (sc *SetCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.cache = make(map[string]*setData)
}

// Count 返回 Set 数量
func (sc *SetCache) Count() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return len(sc.cache)
}

// seedRand 初始化随机种子
var seedRand = sync.Once{}

func init() {
	seedRand.Do(func() {
		rand.Seed(time.Now().UnixNano())
	})
}
