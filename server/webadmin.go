package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sort"
	"time"

	"GoCache/cache"
)

type WebAdminServer struct {
	cache       *cache.MemoryCache
	metrics     *cache.MetricsCollector
	appConfig   *cache.Config
	server      *http.Server
	startTime   time.Time
	staticFiles embed.FS
}

func NewWebAdminServer(port int, c *cache.MemoryCache) *WebAdminServer {
	return &WebAdminServer{
		cache: c,
		server: &http.Server{
			Addr: fmt.Sprintf(":%d", port),
		},
		staticFiles: embed.FS{},
	}
}

func (was *WebAdminServer) SetMetrics(m *cache.MetricsCollector) {
	was.metrics = m
}

func (was *WebAdminServer) SetConfig(cfg *cache.Config) {
	was.appConfig = cfg
}

func (was *WebAdminServer) Start() error {
	was.startTime = time.Now()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/dashboard", was.corsMiddleware(was.dashboardHandler))
	mux.HandleFunc("/api/keys", was.corsMiddleware(was.keysHandler))
	mux.HandleFunc("/api/key/", was.corsMiddleware(was.keyHandler))
	mux.HandleFunc("/api/stats", was.corsMiddleware(was.statsHandler))
	mux.HandleFunc("/api/config", was.corsMiddleware(was.configHandler))

	sub, err := fs.Sub(was.staticFiles, "static")
	if err == nil {
		mux.Handle("/", http.FileServer(http.FS(sub)))
	} else {
		mux.HandleFunc("/", was.indexHandler)
	}

	was.server.Handler = was.loggingMiddleware(mux)
	return was.server.ListenAndServe()
}

func (was *WebAdminServer) StartAsync() <-chan error {
	errCh := make(chan error, 1)
	go func() {
		if err := was.Start(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()
	return errCh
}

func (was *WebAdminServer) Stop() error {
	return was.server.Close()
}

func (was *WebAdminServer) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func (was *WebAdminServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func (was *WebAdminServer) sendJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (was *WebAdminServer) sendError(w http.ResponseWriter, status int, msg string) {
	was.sendJSON(w, status, map[string]string{"error": msg})
}

func (was *WebAdminServer) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		was.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	keyCount := was.cache.Count()
	usedMemory := was.cache.UsedMemory()

	var hitRate float64
	var totalOps int64
	if was.metrics != nil {
		snapshot := was.metrics.GetSnapshot(was.cache)
		hitRate = snapshot.HitRate
		totalOps = snapshot.TotalCommands
	}

	maxMemory := 0
	if was.appConfig != nil {
		maxMemory = int(was.appConfig.GetMaxMemory())
	}

	was.sendJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"uptime":      time.Since(was.startTime).String(),
		"version":     "2.0.0",
		"key_count":   keyCount,
		"used_memory": usedMemory,
		"max_memory":  maxMemory,
		"hit_rate":    fmt.Sprintf("%.2f%%", hitRate),
		"total_ops":   totalOps,
	})
}

func (was *WebAdminServer) keysHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		keys := was.cache.Keys()
		sort.Strings(keys)

		type keyInfo struct {
			Key  string `json:"key"`
			Type string `json:"type"`
			TTL  string `json:"ttl"`
		}

		result := make([]keyInfo, 0, len(keys))
		for _, key := range keys {
			ttl := "forever"
			typ := was.cache.Type(key)
			result = append(result, keyInfo{
				Key:  key,
				Type: typ,
				TTL:  ttl,
			})
		}

		was.sendJSON(w, http.StatusOK, result)

	case http.MethodDelete:
		was.cache.Clear()
		was.sendJSON(w, http.StatusOK, map[string]string{"status": "cleared"})

	default:
		was.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (was *WebAdminServer) keyHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/api/key/"):]

	switch r.Method {
	case http.MethodGet:
		val, found := was.cache.Get(key)
		if !found {
			was.sendError(w, http.StatusNotFound, "key not found")
			return
		}
		was.sendJSON(w, http.StatusOK, map[string]any{
			"key":   key,
			"value": val,
			"type":  was.cache.Type(key),
		})

	case http.MethodDelete:
		if was.cache.Delete(key) {
			was.sendJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		} else {
			was.sendError(w, http.StatusNotFound, "key not found")
		}

	case http.MethodPost:
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			was.sendError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		value, ok := body["value"]
		if !ok {
			was.sendError(w, http.StatusBadRequest, "missing value")
			return
		}

		ttlStr, hasTTL := body["ttl"]
		ttl := time.Duration(0)
		if hasTTL {
			if ttlFloat, ok := ttlStr.(float64); ok {
				ttl = time.Duration(ttlFloat) * time.Second
			}
		}

		was.cache.Set(key, value, ttl)
		was.sendJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		was.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (was *WebAdminServer) statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		was.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if was.metrics == nil {
		was.sendError(w, http.StatusNotFound, "metrics not enabled")
		return
	}

	snapshot := was.metrics.GetSnapshot(was.cache)
	was.sendJSON(w, http.StatusOK, snapshot)
}

