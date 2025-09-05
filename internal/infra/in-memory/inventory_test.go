package inmemory

import (
	"fmt"
	"testing"
	"time"

	"github.com/xnok/dides/internal/inventory"
)

func TestInventoryStore_Save(t *testing.T) {
	store := NewInventoryStore()

	instance := &inventory.Instance{
		IP:       "192.168.1.100",
		Name:     "test-instance",
		Labels:   map[string]string{"env": "test"},
		LastPing: time.Now(),
		Status:   inventory.HEALTHY,
	}

	err := store.Save(instance)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test retrieval
	retrieved, exists := store.Get("test-instance")
	if !exists {
		t.Fatal("Expected instance to exist")
	}

	if retrieved.IP != instance.IP {
		t.Errorf("Expected IP %s, got %s", instance.IP, retrieved.IP)
	}

	if retrieved.Name != instance.Name {
		t.Errorf("Expected Name %s, got %s", instance.Name, retrieved.Name)
	}
}

func TestInventoryStore_GetAll(t *testing.T) {
	store := NewInventoryStore()

	instance1 := &inventory.Instance{
		IP:   "192.168.1.100",
		Name: "instance-1",
	}

	instance2 := &inventory.Instance{
		IP:   "192.168.1.101",
		Name: "instance-2",
	}

	store.Save(instance1)
	store.Save(instance2)

	all := store.GetAll()
	if len(all) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(all))
	}
}

func TestInventoryStore_GetByLabels(t *testing.T) {
	store := NewInventoryStore()

	instance1 := &inventory.Instance{
		IP:     "192.168.1.100",
		Name:   "web-1",
		Labels: map[string]string{"role": "web", "env": "prod"},
	}

	instance2 := &inventory.Instance{
		IP:     "192.168.1.101",
		Name:   "db-1",
		Labels: map[string]string{"role": "db", "env": "prod"},
	}

	instance3 := &inventory.Instance{
		IP:     "192.168.1.102",
		Name:   "web-2",
		Labels: map[string]string{"role": "web", "env": "test"},
	}

	store.Save(instance1)
	store.Save(instance2)
	store.Save(instance3)

	// Test finding by single label
	webInstances := store.GetByLabels(map[string]string{"role": "web"})
	if len(webInstances) != 2 {
		t.Errorf("Expected 2 web instances, got %d", len(webInstances))
	}

	// Test finding by multiple labels
	prodWebInstances := store.GetByLabels(map[string]string{"role": "web", "env": "prod"})
	if len(prodWebInstances) != 1 {
		t.Errorf("Expected 1 prod web instance, got %d", len(prodWebInstances))
	}

	if prodWebInstances[0].Name != "web-1" {
		t.Errorf("Expected web-1, got %s", prodWebInstances[0].Name)
	}
}

func TestInventoryStore_Delete(t *testing.T) {
	store := NewInventoryStore()

	instance := &inventory.Instance{
		IP:   "192.168.1.100",
		Name: "test-instance",
	}

	store.Save(instance)

	// Verify it exists
	_, exists := store.Get("test-instance")
	if !exists {
		t.Fatal("Expected instance to exist before deletion")
	}

	// Delete it
	deleted := store.Delete("test-instance")
	if !deleted {
		t.Fatal("Expected deletion to return true")
	}

	// Verify it's gone
	_, exists = store.Get("test-instance")
	if exists {
		t.Fatal("Expected instance to not exist after deletion")
	}
}

func TestInventoryStore_Update(t *testing.T) {
	store := NewInventoryStore()

	// Create initial instance
	instance := &inventory.Instance{
		IP:       "192.168.1.100",
		Name:     "test-instance",
		Labels:   map[string]string{"env": "test", "version": "1.0"},
		LastPing: time.Now().Add(-time.Hour),
		Status:   inventory.UNKNOWN,
	}

	store.Save(instance)

	// Test partial update
	newStatus := inventory.HEALTHY
	newTime := time.Now()

	patch := inventory.InstancePatch{
		Status:   &newStatus,
		LastPing: &newTime,
		Labels:   map[string]string{"env": "prod", "region": "us-west"}, // Will merge with existing
	}

	updated, err := store.Update("test-instance", patch)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify updates - IP should remain unchanged since it's immutable
	if updated.IP != "192.168.1.100" {
		t.Errorf("Expected IP to remain unchanged: 192.168.1.100, got %s", updated.IP)
	}

	if updated.Status != newStatus {
		t.Errorf("Expected Status %v, got %v", newStatus, updated.Status)
	}

	if updated.Name != "test-instance" {
		t.Errorf("Expected Name to remain unchanged: %s", updated.Name)
	}

	// Verify label merging
	if updated.Labels["env"] != "prod" {
		t.Errorf("Expected env label to be updated to 'prod', got %s", updated.Labels["env"])
	}

	if updated.Labels["version"] != "1.0" {
		t.Errorf("Expected version label to remain '1.0', got %s", updated.Labels["version"])
	}

	if updated.Labels["region"] != "us-west" {
		t.Errorf("Expected region label to be 'us-west', got %s", updated.Labels["region"])
	}

	// Verify in store
	retrieved, exists := store.Get("test-instance")
	if !exists {
		t.Fatal("Expected updated instance to exist in store")
	}

	if retrieved.IP != "192.168.1.100" {
		t.Errorf("Expected stored IP to remain unchanged: 192.168.1.100, got %s", retrieved.IP)
	}
}

