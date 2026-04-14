package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"GoCache/cache"
)

// HTTPServer HTTP 缓存服务器
type HTTPServer struct {
	cache  *cache.MemoryCache
	server *http.Server
}

// HTTPServerConfig 服务器配置
type HTTPServerConfig struct {
	Port int `json:"port"`
}

// NewHTTPServer 创建 HTTP 缓存服务器
func NewHTTPServer(cfg HTTPServerConfig) *HTTPServer {
	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	return &HTTPServer{
		cache: cache.New(),
		server: &http.Server{
			Addr: fmt.Sprintf(":%d", cfg.Port),
		},
	}
}

// NewHTTPServerWithCache 创建带自定义缓存的 HTTP 服务器
func NewHTTPServerWithCache(cfg HTTPServerConfig, c *cache.MemoryCache) *HTTPServer {
	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	return &HTTPServer{
		cache:  c,
		server: &http.Server{
			Addr: fmt.Sprintf(":%d", cfg.Port),
		},
	}
}

// Start 启动 HTTP 服务器
func (hs *HTTPServer) Start() error {
	hs.setupRoutes()
	return hs.server.ListenAndServe()
}

// StartAsync 异步启动 HTTP 服务器
func (hs *HTTPServer) StartAsync() <-chan error {
	errCh := make(chan error, 1)
	go func() {
		hs.setupRoutes()
		errCh <- hs.server.ListenAndServe()
	}()
	return errCh
}

// Stop 停止 HTTP 服务器
func (hs *HTTPServer) Stop() error {
	return hs.server.Close()
}

// setupRoutes 设置路由
func (hs *HTTPServer) setupRoutes() {
	mux := http.NewServeMux()

	// CORS 中间件
	mux.HandleFunc("/cache/", hs.corsMiddleware(hs.cacheHandler))
	mux.HandleFunc("/cache/keys", hs.corsMiddleware(hs.keysHandler))
	mux.HandleFunc("/cache/stats", hs.corsMiddleware(hs.statsHandler))
	mux.HandleFunc("/cache/clear", hs.corsMiddleware(hs.clearHandler))

	hs.server.Handler = hs.loggingMiddleware(mux)
}

// cacheHandler 处理 /cache/{key} 请求
func (hs *HTTPServer) cacheHandler(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/cache/")
	if key == "" {
		hs.sendError(w, http.StatusBadRequest, "key is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		hs.handleGet(w, key)
	case http.MethodPost, http.MethodPut:
		hs.handleSet(w, r, key)
	case http.MethodDelete:
		hs.handleDelete(w, key)
	default:
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleGet 处理 GET 请求
func (hs *HTTPServer) handleGet(w http.ResponseWriter, key string) {
	value, found := hs.cache.Get(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"value": value,
	})
}

// handleSet 处理 POST/PUT 请求
func (hs *HTTPServer) handleSet(w http.ResponseWriter, r *http.Request, key string) {
	var req struct {
		Value any    `json:"value"`
		TTL   string `json:"ttl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 解析 TTL
	var ttl time.Duration
	if req.TTL != "" {
		duration, err := time.ParseDuration(req.TTL)
		if err != nil {
			hs.sendError(w, http.StatusBadRequest, "invalid ttl format")
			return
		}
		ttl = duration
	}

	// 也支持 query string 中的 ttl
	if ttl == 0 {
		ttlStr := r.URL.Query().Get("ttl")
		if ttlStr != "" {
			duration, err := time.ParseDuration(ttlStr)
			if err != nil {
				hs.sendError(w, http.StatusBadRequest, "invalid ttl format")
				return
			}
			ttl = duration
		}
	}

	hs.cache.Set(key, req.Value, ttl)

	hs.sendJSON(w, http.StatusCreated, map[string]any{
		"key":     key,
		"message": "cache set successfully",
		"ttl":     ttl.String(),
	})
}

// handleDelete 处理 DELETE 请求
func (hs *HTTPServer) handleDelete(w http.ResponseWriter, key string) {
	deleted := hs.cache.Delete(key)
	if !deleted {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":     key,
		"message": "cache deleted successfully",
	})
}

// keysHandler 处理 /cache/keys 请求
func (hs *HTTPServer) keysHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	keys := hs.cache.Keys()
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"keys":  keys,
		"count": len(keys),
	})
}

// statsHandler 处理 /cache/stats 请求
func (hs *HTTPServer) statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	snapshot := hs.cache.Stats.GetSnapshot()
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"hits":         snapshot.Hits,
		"misses":       snapshot.Misses,
		"sets":         snapshot.Sets,
		"deletes":      snapshot.Deletes,
		"expired":      snapshot.ExpiredCount,
		"ttl_hits":     snapshot.TTLHits,
		"ttl_misses":   snapshot.TTLMisses,
		"hit_rate":     fmt.Sprintf("%.2f%%", snapshot.HitRate),
		"total_ops":    snapshot.Hits + snapshot.Misses + snapshot.Sets + snapshot.Deletes,
	})
}

// clearHandler 处理 /cache/clear 请求
func (hs *HTTPServer) clearHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	hs.cache.Clear()
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"message": "cache cleared successfully",
	})
}

// sendJSON 发送 JSON 响应
func (hs *HTTPServer) sendJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// sendError 发送错误响应
func (hs *HTTPServer) sendError(w http.ResponseWriter, status int, message string) {
	hs.sendJSON(w, status, map[string]any{
		"error":   http.StatusText(status),
		"message": message,
		"status":  status,
	})
}

// corsMiddleware CORS 中间件
func (hs *HTTPServer) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// loggingMiddleware 请求日志中间件
func (hs *HTTPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		// 可以在这里添加日志逻辑
		_ = start
	})
}

// GetCache 获取内部缓存实例（用于测试）
func (hs *HTTPServer) GetCache() *cache.MemoryCache {
	return hs.cache
}

// SetCache 设置缓存实例（用于测试）
func (hs *HTTPServer) SetCache(c *cache.MemoryCache) {
	hs.cache = c
}

// Helper function to parse integer from string
func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return val
}
