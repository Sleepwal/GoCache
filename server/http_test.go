package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"GoCache/cache"
)

// TestHTTPServer_SetAndGet 测试设置和获取缓存
func TestHTTPServer_SetAndGet(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// 测试 POST 设置缓存
	body := `{"value": "test_value"}`
	req := httptest.NewRequest(http.MethodPost, "/cache/test_key", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.cacheHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	// 测试 GET 获取缓存
	req = httptest.NewRequest(http.MethodGet, "/cache/test_key", nil)
	w = httptest.NewRecorder()
	hs.cacheHandler(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["value"] != "test_value" {
		t.Errorf("expected 'test_value', got %v", result["value"])
	}
}

// TestHTTPServer_GetNotFound 测试获取不存在的键
func TestHTTPServer_GetNotFound(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	req := httptest.NewRequest(http.MethodGet, "/cache/nonexistent", nil)
	w := httptest.NewRecorder()
	hs.cacheHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

// TestHTTPServer_Delete 测试删除缓存
func TestHTTPServer_Delete(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// 先设置
	hs.cache.Set("key1", "value1", 0)

	// 再删除
	req := httptest.NewRequest(http.MethodDelete, "/cache/key1", nil)
	w := httptest.NewRecorder()
	hs.cacheHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// 验证已删除
	req = httptest.NewRequest(http.MethodGet, "/cache/key1", nil)
	w = httptest.NewRecorder()
	hs.cacheHandler(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 after delete, got %d", resp.StatusCode)
	}
}

// TestHTTPServer_SetWithTTL 测试带 TTL 的设置缓存
func TestHTTPServer_SetWithTTL(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// 使用 JSON body 设置 TTL
	body := `{"value": "temp_value", "ttl": "100ms"}`
	req := httptest.NewRequest(http.MethodPost, "/cache/temp_key", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.cacheHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	// 立即获取应该成功
	req = httptest.NewRequest(http.MethodGet, "/cache/temp_key", nil)
	w = httptest.NewRecorder()
	hs.cacheHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Error("expected key to exist immediately after set")
	}

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	req = httptest.NewRequest(http.MethodGet, "/cache/temp_key", nil)
	w = httptest.NewRecorder()
	hs.cacheHandler(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Error("expected key to be expired")
	}
}

// TestHTTPServer_SetWithQueryTTL 测试使用 query string 设置 TTL
func TestHTTPServer_SetWithQueryTTL(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	body := `{"value": "temp_value"}`
	req := httptest.NewRequest(http.MethodPost, "/cache/temp_key?ttl=100ms", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.cacheHandler(w, req)

	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Result().StatusCode)
	}

	time.Sleep(150 * time.Millisecond)

	req = httptest.NewRequest(http.MethodGet, "/cache/temp_key", nil)
	w = httptest.NewRecorder()
	hs.cacheHandler(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Error("expected key to be expired")
	}
}

// TestHTTPServer_Keys 测试获取所有键
func TestHTTPServer_Keys(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.cache.Set("key1", "value1", 0)
	hs.cache.Set("key2", "value2", 0)
	hs.cache.Set("key3", "value3", 0)

	req := httptest.NewRequest(http.MethodGet, "/cache/keys", nil)
	w := httptest.NewRecorder()
	hs.keysHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if int(result["count"].(float64)) != 3 {
		t.Errorf("expected 3 keys, got %v", result["count"])
	}
}

// TestHTTPServer_Stats 测试获取统计信息
func TestHTTPServer_Stats(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.cache.Set("key1", "value1", 0)
	hs.cache.Get("key1")
	hs.cache.Get("nonexistent")

	req := httptest.NewRequest(http.MethodGet, "/cache/stats", nil)
	w := httptest.NewRecorder()
	hs.statsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if int(result["sets"].(float64)) != 1 {
		t.Errorf("expected 1 set, got %v", result["sets"])
	}
	if int(result["hits"].(float64)) != 1 {
		t.Errorf("expected 1 hit, got %v", result["hits"])
	}
	if int(result["misses"].(float64)) != 1 {
		t.Errorf("expected 1 miss, got %v", result["misses"])
	}
}

// TestHTTPServer_Clear 测试清空缓存
func TestHTTPServer_Clear(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.cache.Set("key1", "value1", 0)
	hs.cache.Set("key2", "value2", 0)

	req := httptest.NewRequest(http.MethodPost, "/cache/clear", nil)
	w := httptest.NewRecorder()
	hs.clearHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if hs.cache.Count() != 0 {
		t.Errorf("expected 0 items after clear, got %d", hs.cache.Count())
	}
}

// TestHTTPServer_MethodNotAllowed 测试不允许的方法
func TestHTTPServer_MethodNotAllowed(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// GET 到 /cache/clear 应该返回 405
	req := httptest.NewRequest(http.MethodGet, "/cache/clear", nil)
	w := httptest.NewRecorder()
	hs.clearHandler(w, req)

	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Result().StatusCode)
	}
}

