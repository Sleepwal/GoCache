package cache

import (
	"testing"
	"time"
)

// TestListCache_LPush 测试左侧推入
func TestListCache_LPush(t *testing.T) {
	lc := NewListCache()

	// 推入单个值
	length := lc.LPush("mylist", 0, "a")
	if length != 1 {
		t.Errorf("expected length 1, got %d", length)
	}

	// 推入多个值
	length = lc.LPush("mylist", 0, "b", "c", "d")
	if length != 4 {
		t.Errorf("expected length 4, got %d", length)
	}

	// 验证顺序: LPush("b","c","d") 后列表为 [d,c,b,a]
	vals, _ := lc.LRange("mylist", 0, -1)
	if len(vals) != 4 || vals[0] != "d" || vals[1] != "c" || vals[2] != "b" || vals[3] != "a" {
		t.Errorf("expected [d, c, b, a], got %v", vals)
	}
}

// TestListCache_RPush 测试右侧推入
func TestListCache_RPush(t *testing.T) {
	lc := NewListCache()

	length := lc.RPush("mylist", 0, "a", "b", "c")
	if length != 3 {
		t.Errorf("expected length 3, got %d", length)
	}

	// 验证顺序: a, b, c
	vals, _ := lc.LRange("mylist", 0, -1)
	if len(vals) != 3 || vals[0] != "a" || vals[1] != "b" || vals[2] != "c" {
		t.Errorf("expected [a, b, c], got %v", vals)
	}
}

// TestListCache_LPop 测试左侧弹出
func TestListCache_LPop(t *testing.T) {
	lc := NewListCache()

	lc.RPush("mylist", 0, "a", "b", "c")

	val, found := lc.LPop("mylist")
	if !found || val != "a" {
		t.Errorf("expected 'a', got %v", val)
	}

	length, _ := lc.LLen("mylist")
	if length != 2 {
		t.Errorf("expected length 2, got %d", length)
	}
}

// TestListCache_RPop 测试右侧弹出
func TestListCache_RPop(t *testing.T) {
	lc := NewListCache()

	lc.RPush("mylist", 0, "a", "b", "c")

	val, found := lc.RPop("mylist")
	if !found || val != "c" {
		t.Errorf("expected 'c', got %v", val)
	}

	length, _ := lc.LLen("mylist")
	if length != 2 {
		t.Errorf("expected length 2, got %d", length)
	}
}

// TestListCache_LPop_Empty 测试空列表弹出
func TestListCache_LPop_Empty(t *testing.T) {
	lc := NewListCache()

	_, found := lc.LPop("nonexistent")
	if found {
		t.Error("expected not found for empty list")
	}
}

// TestListCache_LRange 测试范围查询
func TestListCache_LRange(t *testing.T) {
	lc := NewListCache()

	lc.RPush("mylist", 0, "a", "b", "c", "d", "e")

	// 正常范围
	vals, found := lc.LRange("mylist", 1, 3)
	if !found || len(vals) != 3 {
		t.Errorf("expected 3 elements, got %d", len(vals))
	}
	if vals[0] != "b" || vals[1] != "c" || vals[2] != "d" {
		t.Errorf("expected [b, c, d], got %v", vals)
	}

	// 负数索引
	vals, found = lc.LRange("mylist", -2, -1)
	if !found || len(vals) != 2 {
		t.Errorf("expected 2 elements, got %d", len(vals))
	}
	if vals[0] != "d" || vals[1] != "e" {
		t.Errorf("expected [d, e], got %v", vals)
	}
}

// TestListCache_LRange_NotFound 测试不存在的列表
func TestListCache_LRange_NotFound(t *testing.T) {
	lc := NewListCache()

	_, found := lc.LRange("nonexistent", 0, -1)
	if found {
		t.Error("expected not found")
	}
}

// TestListCache_LIndex 测试按索引获取
func TestListCache_LIndex(t *testing.T) {
	lc := NewListCache()

	lc.RPush("mylist", 0, "a", "b", "c")

	// 正数索引
	val, found := lc.LIndex("mylist", 1)
	if !found || val != "b" {
		t.Errorf("expected 'b', got %v", val)
	}

	// 负数索引
	val, found = lc.LIndex("mylist", -1)
	if !found || val != "c" {
		t.Errorf("expected 'c', got %v", val)
	}

	// 越界
	_, found = lc.LIndex("mylist", 10)
	if found {
		t.Error("expected not found for out of range")
	}
}