func TestInventoryStore_UpdateNonExistent(t *testing.T) {
	store := NewInventoryStore()

	newStatus := inventory.HEALTHY
	patch := inventory.InstancePatch{
		Status: &newStatus,
	}

	_, err := store.Update("non-existent", patch)
	if err != ErrInstanceNotFound {
		t.Errorf("Expected ErrInstanceNotFound, got %v", err)
	}
}

func TestInventoryStore_UpdateLabels(t *testing.T) {
	store := NewInventoryStore()

	// Create initial instance
	instance := &inventory.Instance{
		IP:     "192.168.1.100",
		Name:   "test-instance",
		Labels: map[string]string{"env": "test", "version": "1.0", "region": "us-east"},
	}

	store.Save(instance)

	// Update labels: change env, remove region, add new label
	labelUpdates := map[string]string{
		"env":    "prod",    // update existing
		"region": "",        // remove (empty string)
		"team":   "backend", // add new
	}

	updated, err := store.UpdateLabels("test-instance", labelUpdates)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify label changes
	if updated.Labels["env"] != "prod" {
		t.Errorf("Expected env to be 'prod', got %s", updated.Labels["env"])
	}

	if updated.Labels["version"] != "1.0" {
		t.Errorf("Expected version to remain '1.0', got %s", updated.Labels["version"])
	}

	if updated.Labels["team"] != "backend" {
		t.Errorf("Expected team to be 'backend', got %s", updated.Labels["team"])
	}

	if _, exists := updated.Labels["region"]; exists {
		t.Error("Expected region label to be removed")
	}

	// Verify in store
	retrieved, exists := store.Get("test-instance")
	if !exists {
		t.Fatal("Expected instance to exist in store")
	}

	if _, exists := retrieved.Labels["region"]; exists {
		t.Error("Expected region label to be removed from stored instance")
	}
}

func TestInventoryStore_CountByLabels(t *testing.T) {
	store := NewInventoryStore()

	instance1 := &inventory.Instance{
		IP:     "192.168.1.100",
		Name:   "web-1",
		Labels: map[string]string{"role": "web", "env": "prod"},
	}

	instance2 := &inventory.Instance{
		IP:     "192.168.1.101",
		Name:   "db-1",
		Labels: map[string]string{"role": "db", "env": "prod"},
	}

	instance3 := &inventory.Instance{
		IP:     "192.168.1.102",
		Name:   "web-2",
		Labels: map[string]string{"role": "web", "env": "test"},
	}

	store.Save(instance1)
	store.Save(instance2)
	store.Save(instance3)

	// Test counting by single label
	webCount := store.CountByLabels(map[string]string{"role": "web"})
	if webCount != 2 {
		t.Errorf("Expected 2 web instances, got %d", webCount)
	}

	// Test counting by multiple labels
	prodWebCount := store.CountByLabels(map[string]string{"role": "web", "env": "prod"})
	if prodWebCount != 1 {
		t.Errorf("Expected 1 prod web instance, got %d", prodWebCount)
	}

	// Test counting with no matches
	noMatchCount := store.CountByLabels(map[string]string{"role": "cache"})
	if noMatchCount != 0 {
		t.Errorf("Expected 0 cache instances, got %d", noMatchCount)
	}
}

