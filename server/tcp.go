package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"GoCache/cache"
	"GoCache/logger"
	"GoCache/resp"
)

type CommandHandler func(c *client, args []string) error

type TCPServer struct {
	cache              *cache.MemoryCache
	stringCache        *cache.StringCache
	listCache          *cache.ListCache
	hashCache          *cache.HashCache
	setCache           *cache.SetCache
	sortedSetCache     *cache.SortedSetCache
	transactionManager *cache.TransactionManager
	appConfig          *cache.Config
	metrics            *cache.MetricsCollector
	listener           net.Listener
	tcpConfig          TCPServerConfig
	commands           map[string]CommandHandler
	clients            map[net.Conn]*client
	mu                 sync.RWMutex
	running            bool
	startTime          time.Time
}

type TCPServerConfig struct {
	Port int `json:"port"`
}

type client struct {
	conn   net.Conn
	writer *resp.Writer
	reader *resp.Reader
	db     int
	server *TCPServer
	tx     *cache.Transaction
	authed bool
}

func NewTCPServer(cfg TCPServerConfig) *TCPServer {
	if cfg.Port == 0 {
		cfg.Port = 6379
	}

	mc := cache.New()
	ts := &TCPServer{
		cache:              mc,
		stringCache:        cache.NewStringCache(mc),
		listCache:          cache.NewListCacheWithMemory(mc),
		hashCache:          cache.NewHashCacheWithMemory(mc),
		setCache:           cache.NewSetCacheWithMemory(mc),
		sortedSetCache:     cache.NewSortedSetCacheWithMemory(mc),
		transactionManager: cache.NewTransactionManager(mc),
		tcpConfig:          cfg,
		clients:            make(map[net.Conn]*client),
	}

	ts.registerCommands()
	return ts
}

func NewTCPServerWithCache(cfg TCPServerConfig, c *cache.MemoryCache) *TCPServer {
	if cfg.Port == 0 {
		cfg.Port = 6379
	}

	ts := &TCPServer{
		cache:              c,
		stringCache:        cache.NewStringCache(c),
		listCache:          cache.NewListCacheWithMemory(c),
		hashCache:          cache.NewHashCacheWithMemory(c),
		setCache:           cache.NewSetCacheWithMemory(c),
		sortedSetCache:     cache.NewSortedSetCacheWithMemory(c),
		transactionManager: cache.NewTransactionManager(c),
		tcpConfig:          cfg,
		clients:            make(map[net.Conn]*client),
	}

	ts.registerCommands()
	return ts
}

func (ts *TCPServer) registerCommands() {
	ts.commands = map[string]CommandHandler{
		// Connection
		"ping":     ts.cmdPing,
		"echo":     ts.cmdEcho,
		"select":   ts.cmdSelect,
		"quit":     ts.cmdQuit,
		"command":  ts.cmdCommand,
		"info":     ts.cmdInfo,
		"dbsize":   ts.cmdDBSize,
		"flushdb":  ts.cmdFlushDB,
		"flushall": ts.cmdFlushAll,
		"time":     ts.cmdTime,
		"lastsave": ts.cmdLastSave,
		// Key
		"set":     ts.cmdSet,
		"get":     ts.cmdGet,
		"getdel":  ts.cmdGetDel,
		"del":     ts.cmdDel,
		"exists":  ts.cmdExists,
		"keys":    ts.cmdKeys,
		"expire":  ts.cmdExpire,
		"ttl":     ts.cmdTTL,
		"pttl":    ts.cmdPTTL,
		"persist": ts.cmdPersist,
		"type":    ts.cmdType,
		"rename":  ts.cmdRename,
		"scan":    ts.cmdScan,
		// String
		"append":      ts.cmdAppend,
		"incr":        ts.cmdIncr,
		"decr":        ts.cmdDecr,
		"incrby":      ts.cmdIncrBy,
		"decrby":      ts.cmdDecrBy,
		"incrbyfloat": ts.cmdIncrByFloat,
		"strlen":      ts.cmdStrLen,
		"getrange":    ts.cmdGetRange,
		"setrange":    ts.cmdSetRange,
		"getset":      ts.cmdGetSet,
		"mget":        ts.cmdMGet,
		"mset":        ts.cmdMSet,
		"setnx":       ts.cmdSetNX,
		"setex":       ts.cmdSetEX,
		// List
		"lpush":  ts.cmdLPush,
		"rpush":  ts.cmdRPush,
		"lpop":   ts.cmdLPop,
		"rpop":   ts.cmdRPop,
		"lrange": ts.cmdLRange,
		"lindex": ts.cmdLIndex,
		"llen":   ts.cmdLLen,
		"ltrim":  ts.cmdLTrim,
		"lrem":   ts.cmdLRem,
		// Hash
		"hset":         ts.cmdHSet,
		"hget":         ts.cmdHGet,
		"hgetall":      ts.cmdHGetAll,
		"hdel":         ts.cmdHDel,
		"hexists":      ts.cmdHExists,
		"hlen":         ts.cmdHLen,
		"hkeys":        ts.cmdHKeys,
		"hvals":        ts.cmdHVals,
		"hsetnx":       ts.cmdHSetNX,
		"hincrby":      ts.cmdHIncrBy,
		"hincrbyfloat": ts.cmdHIncrByFloat,
		"hmset":        ts.cmdHMSet,
		"hmget":        ts.cmdHMGet,
		// Set
		"sadd":      ts.cmdSAdd,
		"srem":      ts.cmdSRem,
		"sismember": ts.cmdSIsMember,
		"scard":     ts.cmdSCard,
		"smembers":  ts.cmdSMembers,
		"spop":      ts.cmdSPop,
		"sunion":    ts.cmdSUnion,
		"sinter":    ts.cmdSInter,
		"sdiff":     ts.cmdSDiff,
		// Sorted Set
		"zadd":             ts.cmdZAdd,
		"zrem":             ts.cmdZRem,
		"zscore":           ts.cmdZScore,
		"zcard":            ts.cmdZCard,
		"zrank":            ts.cmdZRank,
		"zrevrank":         ts.cmdZRevRank,
		"zrange":           ts.cmdZRange,
		"zrevrange":        ts.cmdZRevRange,
		"zrangebyscore":    ts.cmdZRangeByScore,
		"zrevrangebyscore": ts.cmdZRevRangeByScore,
		"zcount":           ts.cmdZCount,
		"zincrby":          ts.cmdZIncrBy,
		"zpopmin":          ts.cmdZPopMin,
		"zpopmax":          ts.cmdZPopMax,
		"zremrangebyrank":  ts.cmdZRemRangeByRank,
		"zremrangebyscore": ts.cmdZRemRangeByScore,
		// Transaction
		"multi":   ts.cmdMulti,
		"exec":    ts.cmdExec,
		"discard": ts.cmdDiscard,
		"watch":   ts.cmdWatch,
		"unwatch": ts.cmdUnwatch,
		// Auth
		"auth": ts.cmdAuth,
	}
}