// TestListCache_LLen 测试列表长度
func TestListCache_LLen(t *testing.T) {
	lc := NewListCache()

	lc.RPush("mylist", 0, "a", "b", "c")

	length, found := lc.LLen("mylist")
	if !found || length != 3 {
		t.Errorf("expected length 3, got %d", length)
	}
}

// TestListCache_LTrim 测试修剪
func TestListCache_LTrim(t *testing.T) {
	lc := NewListCache()

	lc.RPush("mylist", 0, "a", "b", "c", "d", "e")

	lc.LTrim("mylist", 1, 3)

	vals, _ := lc.LRange("mylist", 0, -1)
	if len(vals) != 3 {
		t.Errorf("expected 3 elements, got %d", len(vals))
	}
	if vals[0] != "b" || vals[1] != "c" || vals[2] != "d" {
		t.Errorf("expected [b, c, d], got %v", vals)
	}
}

// TestListCache_LRem 测试删除元素
func TestListCache_LRem(t *testing.T) {
	lc := NewListCache()

	lc.RPush("mylist", 0, "a", "b", "a", "c", "a", "d")

	// 从头删除 2 个 "a"
	removed := lc.LRem("mylist", 2, "a")
	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}

	vals, _ := lc.LRange("mylist", 0, -1)
	if len(vals) != 4 {
		t.Errorf("expected 4 elements, got %d", len(vals))
	}

	// 删除所有 "a"
	removed = lc.LRem("mylist", 0, "a")
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	// 从尾删除 "d"
	removed = lc.LRem("mylist", -1, "d")
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
}

// TestListCache_Delete 测试删除列表
func TestListCache_Delete(t *testing.T) {
	lc := NewListCache()

	lc.RPush("mylist", 0, "a", "b")

	deleted := lc.Delete("mylist")
	if !deleted {
		t.Error("expected delete to succeed")
	}

	_, found := lc.LLen("mylist")
	if found {
		t.Error("expected list to be deleted")
	}
}

// TestListCache_Exists 测试列表是否存在
func TestListCache_Exists(t *testing.T) {
	lc := NewListCache()

	lc.RPush("mylist", 0, "a")

	if !lc.Exists("mylist") {
		t.Error("expected list to exist")
	}

	if lc.Exists("nonexistent") {
		t.Error("expected nonexistent list to not exist")
	}
}

// TestListCache_Keys 测试获取所有键
func TestListCache_Keys(t *testing.T) {
	lc := NewListCache()

	lc.RPush("list1", 0, "a")
	lc.RPush("list2", 0, "b")
	lc.RPush("list3", 0, "c")

	keys := lc.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

// TestListCache_Clear 测试清空
func TestListCache_Clear(t *testing.T) {
	lc := NewListCache()

	lc.RPush("list1", 0, "a")
	lc.RPush("list2", 0, "b")

	lc.Clear()

	if lc.Count() != 0 {
		t.Errorf("expected 0 lists after clear, got %d", lc.Count())
	}
}

// TestListCache_TTL 测试 TTL 过期
func TestListCache_TTL(t *testing.T) {
	lc := NewListCache()

	lc.RPush("mylist", 50*time.Millisecond, "a", "b")

	// 立即获取应该成功
	vals, found := lc.LRange("mylist", 0, -1)
	if !found || len(vals) != 2 {
		t.Errorf("expected 2 elements, got %d", len(vals))
	}

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	_, found = lc.LRange("mylist", 0, -1)
	if found {
		t.Error("expected list to be expired")
	}
}

// TestListCache_LTrim_Empty 测试修剪空列表
func TestListCache_LTrim_Empty(t *testing.T) {
	lc := NewListCache()

	lc.RPush("mylist", 0, "a")

	// 修剪到空范围
	lc.LTrim("mylist", 5, 10)

	length, _ := lc.LLen("mylist")
	if length != 0 {
		t.Errorf("expected 0 elements after trim, got %d", length)
	}
}

// TestListCache_LRem_NotFound 测试删除不存在的元素
func TestListCache_LRem_NotFound(t *testing.T) {
	lc := NewListCache()

	lc.RPush("mylist", 0, "a", "b", "c")

	removed := lc.LRem("mylist", 0, "x")
	if removed != 0 {
		t.Errorf("expected 0 removed, got %d", removed)
	}
}
