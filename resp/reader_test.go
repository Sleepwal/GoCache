package resp

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadSimpleString(t *testing.T) {
	input := "+OK\r\n"
	r := NewReader(strings.NewReader(input))

	val, err := r.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != SimpleString {
		t.Errorf("expected type %c, got %c", SimpleString, val.Type)
	}

	if val.Str != "OK" {
		t.Errorf("expected 'OK', got '%s'", val.Str)
	}
}

func TestReadError(t *testing.T) {
	input := "-ERR unknown command\r\n"
	r := NewReader(strings.NewReader(input))

	val, err := r.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != Error {
		t.Errorf("expected type %c, got %c", Error, val.Type)
	}

	if val.Str != "ERR unknown command" {
		t.Errorf("expected 'ERR unknown command', got '%s'", val.Str)
	}
}

func TestReadInteger(t *testing.T) {
	input := ":1000\r\n"
	r := NewReader(strings.NewReader(input))

	val, err := r.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != Integer {
		t.Errorf("expected type %c, got %c", Integer, val.Type)
	}

	if val.Int != 1000 {
		t.Errorf("expected 1000, got %d", val.Int)
	}
}

func TestReadBulkString(t *testing.T) {
	input := "$6\r\nfoobar\r\n"
	r := NewReader(strings.NewReader(input))

	val, err := r.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != BulkString {
		t.Errorf("expected type %c, got %c", BulkString, val.Type)
	}

	if val.Bulk != "foobar" {
		t.Errorf("expected 'foobar', got '%s'", val.Bulk)
	}
}

func TestReadNullBulkString(t *testing.T) {
	input := "$-1\r\n"
	r := NewReader(strings.NewReader(input))

	val, err := r.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !val.Null {
		t.Error("expected null value")
	}
}

func TestReadArray(t *testing.T) {
	input := "*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	r := NewReader(strings.NewReader(input))

	val, err := r.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != Array {
		t.Errorf("expected type %c, got %c", Array, val.Type)
	}

	if len(val.Array) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(val.Array))
	}

	if val.Array[0].Bulk != "foo" {
		t.Errorf("expected 'foo', got '%s'", val.Array[0].Bulk)
	}

	if val.Array[1].Bulk != "bar" {
		t.Errorf("expected 'bar', got '%s'", val.Array[1].Bulk)
	}
}

func TestReadNullArray(t *testing.T) {
	input := "*-1\r\n"
	r := NewReader(strings.NewReader(input))

	val, err := r.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !val.Null {
		t.Error("expected null array")
	}
}

func TestReadCommand(t *testing.T) {
	input := "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"
	r := NewReader(strings.NewReader(input))

	cmd, args, err := r.ReadCommand()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd != "SET" {
		t.Errorf("expected 'SET', got '%s'", cmd)
	}

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}

	if args[0] != "key" {
		t.Errorf("expected 'key', got '%s'", args[0])
	}

	if args[1] != "value" {
		t.Errorf("expected 'value', got '%s'", args[1])
	}
}

func TestReadMultipleCommands(t *testing.T) {
	input := "*1\r\n$4\r\nPING\r\n*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"
	r := NewReader(strings.NewReader(input))

	cmd, args, err := r.ReadCommand()
	if err != nil {
		t.Fatalf("unexpected error on first command: %v", err)
	}
	if cmd != "PING" || len(args) != 0 {
		t.Errorf("expected PING with 0 args, got %s with %d args", cmd, len(args))
	}

	cmd, args, err = r.ReadCommand()
	if err != nil {
		t.Fatalf("unexpected error on second command: %v", err)
	}
	if cmd != "SET" || len(args) != 2 {
		t.Errorf("expected SET with 2 args, got %s with %d args", cmd, len(args))
	}
}

func TestWriteSimpleString(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteSimpleString("OK")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "+OK\r\n" {
		t.Errorf("expected '+OK\\r\\n', got '%s'", buf.String())
	}
}

func TestWriteError(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteError("ERR unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "-ERR unknown\r\n" {
		t.Errorf("expected '-ERR unknown\\r\\n', got '%s'", buf.String())
	}
}