func TestInventoryStore_GetNeedingUpdate(t *testing.T) {
	store := NewInventoryStore()

	// Create instances with different states
	instance1 := &inventory.Instance{
		IP:     "192.168.1.100",
		Name:   "web-1",
		Labels: map[string]string{"role": "web", "env": "prod"},
		CurrentState: inventory.State{
			CodeVersion:          "v1.0.0",
			ConfigurationVersion: "config-v1.0",
		},
	}

	instance2 := &inventory.Instance{
		IP:     "192.168.1.101",
		Name:   "web-2",
		Labels: map[string]string{"role": "web", "env": "prod"},
		CurrentState: inventory.State{
			CodeVersion:          "v2.0.0", // Already at desired version
			ConfigurationVersion: "config-v1.0",
		},
	}

	instance3 := &inventory.Instance{
		IP:     "192.168.1.102",
		Name:   "db-1",
		Labels: map[string]string{"role": "db", "env": "prod"},
		CurrentState: inventory.State{
			CodeVersion:          "v1.0.0",
			ConfigurationVersion: "config-v1.0",
		},
	}

	store.Save(instance1)
	store.Save(instance2)
	store.Save(instance3)

	desiredState := inventory.State{
		CodeVersion:          "v2.0.0",
		ConfigurationVersion: "config-v1.0",
	}

	// Get web instances needing update
	instances, err := store.GetNeedingUpdate(map[string]string{"role": "web"}, desiredState, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only return instance1 (web-1) since instance2 already has v2.0.0
	if len(instances) != 1 {
		t.Errorf("Expected 1 web instance needing update, got %d", len(instances))
	}

	if len(instances) > 0 && instances[0].Name != "web-1" {
		t.Errorf("Expected web-1, got %s", instances[0].Name)
	}
}

func TestInventoryStore_GetNeedingUpdateWithLimit(t *testing.T) {
	store := NewInventoryStore()

	// Create multiple instances that need updates
	for i := 1; i <= 5; i++ {
		instance := &inventory.Instance{
			IP:     fmt.Sprintf("192.168.1.%d", 100+i),
			Name:   fmt.Sprintf("web-%d", i),
			Labels: map[string]string{"role": "web", "env": "prod"},
			CurrentState: inventory.State{
				CodeVersion:          "v1.0.0", // All need update
				ConfigurationVersion: "config-v1.0",
			},
		}
		store.Save(instance)
	}

	desiredState := inventory.State{
		CodeVersion:          "v2.0.0",
		ConfigurationVersion: "config-v1.0",
	}

	// Test with limit of 2
	opts := &inventory.GetNeedingUpdateOptions{Limit: 2}
	instances, err := store.GetNeedingUpdate(map[string]string{"role": "web"}, desiredState, opts)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(instances) != 2 {
		t.Errorf("Expected 2 instances with limit, got %d", len(instances))
	}

	// Test without limit (should return all 5)
	instances, err = store.GetNeedingUpdate(map[string]string{"role": "web"}, desiredState, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(instances) != 5 {
		t.Errorf("Expected 5 instances without limit, got %d", len(instances))
	}

	// Test with limit of 0 (should return all)
	opts = &inventory.GetNeedingUpdateOptions{Limit: 0}
	instances, err = store.GetNeedingUpdate(map[string]string{"role": "web"}, desiredState, opts)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(instances) != 5 {
		t.Errorf("Expected 5 instances with limit 0, got %d", len(instances))
	}
}

func TestInventoryStore_CountNeedingUpdate(t *testing.T) {
	store := NewInventoryStore()

	// Create instances with different states
	instance1 := &inventory.Instance{
		IP:     "192.168.1.100",
		Name:   "web-1",
		Labels: map[string]string{"role": "web", "env": "prod"},
		CurrentState: inventory.State{
			CodeVersion:          "v1.0.0",
			ConfigurationVersion: "config-v1.0",
		},
	}

	instance2 := &inventory.Instance{
		IP:     "192.168.1.101",
		Name:   "web-2",
		Labels: map[string]string{"role": "web", "env": "prod"},
		CurrentState: inventory.State{
			CodeVersion:          "v2.0.0", // Already at desired version
			ConfigurationVersion: "config-v1.0",
		},
	}

	store.Save(instance1)
	store.Save(instance2)

	desiredState := inventory.State{
		CodeVersion:          "v2.0.0",
		ConfigurationVersion: "config-v1.0",
	}

	// Count instances needing update
	count, err := store.CountNeedingUpdate(map[string]string{"role": "web"}, desiredState)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Only instance1 needs update (v1.0.0 -> v2.0.0)
	if count != 1 {
		t.Errorf("Expected 1 web instance needing update, got %d", count)
	}
}
