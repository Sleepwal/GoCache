package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"GoCache/resp"
)

const (
	defaultHost = "127.0.0.1"
	defaultPort = 6379
)

var allCommands = []string{
	// Connection
	"PING", "ECHO", "SELECT", "QUIT", "COMMAND", "INFO", "DBSIZE", "FLUSHDB", "FLUSHALL", "TIME",
	// Key
	"SET", "GET", "GETDEL", "DEL", "EXISTS", "KEYS", "EXPIRE", "TTL", "PTTL", "PERSIST", "TYPE", "RENAME", "SCAN",
	// String
	"APPEND", "INCR", "DECR", "INCRBY", "DECRBY", "INCRBYFLOAT", "STRLEN", "GETRANGE", "SETRANGE", "GETSET", "MGET", "MSET", "SETNX", "SETEX",
	// List
	"LPUSH", "RPUSH", "LPOP", "RPOP", "LRANGE", "LINDEX", "LLEN", "LTRIM", "LREM",
	// Hash
	"HSET", "HGET", "HGETALL", "HDEL", "HEXISTS", "HLEN", "HKEYS", "HVALS", "HSETNX", "HINCRBY", "HINCRBYFLOAT", "HMSET", "HMGET",
	// Set
	"SADD", "SREM", "SISMEMBER", "SCARD", "SMEMBERS", "SPOP", "SUNION", "SINTER", "SDIFF",
	// Sorted Set
	"ZADD", "ZREM", "ZSCORE", "ZCARD", "ZRANK", "ZREVRANK", "ZRANGE", "ZREVRANGE", "ZRANGEBYSCORE", "ZREVRANGEBYSCORE", "ZCOUNT", "ZINCRBY", "ZPOPMIN", "ZPOPMAX", "ZREMRANGEBYRANK", "ZREMRANGEBYSCORE",
}

type cli struct {
	host   string
	port   int
	conn   net.Conn
	reader *resp.Reader
	writer *resp.Writer
}

func newCLI(host string, port int) *cli {
	return &cli{
		host: host,
		port: port,
	}
}

func (c *cli) connect() error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.host, c.port))
	if err != nil {
		return fmt.Errorf("could not connect to %s:%d: %w", c.host, c.port, err)
	}

	c.conn = conn
	c.reader = resp.NewReader(conn)
	c.writer = resp.NewWriter(conn)
	return nil
}

func (c *cli) close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *cli) sendCommand(cmd string, args []string) error {
	vals := make([]resp.Value, 1+len(args))
	vals[0] = resp.NewBulkString(cmd)
	for i, arg := range args {
		vals[i+1] = resp.NewBulkString(arg)
	}
	return c.writer.Write(resp.NewArray(vals))
}

func (c *cli) readResponse() (resp.Value, error) {
	return c.reader.Read()
}

func (c *cli) formatValue(val resp.Value) string {
	switch val.Type {
	case resp.SimpleString:
		return val.Str
	case resp.Error:
		return fmt.Sprintf("(error) %s", val.Str)
	case resp.Integer:
		return fmt.Sprintf("(integer) %d", val.Int)
	case resp.BulkString:
		if val.Null {
			return "(nil)"
		}
		return fmt.Sprintf("\"%s\"", val.Bulk)
	case resp.Array:
		if val.Null {
			return "(nil)"
		}
		var sb strings.Builder
		for i, elem := range val.Array {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("%d) %s", i+1, c.formatValue(elem)))
		}
		return sb.String()
	default:
		return fmt.Sprintf("(unknown type: %c)", val.Type)
	}
}

