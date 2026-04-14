package cache

import (
	"testing"
	"time"
)

// TestNamespaceCache_Basic 测试基本操作
func TestNamespaceCache_Basic(t *testing.T) {
	c := New()
	ns := NewNamespaceCache(c, "user")

	ns.Set("key1", "value1", 0)
	ns.Set("key2", "value2", 0)

	val, found := ns.Get("key1")
	if !found || val != "value1" {
		t.Errorf("expected 'value1', got %v", val)
	}

	// 验证键格式
	keys := ns.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

// TestNamespaceCache_Isolation 测试命名空间隔离
func TestNamespaceCache_Isolation(t *testing.T) {
	c := New()

	ns1 := NewNamespaceCache(c, "ns1")
	ns2 := NewNamespaceCache(c, "ns2")

	ns1.Set("key", "value1", 0)
	ns2.Set("key", "value2", 0)

	val1, _ := ns1.Get("key")
	val2, _ := ns2.Get("key")

	if val1 == val2 {
		t.Errorf("expected different values, got %v and %v", val1, val2)
	}

	if val1 != "value1" {
		t.Errorf("expected 'value1' from ns1, got %v", val1)
	}

	if val2 != "value2" {
		t.Errorf("expected 'value2' from ns2, got %v", val2)
	}
}

// TestNamespaceCache_Clear 测试清空命名空间
func TestNamespaceCache_Clear(t *testing.T) {
	c := New()

	ns1 := NewNamespaceCache(c, "ns1")
	ns2 := NewNamespaceCache(c, "ns2")

	ns1.Set("key1", "value1", 0)
	ns2.Set("key2", "value2", 0)

	deleted := ns1.Clear()
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	// ns2 应该不受影响
	if ns2.Count() != 1 {
		t.Errorf("expected ns2 to have 1 item, got %d", ns2.Count())
	}

	// ns1 应该为空
	if ns1.Count() != 0 {
		t.Errorf("expected ns1 to have 0 items, got %d", ns1.Count())
	}
}

// TestNamespaceCache_Keys 测试获取键
func TestNamespaceCache_Keys(t *testing.T) {
	c := New()
	ns := NewNamespaceCache(c, "user")

	ns.Set("a", 1, 0)
	ns.Set("b", 2, 0)
	ns.Set("c", 3, 0)

	keys := ns.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}

	if !keySet["a"] || !keySet["b"] || !keySet["c"] {
		t.Errorf("unexpected keys: %v", keys)
	}
}

// TestNamespaceCache_TTL 测试 TTL
func TestNamespaceCache_TTL(t *testing.T) {
	c := New()
	ns := NewNamespaceCache(c, "temp")

	ns.Set("key", "value", 50*time.Millisecond)

	// 立即获取应该成功
	val, found := ns.Get("key")
	if !found || val != "value" {
		t.Errorf("expected 'value', got %v", val)
	}

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	_, found = ns.Get("key")
	if found {
		t.Error("expected key to be expired")
	}
}

// TestNamespaceCache_NamespacedKey 测试获取完整键
func TestNamespaceCache_NamespacedKey(t *testing.T) {
	ns := NewNamespaceCache(New(), "user")

	fullKey := ns.NamespacedKey("key1")
	expected := "user:key1"
	if fullKey != expected {
		t.Errorf("expected '%s', got '%s'", expected, fullKey)
	}
}

// TestMultiNamespaceCache 测试多命名空间
func TestMultiNamespaceCache(t *testing.T) {
	c := New()
	mnc := NewMultiNamespaceCache(c)

	ns1 := mnc.Namespace("ns1")
	ns2 := mnc.Namespace("ns2")

	ns1.Set("key", "value1", 0)
	ns2.Set("key", "value2", 0)

	val1, _ := ns1.Get("key")
	val2, _ := ns2.Get("key")

	if val1 != "value1" || val2 != "value2" {
		t.Errorf("expected isolated values, got %v and %v", val1, val2)
	}

	// 列出命名空间
	namespaces := mnc.ListNamespaces()
	if len(namespaces) != 2 {
		t.Errorf("expected 2 namespaces, got %d", len(namespaces))
	}
}

// TestFormatKey 测试键格式化
func TestFormatKey(t *testing.T) {
	fullKey := FormatKey("user", "key1")
	if fullKey != "user:key1" {
		t.Errorf("expected 'user:key1', got '%s'", fullKey)
	}
}

// TestParseKey 测试键解析
func TestParseKey(t *testing.T) {
	ns, key, ok := ParseKey("user:key1")
	if !ok {
		t.Error("expected parse to succeed")
	}
	if ns != "user" || key != "key1" {
		t.Errorf("expected 'user' and 'key1', got '%s' and '%s'", ns, key)
	}

	// 无效格式
	_, _, ok = ParseKey("invalid")
	if ok {
		t.Error("expected parse to fail for invalid key")
	}
}
