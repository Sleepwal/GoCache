package cache

import (
	"math"
	"testing"
)

func TestSkipListInsert(t *testing.T) {
	sl := newSkipList()

	sl.insert("a", 1.0)
	sl.insert("b", 2.0)
	sl.insert("c", 3.0)

	if sl.length != 3 {
		t.Errorf("expected length 3, got %d", sl.length)
	}

	node := sl.header.level[0].forward
	if node == nil || node.member != "a" || node.score != 1.0 {
		t.Errorf("first node should be (a, 1.0), got (%s, %f)", node.member, node.score)
	}

	node = sl.tail
	if node == nil || node.member != "c" || node.score != 3.0 {
		t.Errorf("tail should be (c, 3.0), got (%s, %f)", node.member, node.score)
	}
}

func TestSkipListInsertSameScore(t *testing.T) {
	sl := newSkipList()

	sl.insert("c", 1.0)
	sl.insert("a", 1.0)
	sl.insert("b", 1.0)

	node := sl.header.level[0].forward
	expected := []string{"a", "b", "c"}
	for i, exp := range expected {
		if node == nil {
			t.Errorf("node %d is nil, expected %s", i, exp)
			break
		}
		if node.member != exp {
			t.Errorf("position %d: expected %s, got %s", i, exp, node.member)
		}
		node = node.level[0].forward
	}
}

func TestSkipListRemove(t *testing.T) {
	sl := newSkipList()

	sl.insert("a", 1.0)
	sl.insert("b", 2.0)
	sl.insert("c", 3.0)

	removed := sl.remove("b", 2.0)
	if !removed {
		t.Error("expected remove to return true")
	}
	if sl.length != 2 {
		t.Errorf("expected length 2, got %d", sl.length)
	}

	removed = sl.remove("nonexistent", 99.0)
	if removed {
		t.Error("expected remove of nonexistent to return false")
	}
}

func TestSkipListGetRank(t *testing.T) {
	sl := newSkipList()

	sl.insert("a", 1.0)
	sl.insert("b", 2.0)
	sl.insert("c", 3.0)

	rank := sl.getRank("a", 1.0)
	if rank != 1 {
		t.Errorf("expected rank 1 for 'a', got %d", rank)
	}

	rank = sl.getRank("b", 2.0)
	if rank != 2 {
		t.Errorf("expected rank 2 for 'b', got %d", rank)
	}

	rank = sl.getRank("c", 3.0)
	if rank != 3 {
		t.Errorf("expected rank 3 for 'c', got %d", rank)
	}

	rank = sl.getRank("nonexistent", 99.0)
	if rank != 0 {
		t.Errorf("expected rank 0 for nonexistent, got %d", rank)
	}
}

func TestSkipListGetByRank(t *testing.T) {
	sl := newSkipList()

	sl.insert("a", 1.0)
	sl.insert("b", 2.0)
	sl.insert("c", 3.0)

	node := sl.getByRank(1)
	if node == nil || node.member != "a" {
		t.Errorf("rank 1 should be 'a', got %v", node)
	}

	node = sl.getByRank(3)
	if node == nil || node.member != "c" {
		t.Errorf("rank 3 should be 'c', got %v", node)
	}

	node = sl.getByRank(4)
	if node != nil {
		t.Errorf("rank 4 should be nil, got %v", node)
	}
}

func TestSkipListGetRangeByRank(t *testing.T) {
	sl := newSkipList()

	sl.insert("a", 1.0)
	sl.insert("b", 2.0)
	sl.insert("c", 3.0)
	sl.insert("d", 4.0)
	sl.insert("e", 5.0)

	result := sl.getRangeByRank(1, 3)
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	expected := []ScoredMember{
		{Member: "a", Score: 1.0},
		{Member: "b", Score: 2.0},
		{Member: "c", Score: 3.0},
	}

	for i, exp := range expected {
		if result[i].Member != exp.Member || result[i].Score != exp.Score {
			t.Errorf("position %d: expected (%s, %f), got (%s, %f)",
				i, exp.Member, exp.Score, result[i].Member, result[i].Score)
		}
	}
}

