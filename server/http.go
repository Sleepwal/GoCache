package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"GoCache/cache"
)

// HTTPServer HTTP 缓存服务器
type HTTPServer struct {
	cache      *cache.MemoryCache
	stringCache *cache.StringCache
	listCache  *cache.ListCache
	hashCache  *cache.HashCache
	setCache   *cache.SetCache
	server     *http.Server
	startTime  time.Time
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

	memCache := cache.New()
	return &HTTPServer{
		cache:       memCache,
		stringCache: cache.NewStringCache(memCache),
		listCache:   cache.NewListCacheWithMemory(memCache),
		hashCache:   cache.NewHashCacheWithMemory(memCache),
		setCache:    cache.NewSetCacheWithMemory(memCache),
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
		cache:       c,
		stringCache: cache.NewStringCache(c),
		listCache:   cache.NewListCacheWithMemory(c),
		hashCache:   cache.NewHashCacheWithMemory(c),
		setCache:    cache.NewSetCacheWithMemory(c),
		server: &http.Server{
			Addr: fmt.Sprintf(":%d", cfg.Port),
		},
	}
}

// Start 启动 HTTP 服务器
func (hs *HTTPServer) Start() error {
	hs.startTime = time.Now()
	hs.setupRoutes()
	return hs.server.ListenAndServe()
}