func (ts *TCPServer) Start() error {
	var ln net.Listener
	var err error

	addr := fmt.Sprintf(":%d", ts.tcpConfig.Port)

	if ts.appConfig != nil && ts.appConfig.IsTLSEnabled() {
		certFile := ts.appConfig.GetTLSCertFile()
		keyFile := ts.appConfig.GetTLSKeyFile()

		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificate: %w", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}

		ln, err = tls.Listen("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to listen with TLS on port %d: %w", ts.tcpConfig.Port, err)
		}

		logger.Info("RESP TCP server started with TLS", "port", ts.tcpConfig.Port)
	} else {
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to listen on port %d: %w", ts.tcpConfig.Port, err)
		}

		logger.Info("RESP TCP server started", "port", ts.tcpConfig.Port)
	}

	ts.listener = ln
	ts.running = true
	ts.startTime = time.Now()

	for ts.running {
		conn, err := ln.Accept()
		if err != nil {
			if !ts.running {
				return nil
			}
			logger.Error("failed to accept connection", "error", err)
			continue
		}

		c := &client{
			conn:   conn,
			writer: resp.NewWriter(conn),
			reader: resp.NewReader(conn),
			server: ts,
		}

		ts.mu.Lock()
		ts.clients[conn] = c
		ts.mu.Unlock()

		if ts.metrics != nil {
			ts.metrics.AddConnectedClient()
		}

		logger.Info("client connected", "remote", conn.RemoteAddr().String(), "clients", ts.ClientCount())

		go ts.handleConnection(c)
	}

	return nil
}

func (ts *TCPServer) StartAsync() <-chan error {
	errCh := make(chan error, 1)
	go func() {
		if err := ts.Start(); err != nil {
			errCh <- err
		}
	}()
	return errCh
}

func (ts *TCPServer) Stop() error {
	ts.running = false
	logger.Info("RESP TCP server stopping", "clients", ts.ClientCount())

	ts.mu.Lock()
	defer ts.mu.Unlock()

	for conn, c := range ts.clients {
		c.writer.WriteError("server shutting down")
		conn.Close()
	}
	ts.clients = make(map[net.Conn]*client)

	if ts.listener != nil {
		return ts.listener.Close()
	}

	return nil
}

func (ts *TCPServer) ClientCount() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return len(ts.clients)
}

func (ts *TCPServer) handleConnection(c *client) {
	remote := c.conn.RemoteAddr().String()
	defer func() {
		c.conn.Close()
		ts.mu.Lock()
		delete(ts.clients, c.conn)
		ts.mu.Unlock()
		if ts.metrics != nil {
			ts.metrics.RemoveConnectedClient()
		}
		logger.Info("client disconnected", "remote", remote, "clients", ts.ClientCount())
	}()

	for {
		cmd, args, err := c.reader.ReadCommand()
		if err != nil {
			if !ts.running {
				return
			}
			if strings.Contains(err.Error(), "use of closed") || strings.Contains(err.Error(), "reset") || strings.Contains(err.Error(), "EOF") {
				return
			}
			logger.Warn("command read error", "remote", remote, "error", err)
			c.writer.WriteError(fmt.Sprintf("ERR %s", err.Error()))
			return
		}

		cmd = strings.ToLower(cmd)

		if ts.appConfig != nil && ts.appConfig.RequireAuth() && !c.authed {
			if cmd != "auth" {
				c.writer.WriteError("NOAUTH Authentication required")
				continue
			}
		}

		if c.tx != nil && c.tx.State == cache.TxMulti {
			switch cmd {
			case "exec":
				if err := ts.cmdExec(c, args); err != nil {
					logger.Error("command execution error", "remote", remote, "command", cmd, "error", err)
					c.writer.WriteError(fmt.Sprintf("ERR %s", err.Error()))
				}
			case "discard":
				if err := ts.cmdDiscard(c, args); err != nil {
					c.writer.WriteError(fmt.Sprintf("ERR %s", err.Error()))
				}
			case "multi":
				c.writer.WriteError("ERR MULTI calls can not be nested")
			case "watch":
				c.writer.WriteError("ERR WATCH inside MULTI is not allowed")
			default:
				if err := c.server.transactionManager.QueueCommand(c.tx, cmd, args); err != nil {
					c.writer.WriteError(fmt.Sprintf("ERR %s", err.Error()))
				} else {
					c.writer.WriteSimpleString("QUEUED")
				}
			}
			continue
		}

		handler, ok := ts.commands[cmd]
		if !ok {
			logger.Warn("unknown command", "remote", remote, "command", cmd)
			c.writer.WriteError(fmt.Sprintf("ERR unknown command '%s'", cmd))
			continue
		}

		if ts.metrics != nil {
			ts.metrics.RecordCommand()
		}

		if err := handler(c, args); err != nil {
			logger.Error("command execution error", "remote", remote, "command", cmd, "error", err)
			c.writer.WriteError(fmt.Sprintf("ERR %s", err.Error()))
		}
	}
}

