package cache

import (
	"math/rand"
	"sync"
	"time"
)

const (
	skipListMaxLevel = 32
	skipListP        = 0.25
)

type skipListNode struct {
	member   string
	score    float64
	backward *skipListNode
	level    []skipListLevel
}

type skipListLevel struct {
	forward *skipListNode
	span    int
}

type skipList struct {
	header *skipListNode
	tail   *skipListNode
	length int
	level  int
	rng    *rand.Rand
}

type ScoredMember struct {
	Member string
	Score  float64
}

func newSkipListNode(member string, score float64, level int) *skipListNode {
	return &skipListNode{
		member: member,
		score:  score,
		level:  make([]skipListLevel, level),
	}
}

func newSkipList() *skipList {
	sl := &skipList{
		level: 1,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	sl.header = newSkipListNode("", 0, skipListMaxLevel)
	return sl
}

func (sl *skipList) randomLevel() int {
	level := 1
	for level < skipListMaxLevel && sl.rng.Float64() < skipListP {
		level++
	}
	return level
}

func (sl *skipList) insert(member string, score float64) *skipListNode {
	update := make([]*skipListNode, skipListMaxLevel)
	rank := make([]int, skipListMaxLevel)

	node := sl.header
	for i := sl.level - 1; i >= 0; i-- {
		if i == sl.level-1 {
			rank[i] = 0
		} else {
			rank[i] = rank[i+1]
		}

		for node.level[i].forward != nil &&
			(node.level[i].forward.score < score ||
				(node.level[i].forward.score == score && node.level[i].forward.member < member)) {
			rank[i] += node.level[i].span
			node = node.level[i].forward
		}
		update[i] = node
	}

	level := sl.randomLevel()
	if level > sl.level {
		for i := sl.level; i < level; i++ {
			rank[i] = 0
			update[i] = sl.header
			update[i].level[i].span = sl.length
		}
		sl.level = level
	}

	newNode := newSkipListNode(member, score, level)
	for i := 0; i < level; i++ {
		newNode.level[i].forward = update[i].level[i].forward
		update[i].level[i].forward = newNode

		newNode.level[i].span = update[i].level[i].span - (rank[0] - rank[i])
		update[i].level[i].span = (rank[0] - rank[i]) + 1
	}

	for i := level; i < sl.level; i++ {
		update[i].level[i].span++
	}

	if update[0] != sl.header {
		newNode.backward = update[0]
	}

	if newNode.level[0].forward != nil {
		newNode.level[0].forward.backward = newNode
	} else {
		sl.tail = newNode
	}

	sl.length++
	return newNode
}

func (sl *skipList) deleteNode(node *skipListNode, update []*skipListNode) {
	for i := 0; i < sl.level; i++ {
		if update[i].level[i].forward == node {
			update[i].level[i].span += node.level[i].span - 1
			update[i].level[i].forward = node.level[i].forward
		} else {
			update[i].level[i].span--
		}
	}

	if node.level[0].forward != nil {
		node.level[0].forward.backward = node.backward
	} else {
		sl.tail = node.backward
	}

	for sl.level > 1 && sl.header.level[sl.level-1].forward == nil {
		sl.level--
	}

	sl.length--
}

func (sl *skipList) remove(member string, score float64) bool {
	update := make([]*skipListNode, skipListMaxLevel)

	node := sl.header
	for i := sl.level - 1; i >= 0; i-- {
		for node.level[i].forward != nil &&
			(node.level[i].forward.score < score ||
				(node.level[i].forward.score == score && node.level[i].forward.member < member)) {
			node = node.level[i].forward
		}
		update[i] = node
	}

	node = node.level[0].forward
	if node != nil && node.score == score && node.member == member {
		sl.deleteNode(node, update)
		return true
	}

	return false
}

func (sl *skipList) getRank(member string, score float64) int {
	rank := 0
	node := sl.header

	for i := sl.level - 1; i >= 0; i-- {
		for node.level[i].forward != nil &&
			(node.level[i].forward.score < score ||
				(node.level[i].forward.score == score && node.level[i].forward.member <= member)) {
			rank += node.level[i].span
			node = node.level[i].forward
		}

		if node.member == member {
			return rank
		}
	}

	return 0
}

func (sl *skipList) getByRank(rank int) *skipListNode {
	traversed := 0
	node := sl.header

	for i := sl.level - 1; i >= 0; i-- {
		for node.level[i].forward != nil && traversed+node.level[i].span <= rank {
			traversed += node.level[i].span
			node = node.level[i].forward
		}

		if traversed == rank {
			return node
		}
	}

	return nil
}

func (sl *skipList) getRangeByRank(start, end int) []ScoredMember {
	var result []ScoredMember

	startNode := sl.getByRank(start)
	if startNode == nil {
		return result
	}

	traversed := start
	node := startNode

	for node != nil && traversed <= end {
		result = append(result, ScoredMember{Member: node.member, Score: node.score})
		node = node.level[0].forward
		traversed++
	}

	return result
}

func (sl *skipList) getRangeByScore(min, max float64, offset, count int) []ScoredMember {
	var result []ScoredMember

	node := sl.header
	for i := sl.level - 1; i >= 0; i-- {
		for node.level[i].forward != nil && node.level[i].forward.score < min {
			node = node.level[i].forward
		}
	}

	node = node.level[0].forward

	skipped := 0
	for node != nil && node.score <= max {
		if skipped >= offset && (count < 0 || len(result) < count) {
			result = append(result, ScoredMember{Member: node.member, Score: node.score})
		}
		if skipped >= offset {
		}
		skipped++
		node = node.level[0].forward
	}

	return result
}

func (sl *skipList) getRangeByScoreReverse(max, min float64, offset, count int) []ScoredMember {
	var result []ScoredMember

	node := sl.tail
	if node == nil {
		return result
	}

	for node != nil && node.score > max {
		node = node.backward
	}

	skipped := 0
	for node != nil && node.score >= min {
		if skipped >= offset && (count < 0 || len(result) < count) {
			result = append(result, ScoredMember{Member: node.member, Score: node.score})
		}
		skipped++
		node = node.backward
	}

	return result
}

func (sl *skipList) popMin() *ScoredMember {
	if sl.length == 0 {
		return nil
	}

	node := sl.header.level[0].forward
	if node == nil {
		return nil
	}

	sl.remove(node.member, node.score)
	return &ScoredMember{Member: node.member, Score: node.score}
}

func (sl *skipList) popMax() *ScoredMember {
	if sl.length == 0 {
		return nil
	}

	node := sl.tail
	if node == nil {
		return nil
	}

	sl.remove(node.member, node.score)
	return &ScoredMember{Member: node.member, Score: node.score}
}

func (sl *skipList) countInRange(min, max float64) int {
	count := 0

	node := sl.header
	for i := sl.level - 1; i >= 0; i-- {
		for node.level[i].forward != nil && node.level[i].forward.score < min {
			node = node.level[i].forward
		}
	}

	node = node.level[0].forward
	for node != nil && node.score <= max {
		count++
		node = node.level[0].forward
	}

	return count
}

type sortedSetData struct {
	skiplist   *skipList
	dict       map[string]float64
	expiration int64
	mu         sync.RWMutex
}

func newSortedSetData() *sortedSetData {
	return &sortedSetData{
		skiplist: newSkipList(),
		dict:     make(map[string]float64),
	}
}

func (ssd *sortedSetData) isExpired() bool {
	if ssd.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > ssd.expiration
}

func (ssd *sortedSetData) zadd(members map[string]float64) int {
	added := 0
	for member, score := range members {
		if oldScore, exists := ssd.dict[member]; exists {
			if oldScore != score {
				ssd.skiplist.remove(member, oldScore)
				ssd.skiplist.insert(member, score)
				ssd.dict[member] = score
			}
		} else {
			ssd.skiplist.insert(member, score)
			ssd.dict[member] = score
			added++
		}
	}
	return added
}

func (ssd *sortedSetData) zrem(members ...string) int {
	removed := 0
	for _, member := range members {
		if score, exists := ssd.dict[member]; exists {
			ssd.skiplist.remove(member, score)
			delete(ssd.dict, member)
			removed++
		}
	}
	return removed
}

func (ssd *sortedSetData) zscore(member string) (float64, bool) {
	score, exists := ssd.dict[member]
	return score, exists
}

func (ssd *sortedSetData) zincrby(member string, increment float64) float64 {
	score, exists := ssd.dict[member]
	if exists {
		ssd.skiplist.remove(member, score)
		score += increment
	} else {
		score = increment
	}
	ssd.skiplist.insert(member, score)
	ssd.dict[member] = score
	return score
}

func (ssd *sortedSetData) zcard() int {
	return len(ssd.dict)
}

func (ssd *sortedSetData) zrank(member string) (int, bool) {
	score, exists := ssd.dict[member]
	if !exists {
		return 0, false
	}
	rank := ssd.skiplist.getRank(member, score)
	if rank == 0 {
		return 0, false
	}
	return rank - 1, true
}

func (ssd *sortedSetData) zrevrank(member string) (int, bool) {
	rank, exists := ssd.zrank(member)
	if !exists {
		return 0, false
	}
	return len(ssd.dict) - rank - 1, true
}

func (ssd *sortedSetData) zrange(start, end int) []ScoredMember {
	length := len(ssd.dict)
	if length == 0 {
		return nil
	}

	if start < 0 {
		start = length + start
		if start < 0 {
			start = 0
		}
	}
	if end < 0 {
		end = length + end
	}
	if start > end || start >= length {
		return nil
	}
	if end >= length {
		end = length - 1
	}

	return ssd.skiplist.getRangeByRank(start+1, end+1)
}

func (ssd *sortedSetData) zrevrange(start, end int) []ScoredMember {
	length := len(ssd.dict)
	if length == 0 {
		return nil
	}

	if start < 0 {
		start = length + start
		if start < 0 {
			start = 0
		}
	}
	if end < 0 {
		end = length + end
	}
	if start > end || start >= length {
		return nil
	}
	if end >= length {
		end = length - 1
	}

	newStart := length - 1 - end
	newEnd := length - 1 - start

	result := ssd.skiplist.getRangeByRank(newStart+1, newEnd+1)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

func (ssd *sortedSetData) zrangebyscore(min, max float64, offset, count int) []ScoredMember {
	return ssd.skiplist.getRangeByScore(min, max, offset, count)
}

func (ssd *sortedSetData) zrevrangebyscore(max, min float64, offset, count int) []ScoredMember {
	return ssd.skiplist.getRangeByScoreReverse(max, min, offset, count)
}

func (ssd *sortedSetData) zcount(min, max float64) int {
	return ssd.skiplist.countInRange(min, max)
}

func (ssd *sortedSetData) zpopmin() *ScoredMember {
	sm := ssd.skiplist.popMin()
	if sm != nil {
		delete(ssd.dict, sm.Member)
	}
	return sm
}

func (ssd *sortedSetData) zpopmax() *ScoredMember {
	sm := ssd.skiplist.popMax()
	if sm != nil {
		delete(ssd.dict, sm.Member)
	}
	return sm
}

func (ssd *sortedSetData) zremrangebyrank(start, end int) int {
	members := ssd.zrange(start, end)
	if members == nil {
		return 0
	}
	for _, m := range members {
		ssd.skiplist.remove(m.Member, m.Score)
		delete(ssd.dict, m.Member)
	}
	return len(members)
}

func (ssd *sortedSetData) zremrangebyscore(min, max float64) int {
	members := ssd.skiplist.getRangeByScore(min, max, 0, -1)
	if members == nil {
		return 0
	}
	for _, m := range members {
		ssd.skiplist.remove(m.Member, m.Score)
		delete(ssd.dict, m.Member)
	}
	return len(members)
}

type SortedSetCache struct {
	cache *MemoryCache
}

func NewSortedSetCache() *SortedSetCache {
	return &SortedSetCache{
		cache: New(),
	}
}

func NewSortedSetCacheWithMemory(mc *MemoryCache) *SortedSetCache {
	if mc == nil {
		mc = New()
	}
	return &SortedSetCache{
		cache: mc,
	}
}

func (ssc *SortedSetCache) getSortedSetDataIfExist(key string) (*sortedSetData, bool) {
	item, found := ssc.cache.items[key]
	if !found || item.IsExpired() {
		return nil, false
	}

	ssd, ok := item.Value.(*sortedSetData)
	if !ok {
		return nil, false
	}

	if ssd.isExpired() {
		delete(ssc.cache.items, key)
		return nil, false
	}

	return ssd, true
}

func (ssc *SortedSetCache) getOrCreateSortedSet(key string, ttl time.Duration) *sortedSetData {
	item, found := ssc.cache.items[key]
	if found && !item.IsExpired() {
		if ssd, ok := item.Value.(*sortedSetData); ok {
			return ssd
		}
	}

	ssd := newSortedSetData()
	if ttl > 0 {
		ssd.expiration = time.Now().Add(ttl).UnixNano()
	}

	ssc.cache.items[key] = &Item{
		Value:      ssd,
		Expiration: ssd.expiration,
	}

	return ssd
}

func (ssc *SortedSetCache) ZAdd(key string, ttl time.Duration, members map[string]float64) int {
	ssc.cache.mu.Lock()
	defer ssc.cache.mu.Unlock()

	ssc.cache.Stats.Sets.Add(1)

	ssd := ssc.getOrCreateSortedSet(key, ttl)
	return ssd.zadd(members)
}

func (ssc *SortedSetCache) ZRem(key string, members ...string) int {
	ssc.cache.mu.Lock()
	defer ssc.cache.mu.Unlock()

	ssc.cache.Stats.Deletes.Add(1)

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		return 0
	}

	return ssd.zrem(members...)
}

func (ssc *SortedSetCache) ZScore(key, member string) (float64, bool) {
	ssc.cache.mu.RLock()
	defer ssc.cache.mu.RUnlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		ssc.cache.Stats.Misses.Add(1)
		return 0, false
	}

	score, exists := ssd.zscore(member)
	if !exists {
		ssc.cache.Stats.Misses.Add(1)
		return 0, false
	}

	ssc.cache.Stats.Hits.Add(1)
	return score, true
}

