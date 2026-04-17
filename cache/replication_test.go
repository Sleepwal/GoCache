package cache

import (
	"testing"
)

func TestReplication_NewManager(t *testing.T) {
	c := New()
	rm := NewReplicationManager(c)

	if rm.Role() != ReplicaRoleMaster {
		t.Errorf("expected master role, got %s", rm.Role())
	}
	if !rm.IsMaster() {
		t.Error("expected IsMaster=true")
	}
	if rm.ReplID() == "" {
		t.Error("expected non-empty repl ID")
	}
}

func TestReplication_ReplicaCount(t *testing.T) {
	c := New()
	rm := NewReplicationManager(c)

	if rm.ReplicaCount() != 0 {
		t.Errorf("expected 0 replicas, got %d", rm.ReplicaCount())
	}
}

func TestReplication_Offset(t *testing.T) {
	c := New()
	rm := NewReplicationManager(c)

	if rm.Offset() != 0 {
		t.Errorf("expected offset 0, got %d", rm.Offset())
	}
}

func TestReplication_PropagateAsMaster(t *testing.T) {
	c := New()
	rm := NewReplicationManager(c)

	rm.Propagate("SET", []string{"key", "value"})

	if rm.Offset() != 1 {
		t.Errorf("expected offset 1 after propagate, got %d", rm.Offset())
	}
}

func TestReplication_PropagateAsSlave(t *testing.T) {
	c := New()
	rm := NewReplicationManager(c)
	rm.role.Store(ReplicationRole(ReplicaRoleSlave))

	rm.Propagate("SET", []string{"key", "value"})

	if rm.Offset() != 0 {
		t.Errorf("expected offset 0 (slave should not propagate), got %d", rm.Offset())
	}
}

func TestReplication_Info(t *testing.T) {
	c := New()
	rm := NewReplicationManager(c)

	info := rm.Info()
	if info["role"] != "master" {
		t.Errorf("expected master role in info, got %v", info["role"])
	}
	if info["connected_replicas"] != 0 {
		t.Errorf("expected 0 connected replicas, got %v", info["connected_replicas"])
	}
}

func TestReplication_Stop(t *testing.T) {
	c := New()
	rm := NewReplicationManager(c)

	rm.Stop()

	if rm.running {
		t.Error("expected running=false after stop")
	}
}

func TestReplication_BuildFullSyncData(t *testing.T) {
	c := New()
	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	rm := NewReplicationManager(c)

	data, err := rm.buildFullSyncData()
	if err != nil {
		t.Fatalf("buildFullSyncData failed: %v", err)
	}
	if len(data.Items) != 2 {
		t.Errorf("expected 2 items in sync data, got %d", len(data.Items))
	}
	if data.ReplID != rm.ReplID() {
		t.Error("repl ID mismatch in sync data")
	}
}

func TestReplication_ApplyFullSync(t *testing.T) {
	c := New()
	rm := NewReplicationManager(c)

	data := &FullSyncData{
		Items: map[string]*SyncItem{
			"key1": {Value: "value1", Expiration: 0},
			"key2": {Value: "value2", Expiration: 0},
		},
		Offset: 100,
		ReplID: "test-repl-id",
	}

	rm.applyFullSync(data)

	if c.Count() != 2 {
		t.Errorf("expected 2 items after sync, got %d", c.Count())
	}
	val, found := c.Get("key1")
	if !found || val != "value1" {
		t.Errorf("expected key1=value1, found=%v val=%v", found, val)
	}
}
