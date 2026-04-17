package cache

import (
	"testing"
)

func TestScript_Eval(t *testing.T) {
	c := New()
	se := NewScriptEngine(c)

	result, err := se.Eval("SET $0 $1", 1, []string{"mykey", "myvalue"})
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	if result != "OK" {
		t.Errorf("expected OK, got %v", result)
	}

	val, found := c.Get("mykey")
	if !found || val != "myvalue" {
		t.Errorf("expected mykey=myvalue, found=%v val=%v", found, val)
	}
}

func TestScript_EvalGet(t *testing.T) {
	c := New()
	c.Set("key1", "value1", 0)
	se := NewScriptEngine(c)

	result, err := se.Eval("GET $0", 1, []string{"key1"})
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	if result != "value1" {
		t.Errorf("expected value1, got %v", result)
	}
}

func TestScript_EvalDel(t *testing.T) {
	c := New()
	c.Set("key1", "value1", 0)
	se := NewScriptEngine(c)

	result, err := se.Eval("DEL $0", 1, []string{"key1"})
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	if result != 1 {
		t.Errorf("expected 1 deleted, got %v", result)
	}
	if c.Exists("key1") {
		t.Error("key1 should have been deleted")
	}
}

func TestScript_EvalMultiLine(t *testing.T) {
	c := New()
	se := NewScriptEngine(c)

	_, err := se.Eval("SET $0 $1; SET $2 $3", 2, []string{"k1", "v1", "k2", "v2"})
	if err != nil {
		t.Fatalf("Eval multi-line failed: %v", err)
	}

	if !c.Exists("k1") || !c.Exists("k2") {
		t.Error("expected both keys to exist")
	}
}

func TestScript_ScriptLoad(t *testing.T) {
	c := New()
	se := NewScriptEngine(c)

	sha, err := se.ScriptLoad("SET $0 $1")
	if err != nil {
		t.Fatalf("ScriptLoad failed: %v", err)
	}
	if len(sha) != 40 {
		t.Errorf("expected 40-char SHA1, got %d chars: %s", len(sha), sha)
	}
}

func TestScript_EvalSHA(t *testing.T) {
	c := New()
	se := NewScriptEngine(c)

	script := "SET $0 $1"
	sha, _ := se.ScriptLoad(script)

	result, err := se.EvalSHA(sha, 1, []string{"key", "value"})
	if err != nil {
		t.Fatalf("EvalSHA failed: %v", err)
	}
	if result != "OK" {
		t.Errorf("expected OK, got %v", result)
	}
}

func TestScript_EvalSHANotFound(t *testing.T) {
	c := New()
	se := NewScriptEngine(c)

	_, err := se.EvalSHA("nonexistent_sha", 0, nil)
	if err == nil {
		t.Error("expected error for non-existent SHA")
	}
}

func TestScript_ScriptExists(t *testing.T) {
	c := New()
	se := NewScriptEngine(c)

	sha, _ := se.ScriptLoad("SET $0 $1")

	exists := se.ScriptExists(sha, "nonexistent")
	if !exists[0] {
		t.Error("expected loaded script to exist")
	}
	if exists[1] {
		t.Error("expected non-existent script to not exist")
	}
}

func TestScript_ScriptFlush(t *testing.T) {
	c := New()
	se := NewScriptEngine(c)

	sha, _ := se.ScriptLoad("SET $0 $1")
	se.ScriptFlush()

	exists := se.ScriptExists(sha)
	if exists[0] {
		t.Error("expected script to not exist after flush")
	}
}

func TestScript_ScriptCount(t *testing.T) {
	c := New()
	se := NewScriptEngine(c)

	se.ScriptLoad("SET $0 $1")
	se.ScriptLoad("GET $0")

	if se.ScriptCount() != 2 {
		t.Errorf("expected 2 scripts, got %d", se.ScriptCount())
	}
}

func TestScript_UnknownCommand(t *testing.T) {
	c := New()
	se := NewScriptEngine(c)

	_, err := se.Eval("UNKNOWNCMD arg1", 0, nil)
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestScript_ExistsCommand(t *testing.T) {
	c := New()
	c.Set("key1", "value1", 0)
	se := NewScriptEngine(c)

	result, err := se.Eval("EXISTS $0", 1, []string{"key1"})
	if err != nil {
		t.Fatalf("Eval EXISTS failed: %v", err)
	}
	if result != 1 {
		t.Errorf("expected 1, got %v", result)
	}
}