func (ssc *SortedSetCache) ZIncrBy(key, member string, increment float64) (float64, bool) {
	ssc.cache.mu.Lock()
	defer ssc.cache.mu.Unlock()

	ssc.cache.Stats.Sets.Add(1)

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		ssd = ssc.getOrCreateSortedSet(key, 0)
	}

	return ssd.zincrby(member, increment), true
}

func (ssc *SortedSetCache) ZCard(key string) (int, bool) {
	ssc.cache.mu.RLock()
	defer ssc.cache.mu.RUnlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		ssc.cache.Stats.Misses.Add(1)
		return 0, false
	}

	ssc.cache.Stats.Hits.Add(1)
	return ssd.zcard(), true
}

func (ssc *SortedSetCache) ZRank(key, member string) (int, bool) {
	ssc.cache.mu.RLock()
	defer ssc.cache.mu.RUnlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		ssc.cache.Stats.Misses.Add(1)
		return 0, false
	}

	rank, exists := ssd.zrank(member)
	if !exists {
		ssc.cache.Stats.Misses.Add(1)
		return 0, false
	}

	ssc.cache.Stats.Hits.Add(1)
	return rank, true
}

func (ssc *SortedSetCache) ZRevRank(key, member string) (int, bool) {
	ssc.cache.mu.RLock()
	defer ssc.cache.mu.RUnlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		ssc.cache.Stats.Misses.Add(1)
		return 0, false
	}

	rank, exists := ssd.zrevrank(member)
	if !exists {
		ssc.cache.Stats.Misses.Add(1)
		return 0, false
	}

	ssc.cache.Stats.Hits.Add(1)
	return rank, true
}

