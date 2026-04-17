package cache

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"GoCache/logger"
)

type TransactionState int

const (
	TxNone TransactionState = iota
	TxMulti
)

type QueuedCommand struct {
	Cmd  string
	Args []string
}

type WatchKey struct {
	Key       string
	Version   uint64
}

type Transaction struct {
	State     TransactionState
	Commands  []QueuedCommand
	WatchKeys []WatchKey
}

type TransactionManager struct {
	cache     *MemoryCache
	globalVer atomic.Uint64
	mu        sync.Mutex
}

func NewTransactionManager(c *MemoryCache) *TransactionManager {
	return &TransactionManager{
		cache: c,
	}
}

func (tm *TransactionManager) Begin() *Transaction {
	return &Transaction{
		State:    TxMulti,
		Commands: make([]QueuedCommand, 0),
	}
}

func (tm *TransactionManager) QueueCommand(tx *Transaction, cmd string, args []string) error {
	if tx.State != TxMulti {
		return fmt.Errorf("MULTI calls can not be nested")
	}

	tx.Commands = append(tx.Commands, QueuedCommand{Cmd: cmd, Args: args})
	return nil
}

func (tm *TransactionManager) Watch(tx *Transaction, keys ...string) error {
	if tx.State == TxMulti {
		return fmt.Errorf("WATCH inside MULTI is not allowed")
	}

	tm.cache.mu.RLock()
	defer tm.cache.mu.RUnlock()

	for _, key := range keys {
		version := tm.globalVer.Load()
		if item, found := tm.cache.items[key]; found {
			if item.Expiration > 0 && time.Now().UnixNano() > item.Expiration {
				continue
			}
		}

		tx.WatchKeys = append(tx.WatchKeys, WatchKey{
			Key:     key,
			Version: version,
		})
	}

	logger.Debug("WATCH registered", "keys", keys, "watch_count", len(tx.WatchKeys))
	return nil
}

func (tm *TransactionManager) checkWatchConflict(tx *Transaction) bool {
	if len(tx.WatchKeys) == 0 {
		return false
	}

	currentVer := tm.globalVer.Load()
	for _, wk := range tx.WatchKeys {
		if currentVer > wk.Version {
			logger.Warn("WATCH conflict detected", "key", wk.Key, "watched_version", wk.Version, "current_version", currentVer)
			return true
		}
	}

	return false
}

func (tm *TransactionManager) bumpVersion() {
	tm.globalVer.Add(1)
}

func (tm *TransactionManager) Exec(tx *Transaction, executor func(cmd string, args []string) error) ([]error, error) {
	if tx.State != TxMulti {
		return nil, fmt.Errorf("EXEC without MULTI")
	}

	if tm.checkWatchConflict(tx) {
		tx.State = TxNone
		tx.Commands = nil
		tx.WatchKeys = nil
		logger.Warn("transaction aborted due to WATCH conflict")
		return nil, nil
	}

	results := make([]error, len(tx.Commands))

	tm.mu.Lock()
	defer tm.mu.Unlock()

	logger.Info("executing transaction", "commands", len(tx.Commands))

	for i, cmd := range tx.Commands {
		if err := executor(cmd.Cmd, cmd.Args); err != nil {
			results[i] = err
			logger.Warn("transaction command failed", "index", i, "command", cmd.Cmd, "error", err)
		} else {
			results[i] = nil
		}
		tm.bumpVersion()
	}

	tx.State = TxNone
	tx.Commands = nil
	tx.WatchKeys = nil

	return results, nil
}

func (tm *TransactionManager) Discard(tx *Transaction) {
	if tx.State != TxMulti {
		return
	}

	logger.Debug("transaction discarded", "queued_commands", len(tx.Commands))
	tx.State = TxNone
	tx.Commands = nil
	tx.WatchKeys = nil
}

func (tm *TransactionManager) Unwatch(tx *Transaction) {
	tx.WatchKeys = nil
	logger.Debug("WATCH cleared")
}

func (tm *TransactionManager) NotifyKeyChange(keys ...string) {
	tm.bumpVersion()
}