// StartAsync 异步启动 HTTP 服务器
func (hs *HTTPServer) StartAsync() <-chan error {
	errCh := make(chan error, 1)
	go func() {
		hs.startTime = time.Now()
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

	// Basic cache endpoints
	mux.HandleFunc("/cache/", hs.corsMiddleware(hs.cacheHandler))
	mux.HandleFunc("/cache/keys", hs.corsMiddleware(hs.keysHandler))
	mux.HandleFunc("/cache/stats", hs.corsMiddleware(hs.statsHandler))
	mux.HandleFunc("/cache/clear", hs.corsMiddleware(hs.clearHandler))

	// String endpoints
	mux.HandleFunc("/cache/string/", hs.corsMiddleware(hs.stringHandler))

	// List endpoints
	mux.HandleFunc("/cache/list/", hs.corsMiddleware(hs.listHandler))

	// Hash endpoints
	mux.HandleFunc("/cache/hash/", hs.corsMiddleware(hs.hashHandler))

	// Set endpoints
	mux.HandleFunc("/cache/set/", hs.corsMiddleware(hs.setHandler))

	// Health endpoint
	mux.HandleFunc("/health", hs.corsMiddleware(hs.healthHandler))

	hs.server.Handler = hs.loggingMiddleware(mux)
}

// cacheHandler 处理 /cache/{key} 请求
func (hs *HTTPServer) cacheHandler(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/cache/")
	if key == "" || key == "keys" || key == "stats" || key == "clear" {
		return
	}

	switch r.Method {
	case http.MethodGet:
		// 支持 GET /cache/{key}?action=getdel
		action := r.URL.Query().Get("action")
		if action == "getdel" {
			hs.handleGetDel(w, key)
		} else {
			hs.handleGet(w, key)
		}
	case http.MethodPost, http.MethodPut:
		hs.handleSet(w, r, key)
	case http.MethodDelete:
		hs.handleDelete(w, key)
	default:
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleGetDel 处理 GETDEL 请求
func (hs *HTTPServer) handleGetDel(w http.ResponseWriter, key string) {
	value, found := hs.cache.GetDel(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"value": value,
	})
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

// healthHandler 处理 /health 请求
func (hs *HTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	uptime := time.Since(hs.startTime)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"uptime":       uptime.String(),
		"cache_count":  hs.cache.Count(),
		"list_count":   hs.listCache.Count(),
		"hash_count":   hs.hashCache.Count(),
		"set_count":    hs.setCache.Count(),
	})
}

// ===================== String Handlers =====================

// stringHandler 处理 /cache/string/{key}/{action} 请求
func (hs *HTTPServer) stringHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/cache/string/")
	if path == "" {
		hs.sendError(w, http.StatusBadRequest, "key is required")
		return
	}

	parts := strings.SplitN(path, "/", 2)
	key := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "append":
		hs.handleStringAppend(w, r, key)
	case "incr":
		hs.handleStringIncr(w, r, key)
	case "decr":
		hs.handleStringDecr(w, r, key)
	case "range":
		hs.handleStringRange(w, r, key)
	case "setrange":
		hs.handleStringSetRange(w, r, key)
	case "strlen":
		hs.handleStringStrLen(w, r, key)
	case "getset":
		hs.handleStringGetSet(w, r, key)
	default:
		// Default to GET/SET
		switch r.Method {
		case http.MethodGet:
			hs.handleStringGet(w, key)
		case http.MethodPost, http.MethodPut:
			hs.handleStringSet(w, r, key)
		default:
			hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

func (hs *HTTPServer) handleStringGet(w http.ResponseWriter, key string) {
	value, found := hs.stringCache.Get(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"value": value,
	})
}

func (hs *HTTPServer) handleStringSet(w http.ResponseWriter, r *http.Request, key string) {
	var req struct {
		Value string `json:"value"`
		TTL   string `json:"ttl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var ttl time.Duration
	if req.TTL != "" {
		ttl, _ = time.ParseDuration(req.TTL)
	}

	hs.stringCache.Set(key, req.Value, ttl)
	hs.sendJSON(w, http.StatusCreated, map[string]any{
		"key":     key,
		"message": "string set successfully",
	})
}

func (hs *HTTPServer) handleStringAppend(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	length := hs.stringCache.Append(key, req.Value)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"length": length,
	})
}

func (hs *HTTPServer) handleStringIncr(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	n := int64(1)
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		var err error
		n, err = strconv.ParseInt(nStr, 10, 64)
		if err != nil {
			hs.sendError(w, http.StatusBadRequest, "invalid n parameter")
			return
		}
	}

	result, err := hs.stringCache.IncrBy(key, n)
	if err != nil {
		hs.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"value": result,
	})
}

func (hs *HTTPServer) handleStringDecr(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	n := int64(1)
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		var err error
		n, err = strconv.ParseInt(nStr, 10, 64)
		if err != nil {
			hs.sendError(w, http.StatusBadRequest, "invalid n parameter")
			return
		}
	}

	result, err := hs.stringCache.DecrBy(key, n)
	if err != nil {
		hs.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"value": result,
	})
}

func (hs *HTTPServer) handleStringRange(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	start, _ := strconv.Atoi(r.URL.Query().Get("start"))
	end, _ := strconv.Atoi(r.URL.Query().Get("end"))

	value, found := hs.stringCache.GetRange(key, start, end)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"value": value,
	})
}

func (hs *HTTPServer) handleStringSetRange(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Offset int    `json:"offset"`
		Value  string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	length := hs.stringCache.SetRange(key, req.Offset, req.Value)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"length": length,
	})
}

func (hs *HTTPServer) handleStringStrLen(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	length, found := hs.stringCache.StrLen(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"length": length,
	})
}

func (hs *HTTPServer) handleStringGetSet(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	oldValue, found := hs.stringCache.GetSet(key, req.Value)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":     key,
		"value":   oldValue,
		"existed": found,
	})
}

// ===================== List Handlers =====================

// listHandler 处理 /cache/list/{key}/{action} 请求
func (hs *HTTPServer) listHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/cache/list/")
	if path == "" {
		hs.sendError(w, http.StatusBadRequest, "key is required")
		return
	}

	parts := strings.SplitN(path, "/", 2)
	key := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "lpush":
		hs.handleListLPush(w, r, key)
	case "rpush":
		hs.handleListRPush(w, r, key)
	case "lpop":
		hs.handleListLPop(w, r, key)
	case "rpop":
		hs.handleListRPop(w, r, key)
	case "range":
		hs.handleListRange(w, r, key)
	case "index":
		hs.handleListIndex(w, r, key)
	case "len":
		hs.handleListLLen(w, r, key)
	case "trim":
		hs.handleListLTrim(w, r, key)
	case "rem":
		hs.handleListLRem(w, r, key)
	default:
		hs.sendError(w, http.StatusBadRequest, "unknown action")
	}
}

func (hs *HTTPServer) handleListLPush(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Values []any  `json:"values"`
		TTL    string `json:"ttl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var ttl time.Duration
	if req.TTL != "" {
		ttl, _ = time.ParseDuration(req.TTL)
	}

	length := hs.listCache.LPush(key, ttl, req.Values...)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"length": length,
	})
}