// ===================== Connection Commands =====================

func (ts *TCPServer) cmdPing(c *client, args []string) error {
	if len(args) > 0 {
		return c.writer.WriteSimpleString(args[0])
	}
	return c.writer.WritePong()
}

func (ts *TCPServer) cmdEcho(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'echo' command")
	}
	return c.writer.WriteBulkString(args[0])
}

func (ts *TCPServer) cmdSelect(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'select' command")
	}
	db, err := strconv.Atoi(args[0])
	if err != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}
	c.db = db
	return c.writer.WriteOK()
}

func (ts *TCPServer) cmdQuit(c *client, args []string) error {
	c.writer.WriteSimpleString("OK")
	c.conn.Close()
	return nil
}

func (ts *TCPServer) cmdCommand(c *client, args []string) error {
	return c.writer.WriteOK()
}

func (ts *TCPServer) cmdInfo(c *client, args []string) error {
	snapshot := ts.cache.Stats.GetSnapshot()
	info := fmt.Sprintf("# Server\r\nredis_version:GoCache-1.1.0\r\ntcp_port:%d\r\nuptime_in_seconds:%d\r\n# Stats\r\nkeyspace_hits:%d\r\nkeyspace_misses:%d\r\n",
		ts.tcpConfig.Port, int(time.Since(ts.startTime).Seconds()), snapshot.Hits, snapshot.Misses)
	return c.writer.WriteBulkString(info)
}

func (ts *TCPServer) cmdDBSize(c *client, args []string) error {
	return c.writer.WriteInteger(int64(ts.cache.Count()))
}

func (ts *TCPServer) cmdFlushDB(c *client, args []string) error {
	ts.cache.Clear()
	return c.writer.WriteOK()
}

func (ts *TCPServer) cmdFlushAll(c *client, args []string) error {
	ts.cache.Clear()
	return c.writer.WriteOK()
}

func (ts *TCPServer) cmdTime(c *client, args []string) error {
	now := time.Now()
	sec := strconv.FormatInt(now.Unix(), 10)
	usec := strconv.FormatInt(int64(now.Nanosecond()/1000), 10)
	return c.writer.WriteStringArray([]string{sec, usec})
}

func (ts *TCPServer) cmdLastSave(c *client, args []string) error {
	return c.writer.WriteInteger(ts.startTime.Unix())
}

// ===================== Key Commands =====================

func (ts *TCPServer) cmdSet(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'set' command")
	}

	key := args[0]
	value := args[1]
	ttl := time.Duration(0)

	for i := 2; i < len(args); i++ {
		switch strings.ToUpper(args[i]) {
		case "EX":
			if i+1 >= len(args) {
				return c.writer.WriteError("ERR syntax error")
			}
			seconds, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil {
				return c.writer.WriteError("ERR value is not an integer or out of range")
			}
			ttl = time.Duration(seconds) * time.Second
			i++
		case "PX":
			if i+1 >= len(args) {
				return c.writer.WriteError("ERR syntax error")
			}
			ms, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil {
				return c.writer.WriteError("ERR value is not an integer or out of range")
			}
			ttl = time.Duration(ms) * time.Millisecond
			i++
		case "NX":
			if ts.cache.Exists(key) {
				return c.writer.WriteNullBulkString()
			}
		case "XX":
			if !ts.cache.Exists(key) {
				return c.writer.WriteNullBulkString()
			}
		}
	}

	ts.cache.Set(key, value, ttl)
	return c.writer.WriteOK()
}

func (ts *TCPServer) cmdGet(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'get' command")
	}

	value, found := ts.cache.Get(args[0])
	if !found {
		return c.writer.WriteNullBulkString()
	}

	return c.writer.WriteBulkString(fmt.Sprintf("%v", value))
}

func (ts *TCPServer) cmdGetDel(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'getdel' command")
	}

	value, found := ts.cache.GetDel(args[0])
	if !found {
		return c.writer.WriteNullBulkString()
	}

	return c.writer.WriteBulkString(fmt.Sprintf("%v", value))
}

func (ts *TCPServer) cmdDel(c *client, args []string) error {
	if len(args) < 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'del' command")
	}

	deleted := int64(0)
	for _, key := range args {
		if ts.cache.Delete(key) {
			deleted++
		}
	}
	return c.writer.WriteInteger(deleted)
}

func (ts *TCPServer) cmdExists(c *client, args []string) error {
	if len(args) < 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'exists' command")
	}

	count := int64(0)
	for _, key := range args {
		if ts.cache.Exists(key) {
			count++
		}
	}
	return c.writer.WriteInteger(count)
}

func (ts *TCPServer) cmdKeys(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'keys' command")
	}

	pattern := args[0]
	allKeys := ts.cache.Keys()

	if pattern == "*" {
		return c.writer.WriteStringArray(allKeys)
	}

	var matched []string
	for _, key := range allKeys {
		if simpleMatch(pattern, key) {
			matched = append(matched, key)
		}
	}
	return c.writer.WriteStringArray(matched)
}

func (ts *TCPServer) cmdExpire(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'expire' command")
	}

	seconds, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	value, found := ts.cache.Get(args[0])
	if !found {
		return c.writer.WriteInteger(0)
	}

	ts.cache.Set(args[0], value, time.Duration(seconds)*time.Second)
	return c.writer.WriteInteger(1)
}

func (ts *TCPServer) cmdTTL(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'ttl' command")
	}

	key := args[0]
	item, found := ts.cache.Items()[key]
	if !found {
		return c.writer.WriteInteger(-2)
	}

	if item.Expiration == 0 {
		return c.writer.WriteInteger(-1)
	}

	remaining := time.Until(time.Unix(0, item.Expiration))
	if remaining <= 0 {
		return c.writer.WriteInteger(-2)
	}

	return c.writer.WriteInteger(int64(remaining.Seconds()))
}

