package cache

import (
	"testing"
)

func TestCluster_NewClusterManager(t *testing.T) {
	c := New()
	cm := NewClusterManager(c)

	if cm.NodeID() == "" {
		t.Error("expected non-empty node ID")
	}
	if cm.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", cm.NodeCount())
	}
}

func TestCluster_AddNode(t *testing.T) {
	c := New()
	cm := NewClusterManager(c)

	node := &ShardNode{
		ID:      "node-1",
		Address: "127.0.0.1",
		Port:    6379,
	}
	cm.AddNode(node)

	if cm.NodeCount() != 1 {
		t.Errorf("expected 1 node, got %d", cm.NodeCount())
	}
}

func TestCluster_RemoveNode(t *testing.T) {
	c := New()
	cm := NewClusterManager(c)

	cm.AddNode(&ShardNode{ID: "node-1", Address: "127.0.0.1", Port: 6379})
	cm.RemoveNode("node-1")

	if cm.NodeCount() != 0 {
		t.Errorf("expected 0 nodes after removal, got %d", cm.NodeCount())
	}
}

func TestCluster_GetNodeForKey(t *testing.T) {
	c := New()
	cm := NewClusterManager(c)

	cm.AddNode(&ShardNode{ID: "node-1", Address: "127.0.0.1", Port: 6379})

	node := cm.GetNodeForKey("mykey")
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.ID != "node-1" {
		t.Errorf("expected node-1, got %s", node.ID)
	}
}

func TestCluster_KeySlot(t *testing.T) {
	slot1 := KeySlot("mykey")
	slot2 := KeySlot("mykey")

	if slot1 != slot2 {
		t.Error("same key should produce same slot")
	}

	if slot1 < 0 || slot1 >= TotalSlots {
		t.Errorf("slot %d out of range [0, %d)", slot1, TotalSlots)
	}
}

func TestCluster_MultipleNodes(t *testing.T) {
	c := New()
	cm := NewClusterManager(c)

	cm.AddNode(&ShardNode{ID: "node-1", Address: "127.0.0.1", Port: 6379})
	cm.AddNode(&ShardNode{ID: "node-2", Address: "127.0.0.2", Port: 6379})
	cm.AddNode(&ShardNode{ID: "node-3", Address: "127.0.0.3", Port: 6379})

	if cm.NodeCount() != 3 {
		t.Errorf("expected 3 nodes, got %d", cm.NodeCount())
	}

	differentNodes := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		key := string(rune(i))
		node := cm.GetNodeForKey(key)
		if node != nil {
			differentNodes[node.ID] = true
		}
	}

	if len(differentNodes) < 2 {
		t.Error("expected keys to be distributed across multiple nodes")
	}
}

func TestCluster_IsLocal(t *testing.T) {
	c := New()
	cm := NewClusterManager(c)

	if !cm.IsLocal("anykey") {
		t.Error("expected IsLocal=true when no nodes added")
	}

	cm.AddNode(&ShardNode{ID: "node-1", Address: "127.0.0.1", Port: 6379})
	cm.SetLocalNodeID("node-1")

	if !cm.IsLocal("anykey") {
		t.Error("expected IsLocal=true for local node")
	}
}

func TestCluster_Info(t *testing.T) {
	c := New()
	cm := NewClusterManager(c)

	info := cm.Info()
	if info["role"] != "master" && info["node_id"] == "" {
		t.Error("expected valid info")
	}
	if info["total_slots"] != TotalSlots {
		t.Errorf("expected %d total slots, got %v", TotalSlots, info["total_slots"])
	}
}

func TestCluster_GetSlotInfo(t *testing.T) {
	c := New()
	cm := NewClusterManager(c)

	cm.AddNode(&ShardNode{ID: "node-1", Address: "127.0.0.1", Port: 6379})

	slots := cm.GetSlotInfo()
	if len(slots) == 0 {
		t.Error("expected slot info")
	}

	totalSlots := 0
	for _, s := range slots {
		totalSlots += s.EndSlot - s.StartSlot + 1
	}
	if totalSlots != TotalSlots {
		t.Errorf("expected %d total slots in info, got %d", TotalSlots, totalSlots)
	}
}
