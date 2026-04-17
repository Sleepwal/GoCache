package server

import (
	"fmt"
	"net"
	"testing"
	"time"

	"GoCache/resp"
)

func startTestTCPServer(t *testing.T) (*TCPServer, int) {
	t.Helper()

	ts := NewTCPServer(TCPServerConfig{Port: 0})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	ts.listener = ln
	ts.running = true
	ts.startTime = time.Now()

	port := ln.Addr().(*net.TCPAddr).Port

	go func() {
		for ts.running {
			conn, err := ln.Accept()
			if err != nil {
				if !ts.running {
					return
				}
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

			go ts.handleConnection(c)
		}
	}()

	return ts, port
}

func connectToServer(t *testing.T, port int) (net.Conn, *resp.Reader, *resp.Writer) {
	t.Helper()

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	return conn, resp.NewReader(conn), resp.NewWriter(conn)
}

func TestTCPServerPing(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("PING"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.SimpleString || val.Str != "PONG" {
		t.Errorf("expected +PONG, got %c%s", val.Type, val.Str)
	}
}

func TestTCPServerPingWithMessage(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("PING"),
		resp.NewBulkString("hello"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.SimpleString || val.Str != "hello" {
		t.Errorf("expected +hello, got %c%s", val.Type, val.Str)
	}
}

func TestTCPServerEcho(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("ECHO"),
		resp.NewBulkString("test message"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.BulkString || val.Bulk != "test message" {
		t.Errorf("expected $12\r\ntest message, got %c%s", val.Type, val.Bulk)
	}
}

func TestTCPServerSetGet(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("SET"),
		resp.NewBulkString("mykey"),
		resp.NewBulkString("myvalue"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.SimpleString || val.Str != "OK" {
		t.Errorf("expected +OK, got %c%s", val.Type, val.Str)
	}

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("GET"),
		resp.NewBulkString("mykey"),
	}))

	val, err = reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.BulkString || val.Bulk != "myvalue" {
		t.Errorf("expected $7\r\nmyvalue, got %c%s", val.Type, val.Bulk)
	}
}

func TestTCPServerSetWithEX(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("SET"),
		resp.NewBulkString("expkey"),
		resp.NewBulkString("expvalue"),
		resp.NewBulkString("EX"),
		resp.NewBulkString("100"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.SimpleString || val.Str != "OK" {
		t.Errorf("expected +OK, got %c%s", val.Type, val.Str)
	}
}

func TestTCPServerGetNonexistent(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("GET"),
		resp.NewBulkString("nonexistent"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !val.Null {
		t.Errorf("expected null bulk string, got %c%s", val.Type, val.Bulk)
	}
}

func TestTCPServerDel(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("SET"),
		resp.NewBulkString("delkey"),
		resp.NewBulkString("delvalue"),
	}))
	reader.Read()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("DEL"),
		resp.NewBulkString("delkey"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.Integer || val.Int != 1 {
		t.Errorf("expected :1, got %c%d", val.Type, val.Int)
	}
}

func TestTCPServerExists(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("SET"),
		resp.NewBulkString("existkey"),
		resp.NewBulkString("value"),
	}))
	reader.Read()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("EXISTS"),
		resp.NewBulkString("existkey"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.Integer || val.Int != 1 {
		t.Errorf("expected :1, got %c%d", val.Type, val.Int)
	}

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("EXISTS"),
		resp.NewBulkString("nokey"),
	}))

	val, _ = reader.Read()
	if val.Type != resp.Integer || val.Int != 0 {
		t.Errorf("expected :0, got %c%d", val.Type, val.Int)
	}
}

func TestTCPServerDBSize(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("SET"),
		resp.NewBulkString("k1"),
		resp.NewBulkString("v1"),
	}))
	reader.Read()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("DBSIZE"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.Integer || val.Int < 1 {
		t.Errorf("expected :>=1, got %c%d", val.Type, val.Int)
	}
}

func TestTCPServerUnknownCommand(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("UNKNOWNCMD"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.Error {
		t.Errorf("expected error response, got %c", val.Type)
	}
}

func TestTCPServerHSetHGet(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("HSET"),
		resp.NewBulkString("myhash"),
		resp.NewBulkString("field1"),
		resp.NewBulkString("value1"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.Integer || val.Int != 1 {
		t.Errorf("expected :1, got %c%d", val.Type, val.Int)
	}

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("HGET"),
		resp.NewBulkString("myhash"),
		resp.NewBulkString("field1"),
	}))

	val, _ = reader.Read()
	if val.Type != resp.BulkString || val.Bulk != "value1" {
		t.Errorf("expected 'value1', got '%s'", val.Bulk)
	}
}

func TestTCPServerLPushRPush(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("LPUSH"),
		resp.NewBulkString("mylist"),
		resp.NewBulkString("a"),
		resp.NewBulkString("b"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.Integer || val.Int != 2 {
		t.Errorf("expected :2, got %c%d", val.Type, val.Int)
	}

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("LRANGE"),
		resp.NewBulkString("mylist"),
		resp.NewBulkString("0"),
		resp.NewBulkString("-1"),
	}))

	val, _ = reader.Read()
	if val.Type != resp.Array || len(val.Array) != 2 {
		t.Errorf("expected 2 elements, got %d", len(val.Array))
	}
}

