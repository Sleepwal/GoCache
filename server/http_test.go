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