func (ts *TCPServer) cmdPTTL(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'pttl' command")
	}

	key := args[0]
	item, found := ts.cache.Items()[key]
	if !found {
		return c.writer.WriteInteger(-2)
	}

	if item.Expiration == 0 {
		return c.writer.WriteInteger(-1)
	}

	remaining := time.Until(time.Unix(0, item.Expiration))
	if remaining <= 0 {
		return c.writer.WriteInteger(-2)
	}

	return c.writer.WriteInteger(remaining.Milliseconds())
}

func (ts *TCPServer) cmdPersist(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'persist' command")
	}

	value, found := ts.cache.Get(args[0])
	if !found {
		return c.writer.WriteInteger(0)
	}

	ts.cache.Set(args[0], value, 0)
	return c.writer.WriteInteger(1)
}

func (ts *TCPServer) cmdType(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'type' command")
	}

	return c.writer.WriteSimpleString(ts.cache.Type(args[0]))
}

func (ts *TCPServer) cmdRename(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'rename' command")
	}

	value, found := ts.cache.Get(args[0])
	if !found {
		return c.writer.WriteError("ERR no such key")
	}

	ts.cache.Delete(args[0])
	ts.cache.Set(args[1], value, 0)
	return c.writer.WriteOK()
}

func (ts *TCPServer) cmdScan(c *client, args []string) error {
	cursor := uint64(0)
	count := 10

	if len(args) >= 1 {
		if c, err := strconv.ParseUint(args[0], 10, 64); err == nil {
			cursor = c
		}
	}

	for i := 1; i < len(args)-1; i++ {
		if strings.ToUpper(args[i]) == "COUNT" {
			if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
				count = n
			}
		}
	}

	nextCursor, keys := ts.cache.Scan(cursor, count)

	result := make([]resp.Value, 2)
	result[0] = resp.NewBulkString(strconv.FormatUint(nextCursor, 10))
	vals := make([]resp.Value, len(keys))
	for i, k := range keys {
		vals[i] = resp.NewBulkString(k)
	}
	result[1] = resp.NewArray(vals)

	return c.writer.WriteArray(result)
}

// ===================== String Commands =====================

func (ts *TCPServer) cmdAppend(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'append' command")
	}

	length := ts.stringCache.Append(args[0], args[1])
	return c.writer.WriteInteger(int64(length))
}

func (ts *TCPServer) cmdIncr(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'incr' command")
	}

	result, err := ts.stringCache.Incr(args[0])
	if err != nil {
		return c.writer.WriteError(fmt.Sprintf("ERR %s", err.Error()))
	}
	return c.writer.WriteInteger(result)
}

func (ts *TCPServer) cmdDecr(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'decr' command")
	}

	result, err := ts.stringCache.Decr(args[0])
	if err != nil {
		return c.writer.WriteError(fmt.Sprintf("ERR %s", err.Error()))
	}
	return c.writer.WriteInteger(result)
}

func (ts *TCPServer) cmdIncrBy(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'incrby' command")
	}

	n, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	result, err2 := ts.stringCache.IncrBy(args[0], n)
	if err2 != nil {
		return c.writer.WriteError(fmt.Sprintf("ERR %s", err2.Error()))
	}
	return c.writer.WriteInteger(result)
}

func (ts *TCPServer) cmdDecrBy(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'decrby' command")
	}

	n, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	result, err2 := ts.stringCache.DecrBy(args[0], n)
	if err2 != nil {
		return c.writer.WriteError(fmt.Sprintf("ERR %s", err2.Error()))
	}
	return c.writer.WriteInteger(result)
}

func (ts *TCPServer) cmdIncrByFloat(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'incrbyfloat' command")
	}

	n, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return c.writer.WriteError("ERR value is not a valid float")
	}

	current, found := ts.stringCache.Get(args[0])
	if !found {
		ts.stringCache.Set(args[0], fmt.Sprintf("%g", n), 0)
		return c.writer.WriteBulkString(fmt.Sprintf("%g", n))
	}

	currentFloat, err := strconv.ParseFloat(current, 64)
	if err != nil {
		return c.writer.WriteError("ERR value is not a valid float")
	}

	result := currentFloat + n
	ts.stringCache.Set(args[0], fmt.Sprintf("%g", result), 0)
	return c.writer.WriteBulkString(fmt.Sprintf("%g", result))
}

func (ts *TCPServer) cmdStrLen(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'strlen' command")
	}

	length, found := ts.stringCache.StrLen(args[0])
	if !found {
		return c.writer.WriteInteger(0)
	}
	return c.writer.WriteInteger(int64(length))
}

func (ts *TCPServer) cmdGetRange(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'getrange' command")
	}

	start, err1 := strconv.Atoi(args[1])
	end, err2 := strconv.Atoi(args[2])
	if err1 != nil || err2 != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	value, found := ts.stringCache.GetRange(args[0], start, end)
	if !found {
		return c.writer.WriteBulkString("")
	}
	return c.writer.WriteBulkString(value)
}

func (ts *TCPServer) cmdSetRange(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'setrange' command")
	}

	offset, err := strconv.Atoi(args[1])
	if err != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	length := ts.stringCache.SetRange(args[0], offset, args[2])
	return c.writer.WriteInteger(int64(length))
}

func (ts *TCPServer) cmdGetSet(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'getset' command")
	}

	oldValue, found := ts.stringCache.GetSet(args[0], args[1])
	if !found {
		return c.writer.WriteNullBulkString()
	}
	return c.writer.WriteBulkString(oldValue)
}

