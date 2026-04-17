package cache

import (
	"errors"
	"testing"
)

func TestTransactionManager_Begin(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	tx := tm.Begin()
	if tx == nil {
		t.Fatal("Begin should return non-nil transaction")
	}
	if tx.State != TxMulti {
		t.Errorf("expected TxMulti state, got %d", tx.State)
	}
	if len(tx.Commands) != 0 {
		t.Errorf("expected empty commands, got %d", len(tx.Commands))
	}
}

func TestTransactionManager_QueueCommand(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	tx := tm.Begin()

	err := tm.QueueCommand(tx, "SET", []string{"key", "value"})
	if err != nil {
		t.Fatalf("QueueCommand failed: %v", err)
	}

	if len(tx.Commands) != 1 {
		t.Fatalf("expected 1 queued command, got %d", len(tx.Commands))
	}

	if tx.Commands[0].Cmd != "SET" {
		t.Errorf("expected command SET, got %s", tx.Commands[0].Cmd)
	}
	if len(tx.Commands[0].Args) != 2 || tx.Commands[0].Args[0] != "key" || tx.Commands[0].Args[1] != "value" {
		t.Errorf("unexpected args: %v", tx.Commands[0].Args)
	}
}

func TestTransactionManager_QueueMultiple(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	tx := tm.Begin()

	tm.QueueCommand(tx, "SET", []string{"k1", "v1"})
	tm.QueueCommand(tx, "SET", []string{"k2", "v2"})
	tm.QueueCommand(tx, "GET", []string{"k1"})

	if len(tx.Commands) != 3 {
		t.Fatalf("expected 3 queued commands, got %d", len(tx.Commands))
	}
}

func TestTransactionManager_Exec(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	tx := tm.Begin()
	tm.QueueCommand(tx, "SET", []string{"key", "value"})

	executed := 0
	results, err := tm.Exec(tx, func(cmd string, args []string) error {
		executed++
		if cmd == "SET" {
			c.Set(args[0], args[1], 0)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if executed != 1 {
		t.Errorf("expected 1 execution, got %d", executed)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0] != nil {
		t.Errorf("expected nil error result, got %v", results[0])
	}

	val, found := c.Get("key")
	if !found || val != "value" {
		t.Errorf("expected key=value, found=%v val=%v", found, val)
	}
}

func TestTransactionManager_ExecMultiple(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	tx := tm.Begin()
	tm.QueueCommand(tx, "SET", []string{"k1", "v1"})
	tm.QueueCommand(tx, "SET", []string{"k2", "v2"})
	tm.QueueCommand(tx, "DEL", []string{"k1"})

	results, err := tm.Exec(tx, func(cmd string, args []string) error {
		switch cmd {
		case "SET":
			c.Set(args[0], args[1], 0)
		case "DEL":
			c.Delete(args[0])
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if c.Exists("k1") {
		t.Error("k1 should have been deleted")
	}
	if !c.Exists("k2") {
		t.Error("k2 should exist")
	}
}

func TestTransactionManager_ExecWithoutMulti(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	tx := &Transaction{State: TxNone}
	_, err := tm.Exec(tx, func(cmd string, args []string) error {
		return nil
	})

	if err == nil {
		t.Error("expected error for EXEC without MULTI")
	}
}

func TestTransactionManager_Discard(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	tx := tm.Begin()
	tm.QueueCommand(tx, "SET", []string{"key", "value"})

	tm.Discard(tx)

	if tx.State != TxNone {
		t.Errorf("expected TxNone state after discard, got %d", tx.State)
	}
	if len(tx.Commands) != 0 {
		t.Errorf("expected empty commands after discard, got %d", len(tx.Commands))
	}
}

func TestTransactionManager_Watch(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	c.Set("watched_key", "value", 0)

	tx := &Transaction{}
	err := tm.Watch(tx, "watched_key")
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	if len(tx.WatchKeys) != 1 {
		t.Fatalf("expected 1 watch key, got %d", len(tx.WatchKeys))
	}
	if tx.WatchKeys[0].Key != "watched_key" {
		t.Errorf("expected watched_key, got %s", tx.WatchKeys[0].Key)
	}
}

func TestTransactionManager_WatchInsideMulti(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	tx := tm.Begin()
	err := tm.Watch(tx, "key")
	if err == nil {
		t.Error("expected error for WATCH inside MULTI")
	}
}

func TestTransactionManager_WatchConflict(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	c.Set("key", "v1", 0)

	tx := &Transaction{}
	tm.Watch(tx, "key")

	tm.NotifyKeyChange("key")

	tx.State = TxMulti
	tm.QueueCommand(tx, "SET", []string{"key", "v2"})

	results, err := tm.Exec(tx, func(cmd string, args []string) error {
		return nil
	})

	if err != nil {
		t.Fatalf("Exec should not fail: %v", err)
	}
	if results != nil {
		t.Error("expected nil results (WATCH conflict abort), got non-nil")
	}
}

func TestTransactionManager_WatchNoConflict(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	c.Set("key", "v1", 0)

	tx := &Transaction{}
	tm.Watch(tx, "key")

	tx.State = TxMulti
	tm.QueueCommand(tx, "SET", []string{"key", "v2"})

	results, err := tm.Exec(tx, func(cmd string, args []string) error {
		c.Set(args[0], args[1], 0)
		return nil
	})

	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if results == nil {
		t.Error("expected results (no WATCH conflict), got nil")
	}
}

func TestTransactionManager_Unwatch(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	tx := &Transaction{}
	tm.Watch(tx, "key1", "key2")

	if len(tx.WatchKeys) != 2 {
		t.Fatalf("expected 2 watch keys, got %d", len(tx.WatchKeys))
	}

	tm.Unwatch(tx)
	if len(tx.WatchKeys) != 0 {
		t.Errorf("expected 0 watch keys after unwatch, got %d", len(tx.WatchKeys))
	}
}

func TestTransactionManager_ExecWithPartialFailure(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	tx := tm.Begin()
	tm.QueueCommand(tx, "SET", []string{"k1", "v1"})
	tm.QueueCommand(tx, "INVALID", []string{})
	tm.QueueCommand(tx, "SET", []string{"k2", "v2"})

	errCmd := errors.New("unknown command")
	results, err := tm.Exec(tx, func(cmd string, args []string) error {
		switch cmd {
		case "SET":
			c.Set(args[0], args[1], 0)
			return nil
		default:
			return errCmd
		}
	})

	if err != nil {
		t.Fatalf("Exec should not fail overall: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0] != nil {
		t.Errorf("first command should succeed, got error: %v", results[0])
	}
	if results[1] == nil {
		t.Error("second command should fail")
	}
	if results[2] != nil {
		t.Errorf("third command should succeed, got error: %v", results[2])
	}
}

func TestTransactionManager_WatchMultipleKeys(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	c.Set("k1", "v1", 0)
	c.Set("k2", "v2", 0)
	c.Set("k3", "v3", 0)

	tx := &Transaction{}
	tm.Watch(tx, "k1", "k2", "k3")

	if len(tx.WatchKeys) != 3 {
		t.Fatalf("expected 3 watch keys, got %d", len(tx.WatchKeys))
	}
}

func TestTransactionManager_DiscardWithoutMulti(t *testing.T) {
	c := New()
	tm := NewTransactionManager(c)

	tx := &Transaction{State: TxNone}
	tm.Discard(tx)

	if tx.State != TxNone {
		t.Errorf("state should remain TxNone, got %d", tx.State)
	}
}