func (ssc *SortedSetCache) ZRange(key string, start, end int) ([]ScoredMember, bool) {
	ssc.cache.mu.RLock()
	defer ssc.cache.mu.RUnlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		ssc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	ssc.cache.Stats.Hits.Add(1)
	return ssd.zrange(start, end), true
}

func (ssc *SortedSetCache) ZRevRange(key string, start, end int) ([]ScoredMember, bool) {
	ssc.cache.mu.RLock()
	defer ssc.cache.mu.RUnlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		ssc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	ssc.cache.Stats.Hits.Add(1)
	return ssd.zrevrange(start, end), true
}

func (ssc *SortedSetCache) ZRangeByScore(key string, min, max float64, offset, count int) ([]ScoredMember, bool) {
	ssc.cache.mu.RLock()
	defer ssc.cache.mu.RUnlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		ssc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	ssc.cache.Stats.Hits.Add(1)
	return ssd.zrangebyscore(min, max, offset, count), true
}

func (ssc *SortedSetCache) ZRevRangeByScore(key string, max, min float64, offset, count int) ([]ScoredMember, bool) {
	ssc.cache.mu.RLock()
	defer ssc.cache.mu.RUnlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		ssc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	ssc.cache.Stats.Hits.Add(1)
	return ssd.zrevrangebyscore(max, min, offset, count), true
}