func (ts *TCPServer) cmdMGet(c *client, args []string) error {
	if len(args) < 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'mget' command")
	}

	vals := make([]resp.Value, len(args))
	for i, key := range args {
		value, found := ts.cache.Get(key)
		if found {
			vals[i] = resp.NewBulkString(fmt.Sprintf("%v", value))
		} else {
			vals[i] = resp.NewNullBulkString()
		}
	}
	return c.writer.WriteArray(vals)
}

func (ts *TCPServer) cmdMSet(c *client, args []string) error {
	if len(args) < 2 || len(args)%2 != 0 {
		return c.writer.WriteError("ERR wrong number of arguments for 'mset' command")
	}

	for i := 0; i < len(args); i += 2 {
		ts.cache.Set(args[i], args[i+1], 0)
	}
	return c.writer.WriteOK()
}

func (ts *TCPServer) cmdSetNX(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'setnx' command")
	}

	if ts.cache.Exists(args[0]) {
		return c.writer.WriteInteger(0)
	}

	ts.cache.Set(args[0], args[1], 0)
	return c.writer.WriteInteger(1)
}

func (ts *TCPServer) cmdSetEX(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'setex' command")
	}

	seconds, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil || seconds <= 0 {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	ts.cache.Set(args[0], args[2], time.Duration(seconds)*time.Second)
	return c.writer.WriteOK()
}

// ===================== List Commands =====================

func (ts *TCPServer) cmdLPush(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'lpush' command")
	}

	vals := make([]any, len(args)-1)
	for i := 1; i < len(args); i++ {
		vals[i-1] = args[i]
	}

	length := ts.listCache.LPush(args[0], 0, vals...)
	return c.writer.WriteInteger(int64(length))
}

func (ts *TCPServer) cmdRPush(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'rpush' command")
	}

	vals := make([]any, len(args)-1)
	for i := 1; i < len(args); i++ {
		vals[i-1] = args[i]
	}

	length := ts.listCache.RPush(args[0], 0, vals...)
	return c.writer.WriteInteger(int64(length))
}

func (ts *TCPServer) cmdLPop(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'lpop' command")
	}

	value, found := ts.listCache.LPop(args[0])
	if !found {
		return c.writer.WriteNullBulkString()
	}
	return c.writer.WriteBulkString(fmt.Sprintf("%v", value))
}

func (ts *TCPServer) cmdRPop(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'rpop' command")
	}

	value, found := ts.listCache.RPop(args[0])
	if !found {
		return c.writer.WriteNullBulkString()
	}
	return c.writer.WriteBulkString(fmt.Sprintf("%v", value))
}

func (ts *TCPServer) cmdLRange(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'lrange' command")
	}

	start, err1 := strconv.Atoi(args[1])
	stop, err2 := strconv.Atoi(args[2])
	if err1 != nil || err2 != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	values, found := ts.listCache.LRange(args[0], start, stop)
	if !found {
		return c.writer.WriteNullArray()
	}

	vals := make([]resp.Value, len(values))
	for i, v := range values {
		vals[i] = resp.NewBulkString(fmt.Sprintf("%v", v))
	}
	return c.writer.WriteArray(vals)
}

func (ts *TCPServer) cmdLIndex(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'lindex' command")
	}

	index, err := strconv.Atoi(args[1])
	if err != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	value, found := ts.listCache.LIndex(args[0], index)
	if !found {
		return c.writer.WriteNullBulkString()
	}
	return c.writer.WriteBulkString(fmt.Sprintf("%v", value))
}

func (ts *TCPServer) cmdLLen(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'llen' command")
	}

	length, found := ts.listCache.LLen(args[0])
	if !found {
		return c.writer.WriteInteger(0)
	}
	return c.writer.WriteInteger(int64(length))
}

func (ts *TCPServer) cmdLTrim(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'ltrim' command")
	}

	start, err1 := strconv.Atoi(args[1])
	stop, err2 := strconv.Atoi(args[2])
	if err1 != nil || err2 != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	ts.listCache.LTrim(args[0], start, stop)
	return c.writer.WriteOK()
}

func (ts *TCPServer) cmdLRem(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'lrem' command")
	}

	count, err1 := strconv.Atoi(args[1])
	if err1 != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	removed := ts.listCache.LRem(args[0], count, args[2])
	return c.writer.WriteInteger(int64(removed))
}

// ===================== Hash Commands =====================

func (ts *TCPServer) cmdHSet(c *client, args []string) error {
	if len(args) < 3 || (len(args)-1)%2 != 0 {
		return c.writer.WriteError("ERR wrong number of arguments for 'hset' command")
	}

	fields := make(map[string]any)
	for i := 1; i < len(args); i += 2 {
		fields[args[i]] = args[i+1]
	}

	count := ts.hashCache.HSet(args[0], 0, fields)
	return c.writer.WriteInteger(int64(count))
}

func (ts *TCPServer) cmdHGet(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'hget' command")
	}

	value, found := ts.hashCache.HGet(args[0], args[1])
	if !found {
		return c.writer.WriteNullBulkString()
	}
	return c.writer.WriteBulkString(fmt.Sprintf("%v", value))
}

func (ts *TCPServer) cmdHGetAll(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'hgetall' command")
	}

	fields, found := ts.hashCache.HGetAll(args[0])
	if !found {
		return c.writer.WriteNullArray()
	}

	vals := make([]resp.Value, 0, len(fields)*2)
	for k, v := range fields {
		vals = append(vals, resp.NewBulkString(k), resp.NewBulkString(fmt.Sprintf("%v", v)))
	}
	return c.writer.WriteArray(vals)
}

func (ts *TCPServer) cmdHDel(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'hdel' command")
	}

	deleted := ts.hashCache.HDel(args[0], args[1:]...)
	return c.writer.WriteInteger(int64(deleted))
}

func (ts *TCPServer) cmdHExists(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'hexists' command")
	}

	exists := ts.hashCache.HExists(args[0], args[1])
	if exists {
		return c.writer.WriteInteger(1)
	}
	return c.writer.WriteInteger(0)
}