func TestSkipListGetRangeByScore(t *testing.T) {
	sl := newSkipList()

	sl.insert("a", 1.0)
	sl.insert("b", 2.0)
	sl.insert("c", 3.0)
	sl.insert("d", 4.0)
	sl.insert("e", 5.0)

	result := sl.getRangeByScore(2.0, 4.0, 0, -1)
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	expected := []string{"b", "c", "d"}
	for i, exp := range expected {
		if result[i].Member != exp {
			t.Errorf("position %d: expected %s, got %s", i, exp, result[i].Member)
		}
	}
}

func TestSkipListGetRangeByScoreWithOffset(t *testing.T) {
	sl := newSkipList()

	sl.insert("a", 1.0)
	sl.insert("b", 2.0)
	sl.insert("c", 3.0)
	sl.insert("d", 4.0)

	result := sl.getRangeByScore(1.0, 4.0, 1, 2)
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if result[0].Member != "b" || result[1].Member != "c" {
		t.Errorf("expected [b, c], got [%s, %s]", result[0].Member, result[1].Member)
	}
}

func TestSkipListPopMin(t *testing.T) {
	sl := newSkipList()

	sl.insert("a", 1.0)
	sl.insert("b", 2.0)
	sl.insert("c", 3.0)

	sm := sl.popMin()
	if sm == nil || sm.Member != "a" || sm.Score != 1.0 {
		t.Errorf("expected (a, 1.0), got %v", sm)
	}
	if sl.length != 2 {
		t.Errorf("expected length 2, got %d", sl.length)
	}
}

func TestSkipListPopMax(t *testing.T) {
	sl := newSkipList()

	sl.insert("a", 1.0)
	sl.insert("b", 2.0)
	sl.insert("c", 3.0)

	sm := sl.popMax()
	if sm == nil || sm.Member != "c" || sm.Score != 3.0 {
		t.Errorf("expected (c, 3.0), got %v", sm)
	}
	if sl.length != 2 {
		t.Errorf("expected length 2, got %d", sl.length)
	}
}

func TestSkipListCountInRange(t *testing.T) {
	sl := newSkipList()

	sl.insert("a", 1.0)
	sl.insert("b", 2.0)
	sl.insert("c", 3.0)
	sl.insert("d", 4.0)
	sl.insert("e", 5.0)

	count := sl.countInRange(2.0, 4.0)
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}

	count = sl.countInRange(0.0, 10.0)
	if count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}

	count = sl.countInRange(6.0, 10.0)
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
}

func TestSkipListEmptyOperations(t *testing.T) {
	sl := newSkipList()

	if sl.length != 0 {
		t.Errorf("expected length 0, got %d", sl.length)
	}

	sm := sl.popMin()
	if sm != nil {
		t.Errorf("expected nil for popMin on empty list")
	}

	sm = sl.popMax()
	if sm != nil {
		t.Errorf("expected nil for popMax on empty list")
	}

	result := sl.getRangeByRank(1, 10)
	if len(result) != 0 {
		t.Errorf("expected empty result for getRangeByRank on empty list")
	}
}

func TestSortedSetDataZAdd(t *testing.T) {
	ssd := newSortedSetData()

	added := ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})
	if added != 3 {
		t.Errorf("expected 3 added, got %d", added)
	}

	added = ssd.zadd(map[string]float64{"a": 1.0, "d": 4.0})
	if added != 1 {
		t.Errorf("expected 1 added, got %d", added)
	}

	if ssd.zcard() != 4 {
		t.Errorf("expected 4 members, got %d", ssd.zcard())
	}
}

func TestSortedSetDataZAddUpdateScore(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0})

	added := ssd.zadd(map[string]float64{"a": 5.0})
	if added != 0 {
		t.Errorf("expected 0 added when updating, got %d", added)
	}

	score, exists := ssd.zscore("a")
	if !exists || score != 5.0 {
		t.Errorf("expected score 5.0 for 'a', got %f", score)
	}
}