func TestWriteInteger(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteInteger(1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != ":1000\r\n" {
		t.Errorf("expected ':1000\\r\\n', got '%s'", buf.String())
	}
}

func TestWriteBulkString(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteBulkString("foobar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "$6\r\nfoobar\r\n" {
		t.Errorf("expected '$6\\r\\nfoobar\\r\\n', got '%s'", buf.String())
	}
}

func TestWriteNullBulkString(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteNullBulkString()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "$-1\r\n" {
		t.Errorf("expected '$-1\\r\\n', got '%s'", buf.String())
	}
}

func TestWriteArray(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteArray([]Value{
		NewBulkString("foo"),
		NewBulkString("bar"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	if buf.String() != expected {
		t.Errorf("expected '%s', got '%s'", expected, buf.String())
	}
}

func TestWriteOK(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteOK()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "+OK\r\n" {
		t.Errorf("expected '+OK\\r\\n', got '%s'", buf.String())
	}
}

func TestWritePong(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WritePong()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "+PONG\r\n" {
		t.Errorf("expected '+PONG\\r\\n', got '%s'", buf.String())
	}
}

func TestWriteStringArray(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteStringArray([]string{"key1", "key2", "key3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "*3\r\n$4\r\nkey1\r\n$4\r\nkey2\r\n$4\r\nkey3\r\n"
	if buf.String() != expected {
		t.Errorf("expected '%s', got '%s'", expected, buf.String())
	}
}

func TestRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	w.Write(NewSimpleString("OK"))
	w.Write(NewError("ERR test"))
	w.Write(NewInteger(42))
	w.Write(NewBulkString("hello"))
	w.Write(NullValue)
	w.Write(NewArray([]Value{NewBulkString("a"), NewInteger(1)}))

	r := NewReader(&buf)

	val, _ := r.Read()
	if val.Type != SimpleString || val.Str != "OK" {
		t.Errorf("simple string roundtrip failed")
	}

	val, _ = r.Read()
	if val.Type != Error || val.Str != "ERR test" {
		t.Errorf("error roundtrip failed")
	}

	val, _ = r.Read()
	if val.Type != Integer || val.Int != 42 {
		t.Errorf("integer roundtrip failed")
	}

	val, _ = r.Read()
	if val.Type != BulkString || val.Bulk != "hello" {
		t.Errorf("bulk string roundtrip failed")
	}

	val, _ = r.Read()
	if !val.Null {
		t.Errorf("null bulk string roundtrip failed")
	}

	val, _ = r.Read()
	if val.Type != Array || len(val.Array) != 2 {
		t.Errorf("array roundtrip failed")
	}
}

func TestEncode(t *testing.T) {
	encoded, err := Encode(NewSimpleString("OK"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(encoded) != "+OK\r\n" {
		t.Errorf("expected '+OK\\r\\n', got '%s'", string(encoded))
	}

	encoded, err = Encode(NewBulkString("test"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(encoded) != "$4\r\ntest\r\n" {
		t.Errorf("expected '$4\\r\\ntest\\r\\n', got '%s'", string(encoded))
	}

	encoded, err = Encode(NullValue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(encoded) != "$-1\r\n" {
		t.Errorf("expected '$-1\\r\\n', got '%s'", string(encoded))
	}
}

func TestReadEmptyBulkString(t *testing.T) {
	input := "$0\r\n\r\n"
	r := NewReader(strings.NewReader(input))

	val, err := r.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != BulkString {
		t.Errorf("expected BulkString, got %c", val.Type)
	}

	if val.Bulk != "" {
		t.Errorf("expected empty string, got '%s'", val.Bulk)
	}
}

func TestReadEmptyArray(t *testing.T) {
	input := "*0\r\n"
	r := NewReader(strings.NewReader(input))

	val, err := r.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Type != Array {
		t.Errorf("expected Array, got %c", val.Type)
	}

	if len(val.Array) != 0 {
		t.Errorf("expected 0 elements, got %d", len(val.Array))
	}
}

func TestReadNegativeInteger(t *testing.T) {
	input := ":-100\r\n"
	r := NewReader(strings.NewReader(input))

	val, err := r.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Int != -100 {
		t.Errorf("expected -100, got %d", val.Int)
	}
}