func (ts *TCPServer) cmdHLen(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'hlen' command")
	}

	length, found := ts.hashCache.HLen(args[0])
	if !found {
		return c.writer.WriteInteger(0)
	}
	return c.writer.WriteInteger(int64(length))
}

func (ts *TCPServer) cmdHKeys(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'hkeys' command")
	}

	keys, found := ts.hashCache.HKeys(args[0])
	if !found {
		return c.writer.WriteNullArray()
	}
	return c.writer.WriteStringArray(keys)
}

func (ts *TCPServer) cmdHVals(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'hvals' command")
	}

	vals, found := ts.hashCache.HVals(args[0])
	if !found {
		return c.writer.WriteNullArray()
	}

	result := make([]resp.Value, len(vals))
	for i, v := range vals {
		result[i] = resp.NewBulkString(fmt.Sprintf("%v", v))
	}
	return c.writer.WriteArray(result)
}

func (ts *TCPServer) cmdHSetNX(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'hsetnx' command")
	}

	set := ts.hashCache.HSetNX(args[0], args[1], 0, args[2])
	if set {
		return c.writer.WriteInteger(1)
	}
	return c.writer.WriteInteger(0)
}

func (ts *TCPServer) cmdHIncrBy(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'hincrby' command")
	}

	n, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	result, err2 := ts.hashCache.HIncrBy(args[0], args[1], 0, n)
	if err2 != nil {
		return c.writer.WriteError(fmt.Sprintf("ERR %s", err2.Error()))
	}
	return c.writer.WriteInteger(result)
}

func (ts *TCPServer) cmdHIncrByFloat(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'hincrbyfloat' command")
	}

	n, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		return c.writer.WriteError("ERR value is not a valid float")
	}

	result, err2 := ts.hashCache.HIncrBy(args[0], args[1], 0, int64(n))
	if err2 != nil {
		return c.writer.WriteError(fmt.Sprintf("ERR %s", err2.Error()))
	}
	return c.writer.WriteBulkString(strconv.FormatInt(result, 10))
}

func (ts *TCPServer) cmdHMSet(c *client, args []string) error {
	return ts.cmdHSet(c, args)
}

func (ts *TCPServer) cmdHMGet(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'hmget' command")
	}

	vals := make([]resp.Value, len(args)-1)
	for i := 1; i < len(args); i++ {
		value, found := ts.hashCache.HGet(args[0], args[i])
		if found {
			vals[i-1] = resp.NewBulkString(fmt.Sprintf("%v", value))
		} else {
			vals[i-1] = resp.NewNullBulkString()
		}
	}
	return c.writer.WriteArray(vals)
}

// ===================== Set Commands =====================

func (ts *TCPServer) cmdSAdd(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'sadd' command")
	}

	members := make([]any, len(args)-1)
	for i := 1; i < len(args); i++ {
		members[i-1] = args[i]
	}

	added := ts.setCache.SAdd(args[0], 0, members...)
	return c.writer.WriteInteger(int64(added))
}

func (ts *TCPServer) cmdSRem(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'srem' command")
	}

	members := make([]any, len(args)-1)
	for i := 1; i < len(args); i++ {
		members[i-1] = args[i]
	}

	removed := ts.setCache.SRem(args[0], members...)
	return c.writer.WriteInteger(int64(removed))
}

func (ts *TCPServer) cmdSIsMember(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'sismember' command")
	}

	if ts.setCache.SIsMember(args[0], args[1]) {
		return c.writer.WriteInteger(1)
	}
	return c.writer.WriteInteger(0)
}

func (ts *TCPServer) cmdSCard(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'scard' command")
	}

	card, found := ts.setCache.SCard(args[0])
	if !found {
		return c.writer.WriteInteger(0)
	}
	return c.writer.WriteInteger(int64(card))
}

func (ts *TCPServer) cmdSMembers(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'smembers' command")
	}

	members, found := ts.setCache.SMembers(args[0])
	if !found {
		return c.writer.WriteNullArray()
	}

	vals := make([]resp.Value, len(members))
	for i, m := range members {
		vals[i] = resp.NewBulkString(fmt.Sprintf("%v", m))
	}
	return c.writer.WriteArray(vals)
}

func (ts *TCPServer) cmdSPop(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'spop' command")
	}

	member, found := ts.setCache.SPop(args[0])
	if !found {
		return c.writer.WriteNullBulkString()
	}
	return c.writer.WriteBulkString(fmt.Sprintf("%v", member))
}

func (ts *TCPServer) cmdSUnion(c *client, args []string) error {
	if len(args) < 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'sunion' command")
	}

	result := ts.setCache.SUnion(args...)
	vals := make([]resp.Value, len(result))
	for i, m := range result {
		vals[i] = resp.NewBulkString(fmt.Sprintf("%v", m))
	}
	return c.writer.WriteArray(vals)
}

func (ts *TCPServer) cmdSInter(c *client, args []string) error {
	if len(args) < 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'sinter' command")
	}

	result := ts.setCache.SInter(args...)
	vals := make([]resp.Value, len(result))
	for i, m := range result {
		vals[i] = resp.NewBulkString(fmt.Sprintf("%v", m))
	}
	return c.writer.WriteArray(vals)
}

func (ts *TCPServer) cmdSDiff(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'sdiff' command")
	}

	result := ts.setCache.SDiff(args[0], args[1])
	vals := make([]resp.Value, len(result))
	for i, m := range result {
		vals[i] = resp.NewBulkString(fmt.Sprintf("%v", m))
	}
	return c.writer.WriteArray(vals)
}

// ===================== Sorted Set Commands =====================