func (c *cli) runInteractive() {
	fmt.Printf("GoCache CLI v2.0.0\n")
	fmt.Printf("Connecting to %s:%d...\n", c.host, c.port)

	if err := c.connect(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer c.close()

	fmt.Printf("Connected. Type 'quit' or Ctrl+C to exit.\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("%s:%d> ", c.host, c.port)

		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.EqualFold(line, "quit") || strings.EqualFold(line, "exit") {
			c.sendCommand("QUIT", nil)
			break
		}

		parts := parseCommand(line)
		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToUpper(parts[0])
		args := parts[1:]

		if cmd == "HELP" {
			c.printHelp()
			continue
		}

		if err := c.sendCommand(cmd, args); err != nil {
			fmt.Printf("Error sending command: %v\n", err)
			if err := c.connect(); err != nil {
				fmt.Printf("Reconnection failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Reconnected.\n")
			continue
		}

		val, err := c.readResponse()
		if err != nil {
			if err == io.EOF {
				fmt.Printf("Connection closed by server.\n")
				break
			}
			fmt.Printf("Error reading response: %v\n", err)
			continue
		}

		fmt.Println(c.formatValue(val))
	}
}

func (c *cli) runSingleCommand(cmdLine string) {
	if err := c.connect(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer c.close()

	parts := parseCommand(cmdLine)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToUpper(parts[0])
	args := parts[1:]

	if err := c.sendCommand(cmd, args); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	val, err := c.readResponse()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(c.formatValue(val))
}

func (c *cli) printHelp() {
	fmt.Println("GoCache CLI - Available commands:")
	fmt.Println()

	categories := map[string][]string{
		"Connection":  {"PING", "ECHO", "SELECT", "QUIT", "INFO", "DBSIZE", "FLUSHDB", "FLUSHALL", "TIME"},
		"Key":         {"SET", "GET", "DEL", "EXISTS", "KEYS", "EXPIRE", "TTL", "PTTL", "PERSIST", "TYPE", "RENAME"},
		"String":      {"INCR", "DECR", "INCRBY", "DECRBY", "APPEND", "STRLEN", "MGET", "MSET", "SETNX", "SETEX"},
		"List":        {"LPUSH", "RPUSH", "LPOP", "RPOP", "LRANGE", "LLEN", "LTRIM"},
		"Hash":        {"HSET", "HGET", "HGETALL", "HDEL", "HEXISTS", "HLEN", "HKEYS", "HVALS"},
		"Set":         {"SADD", "SREM", "SISMEMBER", "SCARD", "SMEMBERS", "SPOP"},
		"Sorted Set":  {"ZADD", "ZREM", "ZSCORE", "ZRANGE", "ZREVRANGE", "ZRANK", "ZCARD", "ZPOPMIN", "ZPOPMAX"},
	}

	for cat, cmds := range categories {
		fmt.Printf("  %s:\n", cat)
		for _, cmd := range cmds {
			fmt.Printf("    %s\n", cmd)
		}
		fmt.Println()
	}

	fmt.Println("Type 'quit' or 'exit' to disconnect.")
}

func parseCommand(line string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	escape := false

	for _, ch := range line {
		if escape {
			current.WriteRune(ch)
			escape = false
			continue
		}

		if ch == '\\' {
			escape = true
			continue
		}

		if ch == '"' {
			inQuotes = !inQuotes
			continue
		}

		if ch == ' ' && !inQuotes {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(ch)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func main() {
	host := defaultHost
	port := defaultPort

	args := os.Args[1:]

	var cmdLine string
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-h", "--host":
			if i+1 < len(args) {
				host = args[i+1]
				i += 2
			} else {
				i++
			}
		case "-p", "--port":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &port)
				i += 2
			} else {
				i++
			}
		case "--help":
			fmt.Println("Usage: gocache-cli [options] [command]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  -h, --host <host>  Server hostname (default: 127.0.0.1)")
			fmt.Println("  -p, --port <port>  Server port (default: 6379)")
			fmt.Println("  --help             Show this help message")
			fmt.Println()
			fmt.Println("If a command is provided, it will be executed and the CLI will exit.")
			fmt.Println("Otherwise, an interactive REPL will start.")
			os.Exit(0)
		default:
			cmdLine = strings.Join(args[i:], " ")
			i = len(args)
		}
	}

	c := newCLI(host, port)

	if cmdLine != "" {
		c.runSingleCommand(cmdLine)
	} else {
		c.runInteractive()
	}
}
