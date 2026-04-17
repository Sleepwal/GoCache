package cache

import (
	"fmt"
	"hash/crc32"
	"sort"
	"sync"
	"time"

	"GoCache/logger"
)

const (
	TotalSlots = 16384
)

type SlotRange struct {
	Start int
	End   int
}

type ShardNode struct {
	ID       string
	Address  string
	Port     int
	Slots    []SlotRange
	Password string
}

type ClusterSlot struct {
	StartSlot int
	EndSlot   int
	NodeID    string
}

type ClusterManager struct {
	mu       sync.RWMutex
	nodes    map[string]*ShardNode
	slots    [TotalSlots]*ShardNode
	local    *MemoryCache
	nodeID   string
}

func NewClusterManager(local *MemoryCache) *ClusterManager {
	return &ClusterManager{
		nodes:  make(map[string]*ShardNode),
		local:  local,
		nodeID: fmt.Sprintf("node-%016x", time.Now().UnixNano()),
	}
}

func (cm *ClusterManager) NodeID() string {
	return cm.nodeID
}

func (cm *ClusterManager) AddNode(node *ShardNode) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.nodes[node.ID] = node
	cm.reassignSlots()

	logger.Info("cluster node added", "node_id", node.ID, "address", node.Address)
}

func (cm *ClusterManager) RemoveNode(nodeID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.nodes, nodeID)
	cm.reassignSlots()

	logger.Info("cluster node removed", "node_id", nodeID)
}

func (cm *ClusterManager) reassignSlots() {
	nodeIDs := make([]string, 0, len(cm.nodes))
	for id := range cm.nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	for i := range cm.slots {
		cm.slots[i] = nil
	}

	if len(nodeIDs) == 0 {
		return
	}

	slotsPerNode := TotalSlots / len(nodeIDs)
	remainder := TotalSlots % len(nodeIDs)

	slotIndex := 0
	for i, nodeID := range nodeIDs {
		node := cm.nodes[nodeID]
		count := slotsPerNode
		if i < remainder {
			count++
		}

		node.Slots = nil
		start := slotIndex
		for j := 0; j < count; j++ {
			cm.slots[slotIndex] = node
			slotIndex++
		}
		node.Slots = append(node.Slots, SlotRange{Start: start, End: slotIndex - 1})
	}
}

func (cm *ClusterManager) GetNodeForKey(key string) *ShardNode {
	slot := KeySlot(key)
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.slots[slot]
}

func KeySlot(key string) int {
	hash := crc32.ChecksumIEEE([]byte(key))
	return int(hash % TotalSlots)
}

func (cm *ClusterManager) IsLocal(key string) bool {
	node := cm.GetNodeForKey(key)
	if node == nil {
		return true
	}
	return node.ID == cm.nodeID
}

func (cm *ClusterManager) GetLocalCache() *MemoryCache {
	return cm.local
}

func (cm *ClusterManager) GetNodes() []*ShardNode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]*ShardNode, 0, len(cm.nodes))
	for _, node := range cm.nodes {
		result = append(result, node)
	}
	return result
}

func (cm *ClusterManager) GetNodeByID(nodeID string) (*ShardNode, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	node, ok := cm.nodes[nodeID]
	return node, ok
}

func (cm *ClusterManager) NodeCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.nodes)
}

func (cm *ClusterManager) GetSlotInfo() []ClusterSlot {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []ClusterSlot
	var currentNodeID string
	var startSlot int

	for i := 0; i < TotalSlots; i++ {
		node := cm.slots[i]
		nodeID := ""
		if node != nil {
			nodeID = node.ID
		}

		if nodeID != currentNodeID {
			if currentNodeID != "" {
				result = append(result, ClusterSlot{
					StartSlot: startSlot,
					EndSlot:   i - 1,
					NodeID:    currentNodeID,
				})
			}
			startSlot = i
			currentNodeID = nodeID
		}
	}

	if currentNodeID != "" {
		result = append(result, ClusterSlot{
			StartSlot: startSlot,
			EndSlot:   TotalSlots - 1,
			NodeID:    currentNodeID,
		})
	}

	return result
}

func (cm *ClusterManager) SetLocalNodeID(id string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.nodeID = id
}

func (cm *ClusterManager) Info() map[string]any {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return map[string]any{
		"node_id":      cm.nodeID,
		"node_count":   len(cm.nodes),
		"total_slots":  TotalSlots,
		"cluster_state": "ok",
	}
}