func (ts *TCPServer) cmdZAdd(c *client, args []string) error {
	if len(args) < 3 || len(args)%2 != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zadd' command")
	}

	members := make(map[string]float64)
	for i := 1; i < len(args); i += 2 {
		score, err := strconv.ParseFloat(args[i], 64)
		if err != nil {
			return c.writer.WriteError("ERR value is not a valid float")
		}
		members[args[i+1]] = score
	}

	added := ts.sortedSetCache.ZAdd(args[0], 0, members)
	return c.writer.WriteInteger(int64(added))
}

func (ts *TCPServer) cmdZRem(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zrem' command")
	}

	removed := ts.sortedSetCache.ZRem(args[0], args[1:]...)
	return c.writer.WriteInteger(int64(removed))
}

func (ts *TCPServer) cmdZScore(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zscore' command")
	}

	score, found := ts.sortedSetCache.ZScore(args[0], args[1])
	if !found {
		return c.writer.WriteNullBulkString()
	}
	return c.writer.WriteBulkString(strconv.FormatFloat(score, 'f', -1, 64))
}

func (ts *TCPServer) cmdZCard(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zcard' command")
	}

	card, found := ts.sortedSetCache.ZCard(args[0])
	if !found {
		return c.writer.WriteInteger(0)
	}
	return c.writer.WriteInteger(int64(card))
}

func (ts *TCPServer) cmdZRank(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zrank' command")
	}

	rank, found := ts.sortedSetCache.ZRank(args[0], args[1])
	if !found {
		return c.writer.WriteNullBulkString()
	}
	return c.writer.WriteInteger(int64(rank))
}

func (ts *TCPServer) cmdZRevRank(c *client, args []string) error {
	if len(args) != 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zrevrank' command")
	}

	rank, found := ts.sortedSetCache.ZRevRank(args[0], args[1])
	if !found {
		return c.writer.WriteNullBulkString()
	}
	return c.writer.WriteInteger(int64(rank))
}

func (ts *TCPServer) cmdZRange(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zrange' command")
	}

	start, err1 := strconv.Atoi(args[1])
	stop, err2 := strconv.Atoi(args[2])
	if err1 != nil || err2 != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	result, found := ts.sortedSetCache.ZRange(args[0], start, stop)
	if !found {
		return c.writer.WriteNullArray()
	}

	return c.writer.WriteStringArray(zsetMembersToStrings(result))
}

func (ts *TCPServer) cmdZRevRange(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zrevrange' command")
	}

	start, err1 := strconv.Atoi(args[1])
	stop, err2 := strconv.Atoi(args[2])
	if err1 != nil || err2 != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	result, found := ts.sortedSetCache.ZRevRange(args[0], start, stop)
	if !found {
		return c.writer.WriteNullArray()
	}

	return c.writer.WriteStringArray(zsetMembersToStrings(result))
}

func (ts *TCPServer) cmdZRangeByScore(c *client, args []string) error {
	if len(args) < 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zrangebyscore' command")
	}

	min, err1 := strconv.ParseFloat(args[1], 64)
	max, err2 := strconv.ParseFloat(args[2], 64)
	if err1 != nil || err2 != nil {
		return c.writer.WriteError("ERR min or max is not a valid float")
	}

	offset, count := 0, -1
	for i := 3; i < len(args)-1; i++ {
		switch strings.ToUpper(args[i]) {
		case "LIMIT":
			offset, _ = strconv.Atoi(args[i+1])
			count, _ = strconv.Atoi(args[i+2])
			i += 2
		}
	}

	result, found := ts.sortedSetCache.ZRangeByScore(args[0], min, max, offset, count)
	if !found {
		return c.writer.WriteNullArray()
	}

	return c.writer.WriteStringArray(zsetMembersToStrings(result))
}

func (ts *TCPServer) cmdZRevRangeByScore(c *client, args []string) error {
	if len(args) < 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zrevrangebyscore' command")
	}

	max, err1 := strconv.ParseFloat(args[1], 64)
	min, err2 := strconv.ParseFloat(args[2], 64)
	if err1 != nil || err2 != nil {
		return c.writer.WriteError("ERR max or min is not a valid float")
	}

	offset, count := 0, -1
	for i := 3; i < len(args)-1; i++ {
		switch strings.ToUpper(args[i]) {
		case "LIMIT":
			offset, _ = strconv.Atoi(args[i+1])
			count, _ = strconv.Atoi(args[i+2])
			i += 2
		}
	}

	result, found := ts.sortedSetCache.ZRevRangeByScore(args[0], max, min, offset, count)
	if !found {
		return c.writer.WriteNullArray()
	}

	return c.writer.WriteStringArray(zsetMembersToStrings(result))
}

func (ts *TCPServer) cmdZCount(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zcount' command")
	}

	min, err1 := strconv.ParseFloat(args[1], 64)
	max, err2 := strconv.ParseFloat(args[2], 64)
	if err1 != nil || err2 != nil {
		return c.writer.WriteError("ERR min or max is not a valid float")
	}

	count, found := ts.sortedSetCache.ZCount(args[0], min, max)
	if !found {
		return c.writer.WriteInteger(0)
	}
	return c.writer.WriteInteger(int64(count))
}

func (ts *TCPServer) cmdZIncrBy(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zincrby' command")
	}

	increment, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return c.writer.WriteError("ERR value is not a valid float")
	}

	score, _ := ts.sortedSetCache.ZIncrBy(args[0], args[2], increment)
	return c.writer.WriteBulkString(strconv.FormatFloat(score, 'f', -1, 64))
}

func (ts *TCPServer) cmdZPopMin(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zpopmin' command")
	}

	sm, found := ts.sortedSetCache.ZPopMin(args[0])
	if !found {
		return c.writer.WriteNullArray()
	}

	return c.writer.WriteArray([]resp.Value{
		resp.NewBulkString(sm.Member),
		resp.NewBulkString(strconv.FormatFloat(sm.Score, 'f', -1, 64)),
	})
}

