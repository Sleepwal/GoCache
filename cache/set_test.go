package cache

import (
	"testing"
	"time"
)

// TestSetCache_SAdd 测试添加成员
func TestSetCache_SAdd(t *testing.T) {
	sc := NewSetCache()

	// 添加新成员
	added := sc.SAdd("myset", 0, "a", "b", "c")
	if added != 3 {
		t.Errorf("expected 3 added, got %d", added)
	}

	// 添加重复成员
	added = sc.SAdd("myset", 0, "c", "d")
	if added != 1 {
		t.Errorf("expected 1 added, got %d", added)
	}

	card, _ := sc.SCard("myset")
	if card != 4 {
		t.Errorf("expected cardinality 4, got %d", card)
	}
}

// TestSetCache_SRem 测试移除成员
func TestSetCache_SRem(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("myset", 0, "a", "b", "c", "d")

	removed := sc.SRem("myset", "a", "c")
	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}

	members, _ := sc.SMembers("myset")
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

// TestSetCache_SIsMember 测试成员资格
func TestSetCache_SIsMember(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("myset", 0, "a", "b", "c")

	if !sc.SIsMember("myset", "a") {
		t.Error("expected 'a' to be a member")
	}

	if sc.SIsMember("myset", "x") {
		t.Error("expected 'x' to not be a member")
	}
}

// TestSetCache_SCard 测试集合基数
func TestSetCache_SCard(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("myset", 0, "a", "b", "c")

	card, found := sc.SCard("myset")
	if !found || card != 3 {
		t.Errorf("expected cardinality 3, got %d", card)
	}
}

// TestSetCache_SMembers 测试获取所有成员
func TestSetCache_SMembers(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("myset", 0, "a", "b", "c")

	members, found := sc.SMembers("myset")
	if !found || len(members) != 3 {
		t.Errorf("expected 3 members, got %d", len(members))
	}

	memberSet := make(map[any]bool)
	for _, m := range members {
		memberSet[m] = true
	}

	if !memberSet["a"] || !memberSet["b"] || !memberSet["c"] {
		t.Errorf("unexpected members: %v", members)
	}
}

// TestSetCache_SPop 测试随机弹出
func TestSetCache_SPop(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("myset", 0, "a", "b", "c")

	val, found := sc.SPop("myset")
	if !found {
		t.Error("expected to pop a member")
	}

	if val != "a" && val != "b" && val != "c" {
		t.Errorf("unexpected popped value: %v", val)
	}

	card, _ := sc.SCard("myset")
	if card != 2 {
		t.Errorf("expected cardinality 2 after pop, got %d", card)
	}
}

// TestSetCache_SPop_Empty 测试空集合弹出
func TestSetCache_SPop_Empty(t *testing.T) {
	sc := NewSetCache()

	_, found := sc.SPop("nonexistent")
	if found {
		t.Error("expected not to pop from empty set")
	}
}

// TestSetCache_SUnion 测试并集
func TestSetCache_SUnion(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("set1", 0, "a", "b", "c")
	sc.SAdd("set2", 0, "c", "d", "e")

	union := sc.SUnion("set1", "set2")
	if len(union) != 5 {
		t.Errorf("expected 5 members in union, got %d", len(union))
	}

	unionSet := make(map[any]bool)
	for _, m := range union {
		unionSet[m] = true
	}

	expected := []any{"a", "b", "c", "d", "e"}
	for _, exp := range expected {
		if !unionSet[exp] {
			t.Errorf("expected '%v' in union", exp)
		}
	}
}

// TestSetCache_SInter 测试交集
func TestSetCache_SInter(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("set1", 0, "a", "b", "c")
	sc.SAdd("set2", 0, "b", "c", "d")

	inter := sc.SInter("set1", "set2")
	if len(inter) != 2 {
		t.Errorf("expected 2 members in intersection, got %d", len(inter))
	}

	interSet := make(map[any]bool)
	for _, m := range inter {
		interSet[m] = true
	}

	if !interSet["b"] || !interSet["c"] {
		t.Errorf("expected 'b' and 'c' in intersection, got %v", inter)
	}
}

// TestSetCache_SDiff 测试差集
func TestSetCache_SDiff(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("set1", 0, "a", "b", "c")
	sc.SAdd("set2", 0, "b", "c", "d")

	diff := sc.SDiff("set1", "set2")
	if len(diff) != 1 {
		t.Errorf("expected 1 member in difference, got %d", len(diff))
	}

	if diff[0] != "a" {
		t.Errorf("expected 'a' in difference, got %v", diff)
	}
}

// TestSetCache_Delete 测试删除 Set
func TestSetCache_Delete(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("myset", 0, "a", "b")

	deleted := sc.Delete("myset")
	if !deleted {
		t.Error("expected delete to succeed")
	}

	_, found := sc.SCard("myset")
	if found {
		t.Error("expected set to be deleted")
	}
}

// TestSetCache_Exists 测试 Set 是否存在
func TestSetCache_Exists(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("myset", 0, "a")

	if !sc.Exists("myset") {
		t.Error("expected set to exist")
	}

	if sc.Exists("nonexistent") {
		t.Error("expected nonexistent set to not exist")
	}
}

// TestSetCache_Keys 测试获取所有键
func TestSetCache_Keys(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("set1", 0, "a")
	sc.SAdd("set2", 0, "b")
	sc.SAdd("set3", 0, "c")

	keys := sc.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

// TestSetCache_Clear 测试清空
func TestSetCache_Clear(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("set1", 0, "a")
	sc.SAdd("set2", 0, "b")

	sc.Clear()

	if sc.Count() != 0 {
		t.Errorf("expected 0 sets after clear, got %d", sc.Count())
	}
}

// TestSetCache_TTL 测试 TTL 过期
func TestSetCache_TTL(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("myset", 50*time.Millisecond, "a", "b")

	// 立即获取应该成功
	members, found := sc.SMembers("myset")
	if !found || len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	_, found = sc.SMembers("myset")
	if found {
		t.Error("expected set to be expired")
	}
}

// TestSetCache_SRem_NotFound 测试移除不存在的成员
func TestSetCache_SRem_NotFound(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("myset", 0, "a", "b")

	removed := sc.SRem("myset", "x", "y")
	if removed != 0 {
		t.Errorf("expected 0 removed, got %d", removed)
	}
}

// TestSetCache_SUnion_NotFound 测试并集不存在的集合
func TestSetCache_SUnion_NotFound(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("set1", 0, "a", "b")

	union := sc.SUnion("set1", "nonexistent")
	if len(union) != 2 {
		t.Errorf("expected 2 members in union, got %d", len(union))
	}
}

// TestSetCache_SInter_NotFound 测试交集不存在的集合
func TestSetCache_SInter_NotFound(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("set1", 0, "a", "b")

	inter := sc.SInter("set1", "nonexistent")
	if len(inter) != 0 {
		t.Errorf("expected 0 members in intersection, got %d", len(inter))
	}
}

// TestSetCache_SAdd_Duplicate 测试添加重复成员
func TestSetCache_SAdd_Duplicate(t *testing.T) {
	sc := NewSetCache()

	sc.SAdd("myset", 0, "a", "b", "c")

	// 添加全部重复
	added := sc.SAdd("myset", 0, "a", "b", "c")
	if added != 0 {
		t.Errorf("expected 0 added, got %d", added)
	}

	card, _ := sc.SCard("myset")
	if card != 3 {
		t.Errorf("expected cardinality 3, got %d", card)
	}
}