func (was *WebAdminServer) configHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		was.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if was.appConfig == nil {
		was.sendError(w, http.StatusNotFound, "config not available")
		return
	}

	was.sendJSON(w, http.StatusOK, was.appConfig)
}

func (was *WebAdminServer) indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>GoCache Admin</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#0f172a;color:#e2e8f0;min-height:100vh}
.header{background:linear-gradient(135deg,#1e293b,#334155);padding:20px 32px;border-bottom:1px solid #475569;display:flex;align-items:center;justify-content:space-between}
.header h1{font-size:24px;font-weight:700;color:#38bdf8}
.header .version{font-size:12px;color:#94a3b8;background:#1e293b;padding:4px 10px;border-radius:12px}
.container{max-width:1400px;margin:0 auto;padding:24px}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:16px;margin-bottom:24px}
.card{background:#1e293b;border:1px solid #334155;border-radius:12px;padding:20px}
.card .label{font-size:12px;color:#94a3b8;text-transform:uppercase;letter-spacing:1px;margin-bottom:8px}
.card .value{font-size:28px;font-weight:700;color:#f8fafc}
.card .value.accent{color:#38bdf8}
.card .value.green{color:#4ade80}
.card .value.amber{color:#fbbf24}
.panel{background:#1e293b;border:1px solid #334155;border-radius:12px;padding:20px;margin-bottom:24px}
.panel h2{font-size:16px;font-weight:600;margin-bottom:16px;color:#38bdf8}
table{width:100%;border-collapse:collapse}
th{text-align:left;font-size:12px;color:#94a3b8;text-transform:uppercase;letter-spacing:1px;padding:8px 12px;border-bottom:1px solid #334155}
td{padding:8px 12px;border-bottom:1px solid #1e293b;font-size:14px}
.badge{display:inline-block;padding:2px 8px;border-radius:4px;font-size:11px;font-weight:600}
.badge.string{background:#1e3a5f;color:#38bdf8}
.badge.list{background:#1a3a2a;color:#4ade80}
.badge.hash{background:#3a2a1a;color:#fbbf24}
.badge.set{background:#2a1a3a;color:#c084fc}
.badge.zset{background:#3a1a2a;color:#fb7185}
.badge.bitmap{background:#1a2a3a;color:#22d3ee}
.badge.hyperloglog{background:#2a3a1a;color:#a3e635}
.badge.geo{background:#3a3a1a;color:#facc15}
.actions{display:flex;gap:8px;margin-bottom:16px}
.btn{padding:8px 16px;border:none;border-radius:6px;cursor:pointer;font-size:13px;font-weight:600;transition:all .2s}
.btn-primary{background:#0ea5e9;color:#fff}
.btn-primary:hover{background:#0284c7}
.btn-danger{background:#ef4444;color:#fff}
.btn-danger:hover{background:#dc2626}
.btn-secondary{background:#334155;color:#e2e8f0}
.btn-secondary:hover{background:#475569}
.search{padding:8px 16px;background:#0f172a;border:1px solid #334155;border-radius:6px;color:#e2e8f0;font-size:14px;width:300px}
.search:focus{outline:none;border-color:#38bdf8}
.refresh-info{font-size:12px;color:#64748b;margin-top:8px}
</style>
</head>
<body>
<div class="header">
<h1>GoCache Admin</h1>
<span class="version">v2.0.0</span>
</div>
<div class="container">
<div class="cards" id="stats-cards"></div>
<div class="panel">
<h2>Keys</h2>
<div class="actions">
<input type="text" class="search" id="search" placeholder="Search keys...">
<button class="btn btn-primary" onclick="refreshKeys()">Refresh</button>
<button class="btn btn-danger" onclick="clearAll()">Clear All</button>
</div>
<table>
<thead><tr><th>Key</th><th>Type</th><th>TTL</th><th>Action</th></tr></thead>
<tbody id="keys-body"></tbody>
</table>
<div class="refresh-info">Auto-refresh every 5 seconds</div>
</div>
</div>
<script>
function h(tag,attrs,children){
  var el=document.createElement(tag);
  for(var k in attrs){el[k]=attrs[k]}
  if(typeof children==='string'){el.innerHTML=children}
  else if(Array.isArray(children)){children.forEach(function(c){el.appendChild(c)})}
  return el;
}
async function loadDashboard(){
  try{
    var r=await fetch('/api/dashboard');
    var d=await r.json();
    var container=document.getElementById('stats-cards');
    container.innerHTML='';
    var cards=[
      {label:'Keys',value:d.key_count||0,cls:'accent'},
      {label:'Memory',value:formatBytes(d.used_memory||0),cls:''},
      {label:'Hit Rate',value:d.hit_rate||'0%',cls:'green'},
      {label:'Total Ops',value:(d.total_ops||0).toLocaleString(),cls:'amber'},
      {label:'Uptime',value:d.uptime||'-',cls:''}
    ];
    cards.forEach(function(c){
      var card=h('div',{className:'card'},'');
      card.innerHTML='<div class="label">'+c.label+'</div><div class="value '+c.cls+'">'+c.value+'</div>';
      container.appendChild(card);
    });
  }catch(e){console.error(e)}
}
async function refreshKeys(){
  try{
    var r=await fetch('/api/keys');
    var keys=await r.json();
    var search=document.getElementById('search').value.toLowerCase();
    var filtered=search?keys.filter(function(k){return k.key.toLowerCase().includes(search)}):keys;
    var tbody=document.getElementById('keys-body');
    tbody.innerHTML='';
    filtered.forEach(function(k){
      var tr=document.createElement('tr');
      tr.innerHTML='<td>'+escapeHtml(k.key)+'</td>'+
        '<td><span class="badge '+k.type+'">'+k.type+'</span></td>'+
        '<td>'+k.ttl+'</td>'+
        '<td><button class="btn btn-danger" style="padding:4px 8px;font-size:11px" onclick="deleteKey(\''+escapeHtml(k.key).replace(/'/g,"\\'")+'\')">DEL</button></td>';
      tbody.appendChild(tr);
    });
  }catch(e){console.error(e)}
}
async function deleteKey(key){
  if(!confirm('Delete key: '+key+'?'))return;
  await fetch('/api/key/'+encodeURIComponent(key),{method:'DELETE'});
  refreshKeys();loadDashboard();
}
async function clearAll(){
  if(!confirm('Clear ALL keys?'))return;
  await fetch('/api/keys',{method:'DELETE'});
  refreshKeys();loadDashboard();
}
function formatBytes(b){
  if(b===0)return'0 B';
  var k=1024;var s=['B','KB','MB','GB'];
  var i=Math.floor(Math.log(b)/Math.log(k));
  return parseFloat((b/Math.pow(k,i)).toFixed(1))+' '+s[i];
}
function escapeHtml(s){
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}
document.getElementById('search').addEventListener('input',refreshKeys);
loadDashboard();refreshKeys();
setInterval(function(){loadDashboard();refreshKeys()},5000);
</script>
</body>
</html>`