// TestHTTPServer_InvalidTTL 测试无效 TTL
func TestHTTPServer_InvalidTTL(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	body := `{"value": "test", "ttl": "invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/cache/key", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.cacheHandler(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid TTL, got %d", w.Result().StatusCode)
	}
}

// TestHTTPServer_InvalidJSON 测试无效 JSON
func TestHTTPServer_InvalidJSON(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	body := `invalid json`
	req := httptest.NewRequest(http.MethodPost, "/cache/key", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.cacheHandler(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid JSON, got %d", w.Result().StatusCode)
	}
}

// TestHTTPServer_CORS 测试 CORS 头
func TestHTTPServer_CORS(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// 测试 OPTIONS 预检请求
	req := httptest.NewRequest(http.MethodOptions, "/cache/keys", nil)
	w := httptest.NewRecorder()

	// 手动调用 keysHandler 的 CORS 包装
	handler := hs.corsMiddleware(hs.keysHandler)
	handler(w, req)

	resp := w.Result()
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header to be set")
	}
	if resp.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected CORS methods header to be set")
	}
}

// TestHTTPServer_DeleteNotFound 测试删除不存在的键
func TestHTTPServer_DeleteNotFound(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	req := httptest.NewRequest(http.MethodDelete, "/cache/nonexistent", nil)
	w := httptest.NewRecorder()
	hs.cacheHandler(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 for deleting nonexistent key, got %d", w.Result().StatusCode)
	}
}

// TestHTTPServer_FullIntegration 测试完整集成
func TestHTTPServer_FullIntegration(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// 1. 设置缓存
	body := `{"value": "integration_test"}`
	req := httptest.NewRequest(http.MethodPost, "/cache/integration", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.cacheHandler(w, req)

	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Result().StatusCode)
	}

	// 2. 获取缓存
	req = httptest.NewRequest(http.MethodGet, "/cache/integration", nil)
	w = httptest.NewRecorder()
	hs.cacheHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	// 3. 检查统计
	req = httptest.NewRequest(http.MethodGet, "/cache/stats", nil)
	w = httptest.NewRecorder()
	hs.statsHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for stats, got %d", w.Result().StatusCode)
	}

	// 4. 清空缓存
	req = httptest.NewRequest(http.MethodPost, "/cache/clear", nil)
	w = httptest.NewRecorder()
	hs.clearHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for clear, got %d", w.Result().StatusCode)
	}

	// 5. 验证已清空
	req = httptest.NewRequest(http.MethodGet, "/cache/integration", nil)
	w = httptest.NewRecorder()
	hs.cacheHandler(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 after clear, got %d", w.Result().StatusCode)
	}
}

// TestHTTPServer_CustomCache 测试自定义缓存
func TestHTTPServer_CustomCache(t *testing.T) {
	c := cache.New()
	c.Set("custom", "value", 0)

	hs := NewHTTPServerWithCache(HTTPServerConfig{Port: 0}, c)

	req := httptest.NewRequest(http.MethodGet, "/cache/custom", nil)
	w := httptest.NewRecorder()
	hs.cacheHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if result["value"] != "value" {
		t.Errorf("expected 'value', got %v", result["value"])
	}
}

// TestHTTPServer_GetCache 测试获取内部缓存
func TestHTTPServer_GetCache(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	c := hs.GetCache()
	if c == nil {
		t.Error("expected cache to be non-nil")
	}

	c.Set("test", "value", 0)
	val, found := c.Get("test")
	if !found || val != "value" {
		t.Errorf("expected 'value', got %v", val)
	}
}

// ===================== New Tests for Phase 1 =====================

// TestHTTPServer_GetDel 测试 GETDEL 操作
func TestHTTPServer_GetDel(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// 先设置
	hs.cache.Set("key1", "value1", 0)

	// GETDEL 应该返回值并删除
	req := httptest.NewRequest(http.MethodGet, "/cache/key1?action=getdel", nil)
	w := httptest.NewRecorder()
	hs.cacheHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["value"] != "value1" {
		t.Errorf("expected 'value1', got %v", result["value"])
	}

	// 再次 GETDEL 应该返回 404
	req = httptest.NewRequest(http.MethodGet, "/cache/key1?action=getdel", nil)
	w = httptest.NewRecorder()
	hs.cacheHandler(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 after getdel, got %d", w.Result().StatusCode)
	}
}

// TestHTTPServer_Health 测试健康检查端点
func TestHTTPServer_Health(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})
	hs.startTime = time.Now()

	hs.cache.Set("key1", "value1", 0)
	hs.listCache.LPush("list1", 0, "a")
	hs.hashCache.HSetSingle("hash1", "field1", 0, "val1")
	hs.setCache.SAdd("set1", 0, "member1")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	hs.healthHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", result["status"])
	}
	// All data structures now share the same MemoryCache,
	// so Count() returns the total number of keys (4)
	if int(result["cache_count"].(float64)) != 4 {
		t.Errorf("expected cache_count 4, got %v", result["cache_count"])
	}
	if int(result["list_count"].(float64)) != 4 {
		t.Errorf("expected list_count 4, got %v", result["list_count"])
	}
	if int(result["hash_count"].(float64)) != 4 {
		t.Errorf("expected hash_count 4, got %v", result["hash_count"])
	}
	if int(result["set_count"].(float64)) != 4 {
		t.Errorf("expected set_count 4, got %v", result["set_count"])
	}
}

// TestHTTPServer_StringSetAndGet 测试 String SET/GET
func TestHTTPServer_StringSetAndGet(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// SET
	body := `{"value": "hello"}`
	req := httptest.NewRequest(http.MethodPost, "/cache/string/mykey", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.stringHandler(w, req)

	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Result().StatusCode)
	}

	// GET
	req = httptest.NewRequest(http.MethodGet, "/cache/string/mykey", nil)
	w = httptest.NewRecorder()
	hs.stringHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if result["value"] != "hello" {
		t.Errorf("expected 'hello', got %v", result["value"])
	}
}

// TestHTTPServer_StringAppend 测试 String APPEND
func TestHTTPServer_StringAppend(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// 先设置
	hs.stringCache.Set("mykey", "hello", 0)

	// APPEND
	body := `{"value": " world"}`
	req := httptest.NewRequest(http.MethodPost, "/cache/string/mykey/append", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.stringHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if int(result["length"].(float64)) != 11 {
		t.Errorf("expected length 11, got %v", result["length"])
	}
}

// TestHTTPServer_StringIncrDecr 测试 String INCR/DECR
func TestHTTPServer_StringIncrDecr(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// INCR (初始为 0)
	req := httptest.NewRequest(http.MethodPost, "/cache/string/counter/incr", nil)
	w := httptest.NewRecorder()
	hs.stringHandler(w, req)

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if int64(result["value"].(float64)) != 1 {
		t.Errorf("expected value 1, got %v", result["value"])
	}

	// INCRBY
	req = httptest.NewRequest(http.MethodPost, "/cache/string/counter/incr?n=5", nil)
	w = httptest.NewRecorder()
	hs.stringHandler(w, req)

	json.NewDecoder(w.Result().Body).Decode(&result)
	if int64(result["value"].(float64)) != 6 {
		t.Errorf("expected value 6, got %v", result["value"])
	}

	// DECRBY
	req = httptest.NewRequest(http.MethodPost, "/cache/string/counter/decr?n=2", nil)
	w = httptest.NewRecorder()
	hs.stringHandler(w, req)

	json.NewDecoder(w.Result().Body).Decode(&result)
	if int64(result["value"].(float64)) != 4 {
		t.Errorf("expected value 4, got %v", result["value"])
	}
}

// TestHTTPServer_StringRange 测试 String RANGE
func TestHTTPServer_StringRange(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.stringCache.Set("mykey", "hello world", 0)

	req := httptest.NewRequest(http.MethodGet, "/cache/string/mykey/range?start=0&end=4", nil)
	w := httptest.NewRecorder()
	hs.stringHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if result["value"] != "hello" {
		t.Errorf("expected 'hello', got %v", result["value"])
	}
}

// TestHTTPServer_StringStrLen 测试 String STRLEN
func TestHTTPServer_StringStrLen(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.stringCache.Set("mykey", "hello", 0)

	req := httptest.NewRequest(http.MethodGet, "/cache/string/mykey/strlen", nil)
	w := httptest.NewRecorder()
	hs.stringHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if int(result["length"].(float64)) != 5 {
		t.Errorf("expected length 5, got %v", result["length"])
	}
}

// TestHTTPServer_StringGetSet 测试 String GETSET
func TestHTTPServer_StringGetSet(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.stringCache.Set("mykey", "old_value", 0)

	body := `{"value": "new_value"}`
	req := httptest.NewRequest(http.MethodPost, "/cache/string/mykey/getset", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.stringHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if result["value"] != "old_value" {
		t.Errorf("expected 'old_value', got %v", result["value"])
	}
	if result["existed"] != true {
		t.Errorf("expected existed true, got %v", result["existed"])
	}
}

// TestHTTPServer_ListLPushRPush 测试 List LPUSH/RPUSH
func TestHTTPServer_ListLPushRPush(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// LPUSH
	body := `{"values": ["a", "b", "c"]}`
	req := httptest.NewRequest(http.MethodPost, "/cache/list/mylist/lpush", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.listHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if int(result["length"].(float64)) != 3 {
		t.Errorf("expected length 3, got %v", result["length"])
	}
}

// TestHTTPServer_ListLPopRPop 测试 List LPOP/RPOP
func TestHTTPServer_ListLPopRPop(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.listCache.RPush("mylist", 0, "a", "b", "c")

	// LPOP
	req := httptest.NewRequest(http.MethodPost, "/cache/list/mylist/lpop", nil)
	w := httptest.NewRecorder()
	hs.listHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if result["value"] != "a" {
		t.Errorf("expected 'a', got %v", result["value"])
	}

	// RPOP
	req = httptest.NewRequest(http.MethodPost, "/cache/list/mylist/rpop", nil)
	w = httptest.NewRecorder()
	hs.listHandler(w, req)

	json.NewDecoder(w.Result().Body).Decode(&result)
	if result["value"] != "c" {
		t.Errorf("expected 'c', got %v", result["value"])
	}
}

// TestHTTPServer_ListRange 测试 List LRANGE
func TestHTTPServer_ListRange(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.listCache.RPush("mylist", 0, "a", "b", "c", "d")

	req := httptest.NewRequest(http.MethodGet, "/cache/list/mylist/range?start=0&stop=2", nil)
	w := httptest.NewRecorder()
	hs.listHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	values := result["values"].([]any)
	if len(values) != 3 {
		t.Errorf("expected 3 values, got %d", len(values))
	}
	if values[0] != "a" || values[1] != "b" || values[2] != "c" {
		t.Errorf("expected [a, b, c], got %v", values)
	}
}

// TestHTTPServer_ListLen 测试 List LLEN
func TestHTTPServer_ListLen(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.listCache.RPush("mylist", 0, "a", "b")

	req := httptest.NewRequest(http.MethodGet, "/cache/list/mylist/len", nil)
	w := httptest.NewRecorder()
	hs.listHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if int(result["length"].(float64)) != 2 {
		t.Errorf("expected length 2, got %v", result["length"])
	}
}

// TestHTTPServer_HashHSetHGet 测试 Hash HSET/HGET
func TestHTTPServer_HashHSetHGet(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// HSET
	body := `{"fields": {"name": "Alice", "age": 30}}`
	req := httptest.NewRequest(http.MethodPost, "/cache/hash/user1/hset", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.hashHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	// HGET
	req = httptest.NewRequest(http.MethodGet, "/cache/hash/user1/hget?field=name", nil)
	w = httptest.NewRecorder()
	hs.hashHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if result["value"] != "Alice" {
		t.Errorf("expected 'Alice', got %v", result["value"])
	}
}

// TestHTTPServer_HashHGetAll 测试 Hash HGETALL
func TestHTTPServer_HashHGetAll(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.hashCache.HSet("user1", 0, map[string]any{"name": "Bob", "age": 25})

	req := httptest.NewRequest(http.MethodGet, "/cache/hash/user1/hgetall", nil)
	w := httptest.NewRecorder()
	hs.hashHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	fields := result["fields"].(map[string]any)
	if fields["name"] != "Bob" {
		t.Errorf("expected name 'Bob', got %v", fields["name"])
	}
}

// TestHTTPServer_HashHDel 测试 Hash HDEL
func TestHTTPServer_HashHDel(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.hashCache.HSet("user1", 0, map[string]any{"name": "Alice", "age": 30})

	body := `{"fields": ["age"]}`
	req := httptest.NewRequest(http.MethodPost, "/cache/hash/user1/hdel", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.hashHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if int(result["fields_deleted"].(float64)) != 1 {
		t.Errorf("expected 1 deleted, got %v", result["fields_deleted"])
	}

	// Verify deletion
	req = httptest.NewRequest(http.MethodGet, "/cache/hash/user1/hget?field=age", nil)
	w = httptest.NewRecorder()
	hs.hashHandler(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 after delete, got %d", w.Result().StatusCode)
	}
}

// TestHTTPServer_HashHExists 测试 Hash HEXISTS
func TestHTTPServer_HashHExists(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.hashCache.HSetSingle("user1", "name", 0, "Alice")

	req := httptest.NewRequest(http.MethodGet, "/cache/hash/user1/hexists?field=name", nil)
	w := httptest.NewRecorder()
	hs.hashHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if result["exists"] != true {
		t.Errorf("expected exists true, got %v", result["exists"])
	}
}

// TestHTTPServer_HashHLen 测试 Hash HLEN
func TestHTTPServer_HashHLen(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.hashCache.HSet("user1", 0, map[string]any{"name": "Alice", "age": 30})

	req := httptest.NewRequest(http.MethodGet, "/cache/hash/user1/hlen", nil)
	w := httptest.NewRecorder()
	hs.hashHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if int(result["length"].(float64)) != 2 {
		t.Errorf("expected length 2, got %v", result["length"])
	}
}

// TestHTTPServer_SetSAddSMembers 测试 Set SADD/SMEMBERS
func TestHTTPServer_SetSAddSMembers(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	// SADD
	body := `{"members": ["a", "b", "c"]}`
	req := httptest.NewRequest(http.MethodPost, "/cache/set/myset/sadd", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.setHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if int(result["added"].(float64)) != 3 {
		t.Errorf("expected 3 added, got %v", result["added"])
	}

	// SMEMBERS
	req = httptest.NewRequest(http.MethodGet, "/cache/set/myset/smembers", nil)
	w = httptest.NewRecorder()
	hs.setHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	json.NewDecoder(w.Result().Body).Decode(&result)
	members := result["members"].([]any)
	if len(members) != 3 {
		t.Errorf("expected 3 members, got %d", len(members))
	}
}

// TestHTTPServer_SetSIsMember 测试 Set SISMEMBER
func TestHTTPServer_SetSIsMember(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.setCache.SAdd("myset", 0, "a", "b", "c")

	req := httptest.NewRequest(http.MethodGet, "/cache/set/myset/sismember?member=a", nil)
	w := httptest.NewRecorder()
	hs.setHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if result["exists"] != true {
		t.Errorf("expected exists true, got %v", result["exists"])
	}
}

// TestHTTPServer_SetSRem 测试 Set SREM
func TestHTTPServer_SetSRem(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.setCache.SAdd("myset", 0, "a", "b", "c")

	body := `{"members": ["b"]}`
	req := httptest.NewRequest(http.MethodPost, "/cache/set/myset/srem", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.setHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if int(result["removed"].(float64)) != 1 {
		t.Errorf("expected 1 removed, got %v", result["removed"])
	}
}

// TestHTTPServer_SetSUnion 测试 Set SUNION
func TestHTTPServer_SetSUnion(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.setCache.SAdd("set1", 0, "a", "b")
	hs.setCache.SAdd("set2", 0, "b", "c")

	req := httptest.NewRequest(http.MethodGet, "/cache/set/sunion?key=set1&key=set2", nil)
	w := httptest.NewRecorder()
	hs.setHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if int(result["count"].(float64)) != 3 {
		t.Errorf("expected union count 3, got %v", result["count"])
	}
}

// TestHTTPServer_SetSInter 测试 Set SINTER
func TestHTTPServer_SetSInter(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.setCache.SAdd("set1", 0, "a", "b")
	hs.setCache.SAdd("set2", 0, "b", "c")

	req := httptest.NewRequest(http.MethodGet, "/cache/set/sinter?key=set1&key=set2", nil)
	w := httptest.NewRecorder()
	hs.setHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if int(result["count"].(float64)) != 1 {
		t.Errorf("expected intersection count 1, got %v", result["count"])
	}
}

// TestHTTPServer_SetSDiff 测试 Set SDIFF
func TestHTTPServer_SetSDiff(t *testing.T) {
	hs := NewHTTPServer(HTTPServerConfig{Port: 0})

	hs.setCache.SAdd("set1", 0, "a", "b")
	hs.setCache.SAdd("set2", 0, "b", "c")

	req := httptest.NewRequest(http.MethodGet, "/cache/set/sdiff?key=set1&key=set2", nil)
	w := httptest.NewRecorder()
	hs.setHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}

	var result map[string]any
	json.NewDecoder(w.Result().Body).Decode(&result)
	if int(result["count"].(float64)) != 1 {
		t.Errorf("expected diff count 1, got %v", result["count"])
	}
}
