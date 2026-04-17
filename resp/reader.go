package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

const (
	SimpleString = '+'
	Error        = '-'
	Integer      = ':'
	BulkString   = '$'
	Array        = '*'
)

type Value struct {
	Type    byte
	Str     string
	Int     int64
	Bulk    string
	Array   []Value
	Null    bool
}

var NullValue = Value{Type: BulkString, Null: true}

func NewSimpleString(s string) Value {
	return Value{Type: SimpleString, Str: s}
}

func NewError(s string) Value {
	return Value{Type: Error, Str: s}
}

func NewInteger(n int64) Value {
	return Value{Type: Integer, Int: n}
}

func NewBulkString(s string) Value {
	return Value{Type: BulkString, Bulk: s}
}

func NewArray(vals []Value) Value {
	return Value{Type: Array, Array: vals}
}

func NewNullBulkString() Value {
	return NullValue
}

func NewNullArray() Value {
	return Value{Type: Array, Null: true}
}

type Reader struct {
	reader *bufio.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		reader: bufio.NewReader(r),
	}
}

func (r *Reader) Read() (Value, error) {
	line, err := r.readLine()
	if err != nil {
		return Value{}, err
	}

	if len(line) == 0 {
		return Value{}, fmt.Errorf("empty line")
	}

	switch line[0] {
	case SimpleString:
		return Value{Type: SimpleString, Str: string(line[1:])}, nil
	case Error:
		return Value{Type: Error, Str: string(line[1:])}, nil
	case Integer:
		n, err := strconv.ParseInt(string(line[1:]), 10, 64)
		if err != nil {
			return Value{}, fmt.Errorf("invalid integer: %w", err)
		}
		return Value{Type: Integer, Int: n}, nil
	case BulkString:
		return r.readBulkString(line[1:])
	case Array:
		return r.readArray(line[1:])
	default:
		return Value{}, fmt.Errorf("unknown RESP type: %c", line[0])
	}
}

func (r *Reader) readLine() ([]byte, error) {
	line, err := r.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	if len(line) >= 2 && line[len(line)-2] == '\r' {
		line = line[:len(line)-2]
	} else if len(line) >= 1 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}

	return line, nil
}

func (r *Reader) readBulkString(header []byte) (Value, error) {
	length, err := strconv.ParseInt(string(header), 10, 64)
	if err != nil {
		return Value{}, fmt.Errorf("invalid bulk string length: %w", err)
	}

	if length == -1 {
		return NullValue, nil
	}

	data := make([]byte, length+2)
	if _, err := io.ReadFull(r.reader, data); err != nil {
		return Value{}, fmt.Errorf("failed to read bulk string data: %w", err)
	}

	return Value{Type: BulkString, Bulk: string(data[:length])}, nil
}

func (r *Reader) readArray(header []byte) (Value, error) {
	count, err := strconv.ParseInt(string(header), 10, 64)
	if err != nil {
		return Value{}, fmt.Errorf("invalid array length: %w", err)
	}

	if count == -1 {
		return NewNullArray(), nil
	}

	if count < 0 {
		return Value{}, fmt.Errorf("invalid array count: %d", count)
	}

	vals := make([]Value, count)
	for i := int64(0); i < count; i++ {
		val, err := r.Read()
		if err != nil {
			return Value{}, fmt.Errorf("failed to read array element %d: %w", i, err)
		}
		vals[i] = val
	}

	return Value{Type: Array, Array: vals}, nil
}

func (r *Reader) ReadCommand() (string, []string, error) {
	val, err := r.Read()
	if err != nil {
		return "", nil, err
	}

	if val.Type != Array {
		return "", nil, fmt.Errorf("expected array for command, got %c", val.Type)
	}

	if len(val.Array) == 0 {
		return "", nil, fmt.Errorf("empty command array")
	}

	cmd := val.Array[0]
	var command string

	switch cmd.Type {
	case BulkString:
		command = cmd.Bulk
	case SimpleString:
		command = cmd.Str
	default:
		return "", nil, fmt.Errorf("invalid command type: %c", cmd.Type)
	}

	args := make([]string, 0, len(val.Array)-1)
	for i := 1; i < len(val.Array); i++ {
		var arg string
		switch val.Array[i].Type {
		case BulkString:
			arg = val.Array[i].Bulk
		case SimpleString:
			arg = val.Array[i].Str
		case Integer:
			arg = strconv.FormatInt(val.Array[i].Int, 10)
		default:
			arg = ""
		}
		args = append(args, arg)
	}

	return command, args, nil
}