func (ssc *SortedSetCache) ZCount(key string, min, max float64) (int, bool) {
	ssc.cache.mu.RLock()
	defer ssc.cache.mu.RUnlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		ssc.cache.Stats.Misses.Add(1)
		return 0, false
	}

	ssc.cache.Stats.Hits.Add(1)
	return ssd.zcount(min, max), true
}

func (ssc *SortedSetCache) ZPopMin(key string) (*ScoredMember, bool) {
	ssc.cache.mu.Lock()
	defer ssc.cache.mu.Unlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		ssc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	result := ssd.zpopmin()
	if result == nil {
		ssc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	ssc.cache.Stats.Hits.Add(1)
	return result, true
}

func (ssc *SortedSetCache) ZPopMax(key string) (*ScoredMember, bool) {
	ssc.cache.mu.Lock()
	defer ssc.cache.mu.Unlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		ssc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	result := ssd.zpopmax()
	if result == nil {
		ssc.cache.Stats.Misses.Add(1)
		return nil, false
	}

	ssc.cache.Stats.Hits.Add(1)
	return result, true
}

func (ssc *SortedSetCache) ZRemRangeByRank(key string, start, end int) (int, bool) {
	ssc.cache.mu.Lock()
	defer ssc.cache.mu.Unlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		return 0, false
	}

	return ssd.zremrangebyrank(start, end), true
}

