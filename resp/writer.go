package resp

import (
	"fmt"
	"io"
	"strconv"
)

type Writer struct {
	writer io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

func (w *Writer) Write(v Value) error {
	switch v.Type {
	case SimpleString:
		return w.writeSimpleString(v.Str)
	case Error:
		return w.writeError(v.Str)
	case Integer:
		return w.writeInteger(v.Int)
	case BulkString:
		if v.Null {
			return w.writeNullBulkString()
		}
		return w.writeBulkString(v.Bulk)
	case Array:
		if v.Null {
			return w.writeNullArray()
		}
		return w.writeArray(v.Array)
	default:
		return fmt.Errorf("unknown RESP type: %c", v.Type)
	}
}

func (w *Writer) WriteSimpleString(s string) error {
	return w.writeSimpleString(s)
}

func (w *Writer) WriteError(s string) error {
	return w.writeError(s)
}

func (w *Writer) WriteInteger(n int64) error {
	return w.writeInteger(n)
}

func (w *Writer) WriteBulkString(s string) error {
	return w.writeBulkString(s)
}

func (w *Writer) WriteNullBulkString() error {
	return w.writeNullBulkString()
}

func (w *Writer) WriteArray(vals []Value) error {
	return w.writeArray(vals)
}

func (w *Writer) WriteNullArray() error {
	return w.writeNullArray()
}

func (w *Writer) WriteOK() error {
	return w.writeSimpleString("OK")
}

func (w *Writer) WritePong() error {
	return w.writeSimpleString("PONG")
}

func (w *Writer) WriteStringArray(items []string) error {
	vals := make([]Value, len(items))
	for i, item := range items {
		vals[i] = NewBulkString(item)
	}
	return w.writeArray(vals)
}

func (w *Writer) WriteStringMap(m map[string]string) error {
	vals := make([]Value, 0, len(m)*2)
	for k, v := range m {
		vals = append(vals, NewBulkString(k), NewBulkString(v))
	}
	return w.writeArray(vals)
}

func (w *Writer) writeSimpleString(s string) error {
	_, err := fmt.Fprintf(w.writer, "+%s\r\n", s)
	return err
}

func (w *Writer) writeError(s string) error {
	_, err := fmt.Fprintf(w.writer, "-%s\r\n", s)
	return err
}

func (w *Writer) writeInteger(n int64) error {
	_, err := fmt.Fprintf(w.writer, ":%d\r\n", n)
	return err
}

func (w *Writer) writeBulkString(s string) error {
	_, err := fmt.Fprintf(w.writer, "$%d\r\n%s\r\n", len(s), s)
	return err
}

func (w *Writer) writeNullBulkString() error {
	_, err := fmt.Fprintf(w.writer, "$-1\r\n")
	return err
}

func (w *Writer) writeArray(vals []Value) error {
	if _, err := fmt.Fprintf(w.writer, "*%d\r\n", len(vals)); err != nil {
		return err
	}
	for _, v := range vals {
		if err := w.Write(v); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) writeNullArray() error {
	_, err := fmt.Fprintf(w.writer, "*-1\r\n")
	return err
}

func Encode(v Value) ([]byte, error) {
	switch v.Type {
	case SimpleString:
		return []byte(fmt.Sprintf("+%s\r\n", v.Str)), nil
	case Error:
		return []byte(fmt.Sprintf("-%s\r\n", v.Str)), nil
	case Integer:
		return []byte(fmt.Sprintf(":%d\r\n", v.Int)), nil
	case BulkString:
		if v.Null {
			return []byte("$-1\r\n"), nil
		}
		return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(v.Bulk), v.Bulk)), nil
	case Array:
		if v.Null {
			return []byte("*-1\r\n"), nil
		}
		result := []byte(fmt.Sprintf("*%d\r\n", len(v.Array)))
		for _, elem := range v.Array {
			encoded, err := Encode(elem)
			if err != nil {
				return nil, err
			}
			result = append(result, encoded...)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unknown RESP type: %c", v.Type)
	}
}

func ParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func Atoi(s string) (int, error) {
	return strconv.Atoi(s)
}

func AtoiDefault(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