func TestSortedSetDataZRem(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	removed := ssd.zrem("a", "c")
	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}

	if ssd.zcard() != 1 {
		t.Errorf("expected 1 member, got %d", ssd.zcard())
	}

	removed = ssd.zrem("nonexistent")
	if removed != 0 {
		t.Errorf("expected 0 removed for nonexistent, got %d", removed)
	}
}

func TestSortedSetDataZIncrBy(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0})

	score := ssd.zincrby("a", 5.0)
	if score != 6.0 {
		t.Errorf("expected score 6.0, got %f", score)
	}

	score = ssd.zincrby("b", 10.0)
	if score != 10.0 {
		t.Errorf("expected score 10.0 for new member, got %f", score)
	}
}

func TestSortedSetDataZRank(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	rank, exists := ssd.zrank("a")
	if !exists || rank != 0 {
		t.Errorf("expected rank 0 for 'a', got %d", rank)
	}

	rank, exists = ssd.zrank("c")
	if !exists || rank != 2 {
		t.Errorf("expected rank 2 for 'c', got %d", rank)
	}

	_, exists = ssd.zrank("nonexistent")
	if exists {
		t.Error("expected nonexistent member to not have rank")
	}
}

func TestSortedSetDataZRevRank(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	rank, exists := ssd.zrevrank("a")
	if !exists || rank != 2 {
		t.Errorf("expected revrank 2 for 'a', got %d", rank)
	}

	rank, exists = ssd.zrevrank("c")
	if !exists || rank != 0 {
		t.Errorf("expected revrank 0 for 'c', got %d", rank)
	}
}

func TestSortedSetDataZRange(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0, "d": 4.0, "e": 5.0})

	result := ssd.zrange(0, 2)
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	expected := []string{"a", "b", "c"}
	for i, exp := range expected {
		if result[i].Member != exp {
			t.Errorf("position %d: expected %s, got %s", i, exp, result[i].Member)
		}
	}
}

func TestSortedSetDataZRangeNegativeIndices(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	result := ssd.zrange(-2, -1)
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if result[0].Member != "b" || result[1].Member != "c" {
		t.Errorf("expected [b, c], got [%s, %s]", result[0].Member, result[1].Member)
	}
}

func TestSortedSetDataZRevRange(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	result := ssd.zrevrange(0, 1)
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if result[0].Member != "c" || result[1].Member != "b" {
		t.Errorf("expected [c, b], got [%s, %s]", result[0].Member, result[1].Member)
	}
}

func TestSortedSetDataZRangeByScore(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0, "d": 4.0})

	result := ssd.zrangebyscore(2.0, 4.0, 0, -1)
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	expected := []string{"b", "c", "d"}
	for i, exp := range expected {
		if result[i].Member != exp {
			t.Errorf("position %d: expected %s, got %s", i, exp, result[i].Member)
		}
	}
}

func TestSortedSetDataZRevRangeByScore(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0, "d": 4.0})

	result := ssd.zrevrangebyscore(4.0, 2.0, 0, -1)
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	expected := []string{"d", "c", "b"}
	for i, exp := range expected {
		if result[i].Member != exp {
			t.Errorf("position %d: expected %s, got %s", i, exp, result[i].Member)
		}
	}
}

func TestSortedSetDataZCount(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0, "d": 4.0})

	count := ssd.zcount(2.0, 4.0)
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}

	count = ssd.zcount(0.0, 100.0)
	if count != 4 {
		t.Errorf("expected count 4, got %d", count)
	}
}

func TestSortedSetDataZPopMin(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	sm := ssd.zpopmin()
	if sm == nil || sm.Member != "a" || sm.Score != 1.0 {
		t.Errorf("expected (a, 1.0), got %v", sm)
	}

	if ssd.zcard() != 2 {
		t.Errorf("expected 2 members, got %d", ssd.zcard())
	}
}