func (ts *TCPServer) cmdZPopMax(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zpopmax' command")
	}

	sm, found := ts.sortedSetCache.ZPopMax(args[0])
	if !found {
		return c.writer.WriteNullArray()
	}

	return c.writer.WriteArray([]resp.Value{
		resp.NewBulkString(sm.Member),
		resp.NewBulkString(strconv.FormatFloat(sm.Score, 'f', -1, 64)),
	})
}

func (ts *TCPServer) cmdZRemRangeByRank(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zremrangebyrank' command")
	}

	start, err1 := strconv.Atoi(args[1])
	stop, err2 := strconv.Atoi(args[2])
	if err1 != nil || err2 != nil {
		return c.writer.WriteError("ERR value is not an integer or out of range")
	}

	removed, found := ts.sortedSetCache.ZRemRangeByRank(args[0], start, stop)
	if !found {
		return c.writer.WriteInteger(0)
	}
	return c.writer.WriteInteger(int64(removed))
}

func (ts *TCPServer) cmdZRemRangeByScore(c *client, args []string) error {
	if len(args) != 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'zremrangebyscore' command")
	}

	min, err1 := strconv.ParseFloat(args[1], 64)
	max, err2 := strconv.ParseFloat(args[2], 64)
	if err1 != nil || err2 != nil {
		return c.writer.WriteError("ERR min or max is not a valid float")
	}

	removed, found := ts.sortedSetCache.ZRemRangeByScore(args[0], min, max)
	if !found {
		return c.writer.WriteInteger(0)
	}
	return c.writer.WriteInteger(int64(removed))
}

// ===================== Transaction Commands =====================

func (ts *TCPServer) cmdMulti(c *client, args []string) error {
	if c.tx != nil && c.tx.State == cache.TxMulti {
		return c.writer.WriteError("ERR MULTI calls can not be nested")
	}

	c.tx = ts.transactionManager.Begin()
	logger.Info("transaction started", "remote", c.conn.RemoteAddr().String())
	return c.writer.WriteOK()
}

func (ts *TCPServer) cmdExec(c *client, args []string) error {
	if c.tx == nil || c.tx.State != cache.TxMulti {
		return c.writer.WriteError("ERR EXEC without MULTI")
	}

	results, err := ts.transactionManager.Exec(c.tx, func(cmd string, cmdArgs []string) error {
		handler, ok := ts.commands[cmd]
		if !ok {
			return fmt.Errorf("unknown command '%s'", cmd)
		}
		return handler(c, cmdArgs)
	})

	if err != nil {
		return c.writer.WriteError(fmt.Sprintf("ERR %s", err.Error()))
	}

	if results == nil {
		return c.writer.WriteNullArray()
	}

	vals := make([]resp.Value, len(results))
	for i, e := range results {
		if e != nil {
			vals[i] = resp.NewError(e.Error())
		} else {
			vals[i] = resp.NewSimpleString("OK")
		}
	}

	c.tx = nil
	return c.writer.WriteArray(vals)
}

func (ts *TCPServer) cmdDiscard(c *client, args []string) error {
	if c.tx == nil || c.tx.State != cache.TxMulti {
		return c.writer.WriteError("ERR DISCARD without MULTI")
	}

	ts.transactionManager.Discard(c.tx)
	c.tx = nil
	return c.writer.WriteOK()
}

func (ts *TCPServer) cmdWatch(c *client, args []string) error {
	if len(args) < 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'watch' command")
	}

	if c.tx != nil && c.tx.State == cache.TxMulti {
		return c.writer.WriteError("ERR WATCH inside MULTI is not allowed")
	}

	if c.tx == nil {
		c.tx = &cache.Transaction{}
	}

	if err := ts.transactionManager.Watch(c.tx, args...); err != nil {
		return c.writer.WriteError(fmt.Sprintf("ERR %s", err.Error()))
	}

	return c.writer.WriteOK()
}

func (ts *TCPServer) cmdUnwatch(c *client, args []string) error {
	if c.tx != nil {
		ts.transactionManager.Unwatch(c.tx)
	}
	return c.writer.WriteOK()
}

func (ts *TCPServer) cmdAuth(c *client, args []string) error {
	if len(args) != 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'auth' command")
	}

	if ts.appConfig == nil || !ts.appConfig.RequireAuth() {
		return c.writer.WriteError("ERR Client sent AUTH, but no password is set")
	}

	if ts.appConfig.CheckPassword(args[0]) {
		c.authed = true
		logger.Info("client authenticated", "remote", c.conn.RemoteAddr().String())
		return c.writer.WriteOK()
	}

	logger.Warn("authentication failed", "remote", c.conn.RemoteAddr().String())
	return c.writer.WriteError("ERR invalid password")
}

// ===================== Helper Functions =====================

func zsetMembersToStrings(members []cache.ScoredMember) []string {
	result := make([]string, len(members))
	for i, m := range members {
		result[i] = m.Member
	}
	return result
}

func simpleMatch(pattern, s string) bool {
	if pattern == "*" {
		return true
	}

	pi, si := 0, 0
	for pi < len(pattern) && si < len(s) {
		if pattern[pi] == '*' {
			pi++
			if pi >= len(pattern) {
				return true
			}
			for si < len(s) {
				if simpleMatch(pattern[pi:], s[si:]) {
					return true
				}
				si++
			}
			return false
		} else if pattern[pi] == '?' || pattern[pi] == s[si] {
			pi++
			si++
		} else {
			return false
		}
	}

	for pi < len(pattern) && pattern[pi] == '*' {
		pi++
	}

	return pi == len(pattern) && si == len(s)
}

func (ts *TCPServer) SetConfig(cfg *cache.Config) {
	ts.appConfig = cfg
}

func (ts *TCPServer) SetMetrics(m *cache.MetricsCollector) {
	ts.metrics = m
}

func (ts *TCPServer) GetMetrics() *cache.MetricsCollector {
	return ts.metrics
}