func TestTCPServerSAddSMembers(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("SADD"),
		resp.NewBulkString("myset"),
		resp.NewBulkString("member1"),
		resp.NewBulkString("member2"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.Integer || val.Int != 2 {
		t.Errorf("expected :2, got %c%d", val.Type, val.Int)
	}

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("SMEMBERS"),
		resp.NewBulkString("myset"),
	}))

	val, _ = reader.Read()
	if val.Type != resp.Array || len(val.Array) != 2 {
		t.Errorf("expected 2 elements, got %d", len(val.Array))
	}
}

func TestTCPServerZAddZRange(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("ZADD"),
		resp.NewBulkString("myzset"),
		resp.NewBulkString("1"),
		resp.NewBulkString("a"),
		resp.NewBulkString("2"),
		resp.NewBulkString("b"),
		resp.NewBulkString("3"),
		resp.NewBulkString("c"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.Integer || val.Int != 3 {
		t.Errorf("expected :3, got %c%d", val.Type, val.Int)
	}

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("ZRANGE"),
		resp.NewBulkString("myzset"),
		resp.NewBulkString("0"),
		resp.NewBulkString("-1"),
	}))

	val, _ = reader.Read()
	if val.Type != resp.Array || len(val.Array) != 3 {
		t.Errorf("expected 3 elements, got %d", len(val.Array))
	}

	if val.Array[0].Bulk != "a" {
		t.Errorf("expected first element 'a', got '%s'", val.Array[0].Bulk)
	}
}

func TestTCPServerZScore(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("ZADD"),
		resp.NewBulkString("zscore_test"),
		resp.NewBulkString("1.5"),
		resp.NewBulkString("member1"),
	}))
	reader.Read()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("ZSCORE"),
		resp.NewBulkString("zscore_test"),
		resp.NewBulkString("member1"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.BulkString {
		t.Errorf("expected bulk string, got %c", val.Type)
	}
}

func TestTCPServerType(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("SET"),
		resp.NewBulkString("strkey"),
		resp.NewBulkString("value"),
	}))
	reader.Read()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("TYPE"),
		resp.NewBulkString("strkey"),
	}))
	val, _ := reader.Read()
	if val.Str != "string" {
		t.Errorf("expected 'string', got '%s'", val.Str)
	}

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("TYPE"),
		resp.NewBulkString("nokey"),
	}))
	val, _ = reader.Read()
	if val.Str != "none" {
		t.Errorf("expected 'none', got '%s'", val.Str)
	}
}

func TestTCPServerFlushDB(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("SET"),
		resp.NewBulkString("k1"),
		resp.NewBulkString("v1"),
	}))
	reader.Read()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("FLUSHDB"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != resp.SimpleString || val.Str != "OK" {
		t.Errorf("expected +OK, got %c%s", val.Type, val.Str)
	}

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("DBSIZE"),
	}))
	val, _ = reader.Read()
	if val.Int != 0 {
		t.Errorf("expected 0 after flush, got %d", val.Int)
	}
}

func TestTCPServerIncrDecr(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("SET"),
		resp.NewBulkString("counter"),
		resp.NewBulkString("10"),
	}))
	reader.Read()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("INCR"),
		resp.NewBulkString("counter"),
	}))
	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.Int != 11 {
		t.Errorf("expected 11, got %d", val.Int)
	}

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("DECR"),
		resp.NewBulkString("counter"),
	}))
	val, _ = reader.Read()
	if val.Int != 10 {
		t.Errorf("expected 10, got %d", val.Int)
	}
}

func TestTCPServerMSetMGet(t *testing.T) {
	ts, port := startTestTCPServer(t)
	defer ts.Stop()

	conn, reader, writer := connectToServer(t, port)
	defer conn.Close()

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("MSET"),
		resp.NewBulkString("k1"),
		resp.NewBulkString("v1"),
		resp.NewBulkString("k2"),
		resp.NewBulkString("v2"),
	}))

	val, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.Str != "OK" {
		t.Errorf("expected OK, got %s", val.Str)
	}

	writer.Write(resp.NewArray([]resp.Value{
		resp.NewBulkString("MGET"),
		resp.NewBulkString("k1"),
		resp.NewBulkString("k2"),
		resp.NewBulkString("k3"),
	}))

	val, _ = reader.Read()
	if val.Type != resp.Array || len(val.Array) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(val.Array))
	}
	if val.Array[0].Bulk != "v1" {
		t.Errorf("expected v1, got %s", val.Array[0].Bulk)
	}
	if val.Array[1].Bulk != "v2" {
		t.Errorf("expected v2, got %s", val.Array[1].Bulk)
	}
	if !val.Array[2].Null {
		t.Errorf("expected null for k3")
	}
}