func TestSortedSetDataZPopMax(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	sm := ssd.zpopmax()
	if sm == nil || sm.Member != "c" || sm.Score != 3.0 {
		t.Errorf("expected (c, 3.0), got %v", sm)
	}

	if ssd.zcard() != 2 {
		t.Errorf("expected 2 members, got %d", ssd.zcard())
	}
}

func TestSortedSetDataZRemRangeByRank(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0, "d": 4.0, "e": 5.0})

	removed := ssd.zremrangebyrank(1, 3)
	if removed != 3 {
		t.Errorf("expected 3 removed, got %d", removed)
	}

	if ssd.zcard() != 2 {
		t.Errorf("expected 2 members, got %d", ssd.zcard())
	}

	score, exists := ssd.zscore("a")
	if !exists || score != 1.0 {
		t.Error("expected 'a' to still exist")
	}

	score, exists = ssd.zscore("e")
	if !exists || score != 5.0 {
		t.Error("expected 'e' to still exist")
	}
}

func TestSortedSetDataZRemRangeByScore(t *testing.T) {
	ssd := newSortedSetData()

	ssd.zadd(map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0, "d": 4.0, "e": 5.0})

	removed := ssd.zremrangebyscore(2.0, 4.0)
	if removed != 3 {
		t.Errorf("expected 3 removed, got %d", removed)
	}

	if ssd.zcard() != 2 {
		t.Errorf("expected 2 members, got %d", ssd.zcard())
	}
}

func TestSortedSetCacheZAdd(t *testing.T) {
	ssc := NewSortedSetCache()

	added := ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})
	if added != 3 {
		t.Errorf("expected 3 added, got %d", added)
	}

	card, found := ssc.ZCard("myzset")
	if !found || card != 3 {
		t.Errorf("expected card 3, got %d", card)
	}
}

func TestSortedSetCacheZRem(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	removed := ssc.ZRem("myzset", "a", "c")
	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}

	card, _ := ssc.ZCard("myzset")
	if card != 1 {
		t.Errorf("expected card 1, got %d", card)
	}
}

func TestSortedSetCacheZScore(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0})

	score, found := ssc.ZScore("myzset", "a")
	if !found || score != 1.0 {
		t.Errorf("expected score 1.0, got %f", score)
	}

	_, found = ssc.ZScore("myzset", "nonexistent")
	if found {
		t.Error("expected nonexistent member to not be found")
	}

	_, found = ssc.ZScore("nonexistent_set", "a")
	if found {
		t.Error("expected nonexistent key to not be found")
	}
}

func TestSortedSetCacheZIncrBy(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0})

	score, _ := ssc.ZIncrBy("myzset", "a", 5.0)
	if score != 6.0 {
		t.Errorf("expected score 6.0, got %f", score)
	}

	score, _ = ssc.ZIncrBy("myzset", "b", 10.0)
	if score != 10.0 {
		t.Errorf("expected score 10.0 for new member, got %f", score)
	}
}

func TestSortedSetCacheZRank(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	rank, found := ssc.ZRank("myzset", "a")
	if !found || rank != 0 {
		t.Errorf("expected rank 0 for 'a', got %d", rank)
	}

	rank, found = ssc.ZRank("myzset", "c")
	if !found || rank != 2 {
		t.Errorf("expected rank 2 for 'c', got %d", rank)
	}
}

func TestSortedSetCacheZRevRank(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	rank, found := ssc.ZRevRank("myzset", "a")
	if !found || rank != 2 {
		t.Errorf("expected revrank 2 for 'a', got %d", rank)
	}

	rank, found = ssc.ZRevRank("myzset", "c")
	if !found || rank != 0 {
		t.Errorf("expected revrank 0 for 'c', got %d", rank)
	}
}