func (ssc *SortedSetCache) ZRemRangeByScore(key string, min, max float64) (int, bool) {
	ssc.cache.mu.Lock()
	defer ssc.cache.mu.Unlock()

	ssd, found := ssc.getSortedSetDataIfExist(key)
	if !found {
		return 0, false
	}

	return ssd.zremrangebyscore(min, max), true
}

func (ssc *SortedSetCache) ZUnionStore(dest string, ttl time.Duration, keys []string, weights []float64, aggregate string) (int, bool) {
	ssc.cache.mu.Lock()
	defer ssc.cache.mu.Unlock()

	ssc.cache.Stats.Sets.Add(1)

	merged := make(map[string]float64)

	for i, k := range keys {
		ssd, found := ssc.getSortedSetDataIfExist(k)
		if !found {
			continue
		}

		weight := 1.0
		if i < len(weights) {
			weight = weights[i]
		}

		for member, score := range ssd.dict {
			weightedScore := score * weight
			if existing, exists := merged[member]; exists {
				switch aggregate {
				case "MIN":
					if weightedScore < existing {
						merged[member] = weightedScore
					}
				case "MAX":
					if weightedScore > existing {
						merged[member] = weightedScore
					}
				default:
					merged[member] = existing + weightedScore
				}
			} else {
				merged[member] = weightedScore
			}
		}
	}

	destSsd := ssc.getOrCreateSortedSet(dest, ttl)

	for member := range destSsd.dict {
		if _, exists := merged[member]; !exists {
			destSsd.skiplist.remove(member, destSsd.dict[member])
			delete(destSsd.dict, member)
		}
	}

	destSsd.dict = make(map[string]float64)
	for member, score := range merged {
		destSsd.skiplist.insert(member, score)
		destSsd.dict[member] = score
	}

	return len(merged), true
}

func (ssc *SortedSetCache) ZInterStore(dest string, ttl time.Duration, keys []string, weights []float64, aggregate string) (int, bool) {
	ssc.cache.mu.Lock()
	defer ssc.cache.mu.Unlock()

	ssc.cache.Stats.Sets.Add(1)

	if len(keys) == 0 {
		return 0, true
	}

	firstSsd, found := ssc.getSortedSetDataIfExist(keys[0])
	if !found {
		return 0, true
	}

	intersection := make(map[string]float64)
	for member, score := range firstSsd.dict {
		weight := 1.0
		if 0 < len(weights) {
			weight = weights[0]
		}
		intersection[member] = score * weight
	}

	for i, k := range keys[1:] {
		ssd, found := ssc.getSortedSetDataIfExist(k)
		if !found {
			return 0, true
		}

		weight := 1.0
		if i+1 < len(weights) {
			weight = weights[i+1]
		}

		for member := range intersection {
			if score, exists := ssd.dict[member]; exists {
				weightedScore := score * weight
				switch aggregate {
				case "MIN":
					if weightedScore < intersection[member] {
						intersection[member] = weightedScore
					}
				case "MAX":
					if weightedScore > intersection[member] {
						intersection[member] = weightedScore
					}
				default:
					intersection[member] = intersection[member] + weightedScore
				}
			} else {
				delete(intersection, member)
			}
		}
	}

	destSsd := ssc.getOrCreateSortedSet(dest, ttl)

	for member := range destSsd.dict {
		if _, exists := intersection[member]; !exists {
			destSsd.skiplist.remove(member, destSsd.dict[member])
			delete(destSsd.dict, member)
		}
	}

	destSsd.dict = make(map[string]float64)
	for member, score := range intersection {
		destSsd.skiplist.insert(member, score)
		destSsd.dict[member] = score
	}

	return len(intersection), true
}

func (ssc *SortedSetCache) Delete(key string) bool {
	return ssc.cache.Delete(key)
}

func (ssc *SortedSetCache) Exists(key string) bool {
	return ssc.cache.Exists(key)
}

func (ssc *SortedSetCache) Keys() []string {
	return ssc.cache.Keys()
}

func (ssc *SortedSetCache) Clear() {
	ssc.cache.Clear()
}

func (ssc *SortedSetCache) Count() int {
	return ssc.cache.Count()
}

func (ssc *SortedSetCache) GetCache() *MemoryCache {
	return ssc.cache
}
