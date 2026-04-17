package cache

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"GoCache/logger"
)

type ScriptEngine struct {
	cache    *MemoryCache
	mu       sync.RWMutex
	scripts  map[string]string
	timeout  time.Duration
}

func NewScriptEngine(cache *MemoryCache) *ScriptEngine {
	return &ScriptEngine{
		cache:   cache,
		scripts: make(map[string]string),
		timeout: 5 * time.Second,
	}
}

func (se *ScriptEngine) SetTimeout(d time.Duration) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.timeout = d
}

func scriptSHA1(script string) string {
	h := sha1.Sum([]byte(script))
	return hex.EncodeToString(h[:])
}

func (se *ScriptEngine) Eval(script string, numKeys int, keysAndArgs []string) (any, error) {
	logger.Debug("EVAL executing script", "sha", scriptSHA1(script), "num_keys", numKeys)

	ctx := &ScriptContext{
		cache:    se.cache,
		numKeys:  numKeys,
		keys:     make([]string, 0),
		args:     make([]string, 0),
		startTime: time.Now(),
	}

	for i := 0; i < len(keysAndArgs); i++ {
		if i < numKeys {
			ctx.keys = append(ctx.keys, keysAndArgs[i])
		} else {
			ctx.args = append(ctx.args, keysAndArgs[i])
		}
	}

	result, err := se.executeScript(script, ctx)
	if err != nil {
		logger.Warn("script execution failed", "error", err)
		return nil, err
	}

	return result, nil
}

func (se *ScriptEngine) EvalSHA(sha string, numKeys int, keysAndArgs []string) (any, error) {
	se.mu.RLock()
	script, found := se.scripts[sha]
	se.mu.RUnlock()

	if !found {
		return nil, fmt.Errorf("NOSCRIPT No matching script. Please use EVAL")
	}

	return se.Eval(script, numKeys, keysAndArgs)
}

func (se *ScriptEngine) ScriptLoad(script string) (string, error) {
	sha := scriptSHA1(script)

	se.mu.Lock()
	se.scripts[sha] = script
	se.mu.Unlock()

	logger.Debug("SCRIPT LOAD", "sha", sha)
	return sha, nil
}

func (se *ScriptEngine) ScriptExists(shas ...string) []bool {
	se.mu.RLock()
	defer se.mu.RUnlock()

	result := make([]bool, len(shas))
	for i, sha := range shas {
		_, found := se.scripts[sha]
		result[i] = found
	}
	return result
}

func (se *ScriptEngine) ScriptFlush() {
	se.mu.Lock()
	defer se.mu.Unlock()

	se.scripts = make(map[string]string)
	logger.Info("script cache flushed")
}

func (se *ScriptEngine) ScriptCount() int {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return len(se.scripts)
}

type ScriptContext struct {
	cache     *MemoryCache
	numKeys   int
	keys      []string
	args      []string
	startTime time.Time
}

func (sc *ScriptContext) KEYS() []string {
	return sc.keys
}

func (sc *ScriptContext) ARGV() []string {
	return sc.args
}

func (sc *ScriptContext) Call(command string, args ...string) (any, error) {
	return executeCacheCommand(sc.cache, command, args)
}