func TestSortedSetCacheZRange(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0, "d": 4.0})

	result, found := ssc.ZRange("myzset", 0, 2)
	if !found {
		t.Error("expected to find the key")
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	expected := []string{"a", "b", "c"}
	for i, exp := range expected {
		if result[i].Member != exp {
			t.Errorf("position %d: expected %s, got %s", i, exp, result[i].Member)
		}
	}
}

func TestSortedSetCacheZRevRange(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	result, found := ssc.ZRevRange("myzset", 0, 1)
	if !found {
		t.Error("expected to find the key")
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if result[0].Member != "c" || result[1].Member != "b" {
		t.Errorf("expected [c, b], got [%s, %s]", result[0].Member, result[1].Member)
	}
}

func TestSortedSetCacheZRangeByScore(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0, "d": 4.0})

	result, found := ssc.ZRangeByScore("myzset", 2.0, 4.0, 0, -1)
	if !found {
		t.Error("expected to find the key")
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	expected := []string{"b", "c", "d"}
	for i, exp := range expected {
		if result[i].Member != exp {
			t.Errorf("position %d: expected %s, got %s", i, exp, result[i].Member)
		}
	}
}

func TestSortedSetCacheZPopMin(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	sm, found := ssc.ZPopMin("myzset")
	if !found || sm.Member != "a" || sm.Score != 1.0 {
		t.Errorf("expected (a, 1.0), got %v", sm)
	}

	card, _ := ssc.ZCard("myzset")
	if card != 2 {
		t.Errorf("expected card 2, got %d", card)
	}
}

func TestSortedSetCacheZPopMax(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})

	sm, found := ssc.ZPopMax("myzset")
	if !found || sm.Member != "c" || sm.Score != 3.0 {
		t.Errorf("expected (c, 3.0), got %v", sm)
	}

	card, _ := ssc.ZCard("myzset")
	if card != 2 {
		t.Errorf("expected card 2, got %d", card)
	}
}

func TestSortedSetCacheZCount(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0, "d": 4.0})

	count, found := ssc.ZCount("myzset", 2.0, 4.0)
	if !found || count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

func TestSortedSetCacheZRemRangeByRank(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0, "d": 4.0, "e": 5.0})

	removed, _ := ssc.ZRemRangeByRank("myzset", 1, 3)
	if removed != 3 {
		t.Errorf("expected 3 removed, got %d", removed)
	}

	card, _ := ssc.ZCard("myzset")
	if card != 2 {
		t.Errorf("expected card 2, got %d", card)
	}
}

func TestSortedSetCacheZRemRangeByScore(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0, "d": 4.0, "e": 5.0})

	removed, _ := ssc.ZRemRangeByScore("myzset", 2.0, 4.0)
	if removed != 3 {
		t.Errorf("expected 3 removed, got %d", removed)
	}

	card, _ := ssc.ZCard("myzset")
	if card != 2 {
		t.Errorf("expected card 2, got %d", card)
	}
}

func TestSortedSetCacheZUnionStore(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("zset1", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})
	ssc.ZAdd("zset2", 0, map[string]float64{"b": 4.0, "c": 5.0, "d": 6.0})

	count, _ := ssc.ZUnionStore("zunion", 0, []string{"zset1", "zset2"}, nil, "SUM")
	if count != 4 {
		t.Errorf("expected 4 members in union, got %d", count)
	}

	score, found := ssc.ZScore("zunion", "b")
	if !found || score != 6.0 {
		t.Errorf("expected b score 6.0 (2+4), got %f", score)
	}

	score, found = ssc.ZScore("zunion", "a")
	if !found || score != 1.0 {
		t.Errorf("expected a score 1.0, got %f", score)
	}
}

func TestSortedSetCacheZUnionStoreWithWeights(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("zset1", 0, map[string]float64{"a": 1.0, "b": 2.0})
	ssc.ZAdd("zset2", 0, map[string]float64{"b": 3.0, "c": 4.0})

	count, _ := ssc.ZUnionStore("zunion", 0, []string{"zset1", "zset2"}, []float64{2.0, 3.0}, "SUM")
	if count != 3 {
		t.Errorf("expected 3 members, got %d", count)
	}

	score, _ := ssc.ZScore("zunion", "b")
	expected := 2.0*2.0 + 3.0*3.0
	if math.Abs(score-expected) > 0.001 {
		t.Errorf("expected b score %f, got %f", expected, score)
	}
}

