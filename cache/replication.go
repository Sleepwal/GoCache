package cache

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"GoCache/logger"
	"GoCache/resp"
)

const (
	ReplicaRoleSlave    = "slave"
	ReplicaRoleMaster   = "master"
	replicationProtocol = "GOCACHE_REPL"
)

type ReplicationRole string

type ReplicaInfo struct {
	Address    string
	Connected  bool
	LastSync   time.Time
	Offset     int64
	Connection net.Conn
}

type ReplicationManager struct {
	cache      *MemoryCache
	role       atomic.Value
	masterAddr string
	masterPort int

	mu             sync.RWMutex
	replicas       map[string]*ReplicaInfo
	replicationLog []ReplicationEntry
	logOffset      int64

	stopCh  chan struct{}
	running bool
	conn    net.Conn

	replID string
	offset int64
}

type ReplicationEntry struct {
	Offset  int64    `json:"offset"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type FullSyncData struct {
	Items  map[string]*SyncItem `json:"items"`
	Offset int64                `json:"offset"`
	ReplID string               `json:"repl_id"`
}

type SyncItem struct {
	Value      any   `json:"value"`
	Expiration int64 `json:"expiration"`
}

func NewReplicationManager(cache *MemoryCache) *ReplicationManager {
	rm := &ReplicationManager{
		cache:    cache,
		replicas: make(map[string]*ReplicaInfo),
		stopCh:   make(chan struct{}),
		replID:   fmt.Sprintf("%016x", time.Now().UnixNano()),
	}
	rm.role.Store(ReplicationRole(ReplicaRoleMaster))
	return rm
}

func (rm *ReplicationManager) Role() ReplicationRole {
	return rm.role.Load().(ReplicationRole)
}

func (rm *ReplicationManager) IsMaster() bool {
	return rm.Role() == ReplicaRoleMaster
}

func (rm *ReplicationManager) Replicas() []*ReplicaInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	result := make([]*ReplicaInfo, 0, len(rm.replicas))
	for _, r := range rm.replicas {
		result = append(result, r)
	}
	return result
}

func (rm *ReplicationManager) ReplicaCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.replicas)
}

func (rm *ReplicationManager) Offset() int64 {
	return atomic.LoadInt64(&rm.offset)
}

func (rm *ReplicationManager) ReplID() string {
	return rm.replID
}

func (rm *ReplicationManager) StartMaster(listenPort int) error {
	if rm.Role() != ReplicaRoleMaster {
		return fmt.Errorf("cannot start master: already a slave")
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return fmt.Errorf("failed to listen for replicas: %w", err)
	}

	rm.running = true
	logger.Info("replication master listening", "port", listenPort)

	go func() {
		for rm.running {
			conn, err := ln.Accept()
			if err != nil {
				if !rm.running {
					return
				}
				logger.Error("failed to accept replica connection", "error", err)
				continue
			}
			go rm.handleReplicaConnection(conn)
		}
	}()

	return nil
}

func (rm *ReplicationManager) handleReplicaConnection(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	reader := resp.NewReader(conn)
	writer := resp.NewWriter(conn)

	cmd, args, err := reader.ReadCommand()
	if err != nil {
		logger.Error("failed to read replica handshake", "remote", remote, "error", err)
		conn.Close()
		return
	}

	if cmd != "REPLCONF" || len(args) < 1 || args[0] != "handshake" {
		logger.Error("invalid replication handshake", "remote", remote, "cmd", cmd)
		conn.Close()
		return
	}

	writer.WriteSimpleString("OK")

	syncData, err := rm.buildFullSyncData()
	if err != nil {
		logger.Error("failed to build sync data", "error", err)
		conn.Close()
		return
	}

	data, err := json.Marshal(syncData)
	if err != nil {
		logger.Error("failed to marshal sync data", "error", err)
		conn.Close()
		return
	}

	writer.WriteBulkString(string(data))

	rm.mu.Lock()
	rm.replicas[remote] = &ReplicaInfo{
		Address:   remote,
		Connected: true,
		LastSync:  time.Now(),
		Offset:    syncData.Offset,
	}
	rm.mu.Unlock()

	logger.Info("replica connected and synced", "remote", remote)

	buf := make([]byte, 1)
	for rm.running {
		_, err := conn.Read(buf)
		if err != nil {
			logger.Info("replica disconnected", "remote", remote)
			rm.mu.Lock()
			delete(rm.replicas, remote)
			rm.mu.Unlock()
			return
		}
	}
}

func (rm *ReplicationManager) buildFullSyncData() (*FullSyncData, error) {
	rm.cache.mu.RLock()
	defer rm.cache.mu.RUnlock()

	items := make(map[string]*SyncItem, len(rm.cache.items))
	for key, item := range rm.cache.items {
		if item.IsExpired() {
			continue
		}
		items[key] = &SyncItem{
			Value:      item.Value,
			Expiration: item.Expiration,
		}
	}

	return &FullSyncData{
		Items:  items,
		Offset: atomic.LoadInt64(&rm.offset),
		ReplID: rm.replID,
	}, nil
}

func (rm *ReplicationManager) Propagate(command string, args []string) {
	if rm.Role() != ReplicaRoleMaster {
		return
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	offset := atomic.AddInt64(&rm.offset, 1)

	entry := ReplicationEntry{
		Offset:  offset,
		Command: command,
		Args:    args,
	}
	rm.replicationLog = append(rm.replicationLog, entry)

	data, err := json.Marshal(entry)
	if err != nil {
		logger.Error("failed to marshal replication entry", "error", err)
		return
	}

	for remote, replica := range rm.replicas {
		if !replica.Connected {
			continue
		}
		writer := resp.NewWriter(replica.Connection)
		if err := writer.WriteBulkString(string(data)); err != nil {
			logger.Warn("failed to propagate to replica", "remote", remote, "error", err)
			replica.Connected = false
		}
		replica.Offset = offset
		replica.LastSync = time.Now()
	}
}

func (rm *ReplicationManager) ConnectToMaster(masterAddr string, masterPort int) error {
	if rm.Role() != ReplicaRoleMaster {
		return fmt.Errorf("already a slave")
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", masterAddr, masterPort), 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to master: %w", err)
	}

	writer := resp.NewWriter(conn)
	reader := resp.NewReader(conn)

	writer.WriteCommand("REPLCONF", "handshake")

	val, err := reader.Read()
	if err != nil {
		conn.Close()
		return fmt.Errorf("handshake failed: %w", err)
	}

	if val.Type != resp.SimpleString || val.Str != "OK" {
		conn.Close()
		return fmt.Errorf("invalid master handshake response")
	}

	syncVal, err := reader.Read()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to receive sync data: %w", err)
	}

	var syncData FullSyncData
	syncStr := string(syncVal.Bulk)
	if err := json.Unmarshal([]byte(syncStr), &syncData); err != nil {
		conn.Close()
		return fmt.Errorf("failed to parse sync data: %w", err)
	}

	rm.applyFullSync(&syncData)

	rm.role.Store(ReplicationRole(ReplicaRoleSlave))
	rm.masterAddr = masterAddr
	rm.masterPort = masterPort
	rm.conn = conn
	rm.replID = syncData.ReplID
	atomic.StoreInt64(&rm.offset, syncData.Offset)

	rm.running = true
	go rm.receiveFromMaster()

	logger.Info("connected to master", "addr", masterAddr, "port", masterPort, "synced_items", len(syncData.Items))
	return nil
}

func (rm *ReplicationManager) applyFullSync(data *FullSyncData) {
	rm.cache.mu.Lock()
	defer rm.cache.mu.Unlock()

	rm.cache.items = make(map[string]*Item)
	rm.cache.currentBytes = 0

	for key, si := range data.Items {
		rm.cache.items[key] = &Item{
			Value:      si.Value,
			Expiration: si.Expiration,
			LastAccess: time.Now().UnixNano(),
		}
	}
}

func (rm *ReplicationManager) receiveFromMaster() {
	reader := resp.NewReader(rm.conn)
	for rm.running {
		val, err := reader.Read()
		if err != nil {
			logger.Error("error reading from master", "error", err)
			rm.role.Store(ReplicationRole(ReplicaRoleMaster))
			rm.running = false
			return
		}

		if val.Type != resp.BulkString {
			continue
		}

		var entry ReplicationEntry
		if err := json.Unmarshal([]byte(val.Bulk), &entry); err != nil {
			logger.Warn("failed to parse replication entry", "error", err)
			continue
		}

		rm.applyEntry(&entry)
		atomic.StoreInt64(&rm.offset, entry.Offset)
	}
}

func (rm *ReplicationManager) applyEntry(entry *ReplicationEntry) {
	switch stringsToUpper(entry.Command) {
	case "SET":
		if len(entry.Args) >= 2 {
			rm.cache.Set(entry.Args[0], entry.Args[1], 0)
		}
	case "DEL", "DELETE":
		if len(entry.Args) >= 1 {
			rm.cache.Delete(entry.Args[0])
		}
	}
}

func (rm *ReplicationManager) Stop() {
	rm.running = false
	close(rm.stopCh)

	if rm.conn != nil {
		rm.conn.Close()
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()
	for _, replica := range rm.replicas {
		if replica.Connection != nil {
			replica.Connection.Close()
		}
	}
	rm.replicas = make(map[string]*ReplicaInfo)

	logger.Info("replication manager stopped")
}

func (rm *ReplicationManager) Info() map[string]any {
	info := map[string]any{
		"role":    string(rm.Role()),
		"repl_id": rm.replID,
		"offset":  atomic.LoadInt64(&rm.offset),
	}

	if rm.Role() == ReplicaRoleSlave {
		info["master_addr"] = rm.masterAddr
		info["master_port"] = rm.masterPort
	} else {
		rm.mu.RLock()
		replicaCount := len(rm.replicas)
		rm.mu.RUnlock()
		info["connected_replicas"] = replicaCount
	}

	return info
}