func executeCacheCommand(cache *MemoryCache, command string, args []string) (any, error) {
	switch stringsToUpper(command) {
	case "SET":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'set' command")
		}
		cache.Set(args[0], args[1], 0)
		return "OK", nil

	case "GET":
		if len(args) < 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'get' command")
		}
		val, found := cache.Get(args[0])
		if !found {
			return nil, nil
		}
		return val, nil

	case "DEL":
		if len(args) < 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'del' command")
		}
		deleted := 0
		for _, key := range args {
			if cache.Delete(key) {
				deleted++
			}
		}
		return deleted, nil

	case "EXISTS":
		if len(args) < 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'exists' command")
		}
		count := 0
		for _, key := range args {
			if cache.Exists(key) {
				count++
			}
		}
		return count, nil

	case "INCR":
		if len(args) < 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'incr' command")
		}
		sc := NewStringCache(cache)
		return sc.Incr(args[0])

	case "DECR":
		if len(args) < 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'decr' command")
		}
		sc := NewStringCache(cache)
		return sc.Decr(args[0])

	case "HSET":
		if len(args) < 3 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'hset' command")
		}
		hc := NewHashCacheWithMemory(cache)
		return hc.HSet(args[0], 0, map[string]any{args[1]: args[2]}), nil

	case "HGET":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'hget' command")
		}
		hc := NewHashCacheWithMemory(cache)
		val, found := hc.HGet(args[0], args[1])
		if !found {
			return nil, nil
		}
		return val, nil

	case "HDEL":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'hdel' command")
		}
		hc := NewHashCacheWithMemory(cache)
		return hc.HDel(args[0], args[1:]...), nil

	case "LPUSH":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'lpush' command")
		}
		lc := NewListCacheWithMemory(cache)
		vals := make([]any, len(args)-1)
		for i, a := range args[1:] {
			vals[i] = a
		}
		return lc.LPush(args[0], 0, vals...), nil

	case "RPUSH":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'rpush' command")
		}
		lc := NewListCacheWithMemory(cache)
		vals := make([]any, len(args)-1)
		for i, a := range args[1:] {
			vals[i] = a
		}
		return lc.RPush(args[0], 0, vals...), nil

	case "LPOP":
		if len(args) < 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'lpop' command")
		}
		lc := NewListCacheWithMemory(cache)
		val, found := lc.LPop(args[0])
		if !found {
			return nil, nil
		}
		return val, nil

	case "RPOP":
		if len(args) < 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'rpop' command")
		}
		lc := NewListCacheWithMemory(cache)
		val, found := lc.RPop(args[0])
		if !found {
			return nil, nil
		}
		return val, nil

	case "SADD":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'sadd' command")
		}
		setc := NewSetCacheWithMemory(cache)
		vals := make([]any, len(args)-1)
		for i, a := range args[1:] {
			vals[i] = a
		}
		return setc.SAdd(args[0], 0, vals...), nil

	case "SREM":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'srem' command")
		}
		setc := NewSetCacheWithMemory(cache)
		vals := make([]any, len(args)-1)
		for i, a := range args[1:] {
			vals[i] = a
		}
		return setc.SRem(args[0], vals...), nil

	case "ZADD":
		if len(args) < 3 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'zadd' command")
		}
		zc := NewSortedSetCacheWithMemory(cache)
		score, err := parseFloat(args[1])
		if err != nil {
			return nil, fmt.Errorf("ERR value is not a valid float")
		}
		return zc.ZAdd(args[0], 0, map[string]float64{args[2]: score}), nil

	case "ZREM":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'zrem' command")
		}
		zc := NewSortedSetCacheWithMemory(cache)
		return zc.ZRem(args[0], args[1]), nil

	default:
		return nil, fmt.Errorf("ERR unknown command '%s' in script", command)
	}
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func (se *ScriptEngine) executeScript(script string, ctx *ScriptContext) (any, error) {
	lines := splitScriptLines(script)
	if len(lines) == 0 {
		return nil, nil
	}

	var lastResult any
	for _, line := range lines {
		line = trimSpace(line)
		if line == "" {
			continue
		}

		parts := splitCommand(line)
		if len(parts) == 0 {
			continue
		}

		cmd := parts[0]
		args := expandArgs(parts[1:], ctx)

		result, err := executeCacheCommand(se.cache, cmd, args)
		if err != nil {
			return nil, fmt.Errorf("script error at '%s': %w", line, err)
		}
		lastResult = result
	}

	return lastResult, nil
}

func splitScriptLines(script string) []string {
	var lines []string
	current := ""
	for _, ch := range script {
		if ch == '\n' || ch == ';' {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func splitCommand(line string) []string {
	var parts []string
	current := ""
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if inQuote {
			if ch == quoteChar {
				inQuote = false
			} else {
				current += string(ch)
			}
		} else {
			switch ch {
			case '"', '\'':
				inQuote = true
				quoteChar = ch
			case ' ', '\t':
				if current != "" {
					parts = append(parts, current)
					current = ""
				}
			default:
				current += string(ch)
			}
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func expandArgs(args []string, ctx *ScriptContext) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		result[i] = expandArg(arg, ctx)
	}
	return result
}

func expandArg(arg string, ctx *ScriptContext) string {
	if len(arg) < 2 {
		return arg
	}

	if arg[0] == '$' {
		idxStr := arg[1:]
		idx := 0
		if _, err := fmt.Sscanf(idxStr, "%d", &idx); err == nil {
			if idx < len(ctx.keys) {
				return ctx.keys[idx]
			}
			argvIdx := idx - len(ctx.keys)
			if argvIdx < len(ctx.args) {
				return ctx.args[argvIdx]
			}
		}
	}

	return arg
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