func (hs *HTTPServer) handleListRPush(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Values []any  `json:"values"`
		TTL    string `json:"ttl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var ttl time.Duration
	if req.TTL != "" {
		ttl, _ = time.ParseDuration(req.TTL)
	}

	length := hs.listCache.RPush(key, ttl, req.Values...)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"length": length,
	})
}

func (hs *HTTPServer) handleListLPop(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	value, found := hs.listCache.LPop(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found or empty")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"value": value,
	})
}

func (hs *HTTPServer) handleListRPop(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	value, found := hs.listCache.RPop(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found or empty")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"value": value,
	})
}

func (hs *HTTPServer) handleListRange(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	start, _ := strconv.Atoi(r.URL.Query().Get("start"))
	stop, _ := strconv.Atoi(r.URL.Query().Get("stop"))

	values, found := hs.listCache.LRange(key, start, stop)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"values": values,
	})
}

func (hs *HTTPServer) handleListIndex(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	index, _ := strconv.Atoi(r.URL.Query().Get("index"))

	value, found := hs.listCache.LIndex(key, index)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found or index out of range")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"value": value,
	})
}

func (hs *HTTPServer) handleListLLen(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	length, found := hs.listCache.LLen(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"length": length,
	})
}

func (hs *HTTPServer) handleListLTrim(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	start, _ := strconv.Atoi(r.URL.Query().Get("start"))
	stop, _ := strconv.Atoi(r.URL.Query().Get("stop"))

	hs.listCache.LTrim(key, start, stop)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":     key,
		"message": "list trimmed successfully",
	})
}

func (hs *HTTPServer) handleListLRem(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Count int `json:"count"`
		Value any `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	removed := hs.listCache.LRem(key, req.Count, req.Value)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":     key,
		"removed": removed,
	})
}

// ===================== Hash Handlers =====================

// hashHandler 处理 /cache/hash/{key}/{action} 请求
func (hs *HTTPServer) hashHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/cache/hash/")
	if path == "" {
		hs.sendError(w, http.StatusBadRequest, "key is required")
		return
	}

	parts := strings.SplitN(path, "/", 2)
	key := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "hset":
		hs.handleHashHSet(w, r, key)
	case "hget":
		hs.handleHashHGet(w, r, key)
	case "hgetall":
		hs.handleHashHGetAll(w, r, key)
	case "hdel":
		hs.handleHashHDel(w, r, key)
	case "hexists":
		hs.handleHashHExists(w, r, key)
	case "hlen":
		hs.handleHashHLen(w, r, key)
	case "hkeys":
		hs.handleHashHKeys(w, r, key)
	case "hvals":
		hs.handleHashHVals(w, r, key)
	case "hincrby":
		hs.handleHashHIncrBy(w, r, key)
	default:
		hs.sendError(w, http.StatusBadRequest, "unknown action")
	}
}

func (hs *HTTPServer) handleHashHSet(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Fields map[string]any `json:"fields"`
		TTL    string         `json:"ttl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var ttl time.Duration
	if req.TTL != "" {
		ttl, _ = time.ParseDuration(req.TTL)
	}

	count := hs.hashCache.HSet(key, ttl, req.Fields)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":          key,
		"fields_added": count,
	})
}

func (hs *HTTPServer) handleHashHGet(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	field := r.URL.Query().Get("field")
	if field == "" {
		hs.sendError(w, http.StatusBadRequest, "field parameter is required")
		return
	}

	value, found := hs.hashCache.HGet(key, field)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key or field not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"field": field,
		"value": value,
	})
}

func (hs *HTTPServer) handleHashHGetAll(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	fields, found := hs.hashCache.HGetAll(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"fields": fields,
	})
}

func (hs *HTTPServer) handleHashHDel(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Fields []string `json:"fields"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	deleted := hs.hashCache.HDel(key, req.Fields...)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":           key,
		"fields_deleted": deleted,
	})
}

func (hs *HTTPServer) handleHashHExists(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	field := r.URL.Query().Get("field")
	if field == "" {
		hs.sendError(w, http.StatusBadRequest, "field parameter is required")
		return
	}

	exists := hs.hashCache.HExists(key, field)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"field":  field,
		"exists": exists,
	})
}

func (hs *HTTPServer) handleHashHLen(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	length, found := hs.hashCache.HLen(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"length": length,
	})
}

func (hs *HTTPServer) handleHashHKeys(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	keys, found := hs.hashCache.HKeys(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":  key,
		"keys": keys,
	})
}

func (hs *HTTPServer) handleHashHVals(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	vals, found := hs.hashCache.HVals(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":  key,
		"vals": vals,
	})
}