func TestSortedSetCacheZInterStore(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("zset1", 0, map[string]float64{"a": 1.0, "b": 2.0, "c": 3.0})
	ssc.ZAdd("zset2", 0, map[string]float64{"b": 4.0, "c": 5.0, "d": 6.0})

	count, _ := ssc.ZInterStore("zinter", 0, []string{"zset1", "zset2"}, nil, "SUM")
	if count != 2 {
		t.Errorf("expected 2 members in intersection, got %d", count)
	}

	score, found := ssc.ZScore("zinter", "b")
	if !found || score != 6.0 {
		t.Errorf("expected b score 6.0 (2+4), got %f", score)
	}

	_, found = ssc.ZScore("zinter", "a")
	if found {
		t.Error("expected 'a' to not be in intersection")
	}
}

func TestSortedSetCacheDelete(t *testing.T) {
	ssc := NewSortedSetCache()

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0})

	deleted := ssc.Delete("myzset")
	if !deleted {
		t.Error("expected delete to return true")
	}

	_, found := ssc.ZCard("myzset")
	if found {
		t.Error("expected key to not exist after delete")
	}
}

func TestSortedSetCacheExists(t *testing.T) {
	ssc := NewSortedSetCache()

	if ssc.Exists("myzset") {
		t.Error("expected nonexistent key to not exist")
	}

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0})

	if !ssc.Exists("myzset") {
		t.Error("expected key to exist after ZAdd")
	}
}

func TestSortedSetCacheNonexistentKey(t *testing.T) {
	ssc := NewSortedSetCache()

	_, found := ssc.ZCard("nonexistent")
	if found {
		t.Error("expected nonexistent key to not be found")
	}

	_, found = ssc.ZScore("nonexistent", "a")
	if found {
		t.Error("expected nonexistent key to not be found")
	}

	_, found = ssc.ZRank("nonexistent", "a")
	if found {
		t.Error("expected nonexistent key to not be found")
	}

	_, found = ssc.ZRange("nonexistent", 0, -1)
	if found {
		t.Error("expected nonexistent key to not be found")
	}

	removed := ssc.ZRem("nonexistent", "a")
	if removed != 0 {
		t.Errorf("expected 0 removed for nonexistent key, got %d", removed)
	}
}

func TestSortedSetCacheLargeDataset(t *testing.T) {
	ssc := NewSortedSetCache()

	members := make(map[string]float64)
	for i := 0; i < 1000; i++ {
		members[string(rune('a'+i%26))+string(rune('0'+i/26))] = float64(i)
	}

	added := ssc.ZAdd("large", 0, members)
	if added != 1000 {
		t.Errorf("expected 1000 added, got %d", added)
	}

	card, _ := ssc.ZCard("large")
	if card != 1000 {
		t.Errorf("expected card 1000, got %d", card)
	}

	rank, found := ssc.ZRank("large", string(rune('a'))+"0")
	if !found || rank != 0 {
		t.Errorf("expected rank 0 for first item, got %d", rank)
	}

	result, found := ssc.ZRange("large", 0, 9)
	if !found || len(result) != 10 {
		t.Errorf("expected 10 results, got %d", len(result))
	}
}

func TestSortedSetCacheWithSharedMemory(t *testing.T) {
	mc := New()
	ssc := NewSortedSetCacheWithMemory(mc)

	ssc.ZAdd("myzset", 0, map[string]float64{"a": 1.0, "b": 2.0})

	card, found := ssc.ZCard("myzset")
	if !found || card != 2 {
		t.Errorf("expected card 2, got %d", card)
	}

	if ssc.GetCache() != mc {
		t.Error("expected shared MemoryCache")
	}
}