func (hs *HTTPServer) handleHashHIncrBy(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	field := r.URL.Query().Get("field")
	if field == "" {
		hs.sendError(w, http.StatusBadRequest, "field parameter is required")
		return
	}

	n, _ := strconv.ParseInt(r.URL.Query().Get("n"), 10, 64)

	result, err := hs.hashCache.HIncrBy(key, field, 0, n)
	if err != nil {
		hs.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"field": field,
		"value": result,
	})
}

// ===================== Set Handlers =====================

// setHandler 处理 /cache/set/{key}/{action} 请求
func (hs *HTTPServer) setHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/cache/set/")
	if path == "" {
		return
	}

	// Handle multi-key operations that don't require a key
	if path == "sunion" || path == "sinter" || path == "sdiff" {
		switch path {
		case "sunion":
			hs.handleSetSUnion(w, r)
		case "sinter":
			hs.handleSetSInter(w, r)
		case "sdiff":
			hs.handleSetSDiff(w, r)
		}
		return
	}

	parts := strings.SplitN(path, "/", 2)
	key := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "sadd":
		hs.handleSetSAdd(w, r, key)
	case "srem":
		hs.handleSetSRem(w, r, key)
	case "sismember":
		hs.handleSetSIsMember(w, r, key)
	case "scard":
		hs.handleSetSCard(w, r, key)
	case "smembers":
		hs.handleSetSMembers(w, r, key)
	case "spop":
		hs.handleSetSPop(w, r, key)
	default:
		hs.sendError(w, http.StatusBadRequest, "unknown action")
	}
}

func (hs *HTTPServer) handleSetSAdd(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Members []any `json:"members"`
		TTL     string `json:"ttl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var ttl time.Duration
	if req.TTL != "" {
		ttl, _ = time.ParseDuration(req.TTL)
	}

	added := hs.setCache.SAdd(key, ttl, req.Members...)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":     key,
		"added":   added,
	})
}

func (hs *HTTPServer) handleSetSRem(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Members []any `json:"members"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	removed := hs.setCache.SRem(key, req.Members...)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":     key,
		"removed": removed,
	})
}

func (hs *HTTPServer) handleSetSIsMember(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	member := r.URL.Query().Get("member")
	if member == "" {
		hs.sendError(w, http.StatusBadRequest, "member parameter is required")
		return
	}

	exists := hs.setCache.SIsMember(key, member)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"member": member,
		"exists": exists,
	})
}

func (hs *HTTPServer) handleSetSCard(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	card, found := hs.setCache.SCard(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":  key,
		"card": card,
	})
}

func (hs *HTTPServer) handleSetSMembers(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	members, found := hs.setCache.SMembers(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":     key,
		"members": members,
	})
}

func (hs *HTTPServer) handleSetSPop(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodPost {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	member, found := hs.setCache.SPop(key)
	if !found {
		hs.sendError(w, http.StatusNotFound, "key not found or empty")
		return
	}

	hs.sendJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"member": member,
	})
}

func (hs *HTTPServer) handleSetSUnion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	keys := r.URL.Query()["key"]
	if len(keys) == 0 {
		hs.sendError(w, http.StatusBadRequest, "key parameter is required")
		return
	}

	result := hs.setCache.SUnion(keys...)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"keys":   keys,
		"union":  result,
		"count":  len(result),
	})
}

func (hs *HTTPServer) handleSetSInter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	keys := r.URL.Query()["key"]
	if len(keys) == 0 {
		hs.sendError(w, http.StatusBadRequest, "key parameter is required")
		return
	}

	result := hs.setCache.SInter(keys...)
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"keys":      keys,
		"intersection": result,
		"count":     len(result),
	})
}

func (hs *HTTPServer) handleSetSDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		hs.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	keys := r.URL.Query()["key"]
	if len(keys) < 2 {
		hs.sendError(w, http.StatusBadRequest, "at least 2 key parameters are required")
		return
	}

	result := hs.setCache.SDiff(keys[0], keys[1])
	hs.sendJSON(w, http.StatusOK, map[string]any{
		"keys": keys,
		"diff": result,
		"count": len(result),
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
		
		// Wrap response writer to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(rw, r)
		
		duration := time.Since(start)
		log.Printf("[%s] %s %s %d %s", 
			r.Method, 
			r.URL.Path, 
			http.StatusText(rw.statusCode), 
			rw.statusCode, 
			duration)
	})
}

// responseWriter 包装 http.ResponseWriter 以获取状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
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
