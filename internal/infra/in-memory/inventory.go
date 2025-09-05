package inmemory

import (
	"errors"
	"sync"

	"github.com/xnok/dides/internal/inventory"
)

var (
	ErrInstanceNotFound = errors.New("instance not found")
)

// InventoryStore is an in-memory implementation of the inventory.Store interface
type InventoryStore struct {
	mu        sync.RWMutex
	instances map[string]*inventory.Instance // key is instance name or IP
}

// NewInventoryStore creates a new in-memory inventory store
func NewInventoryStore() *InventoryStore {
	return &InventoryStore{
		instances: make(map[string]*inventory.Instance),
	}
}

// Save stores an instance in memory, using the instance name as the key
// If an instance with the same name already exists, it will be updated
func (s *InventoryStore) Save(instance *inventory.Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use instance name as the key, fallback to IP if name is empty
	key := instance.Name
	if key == "" {
		key = instance.IP
	}

	// Create a copy to avoid external modifications
	instanceCopy := *instance
	s.instances[key] = &instanceCopy

	return nil
}

// Update applies a partial update (patch) to an existing instance
func (s *InventoryStore) Update(key string, patch inventory.InstancePatch) (*inventory.Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, exists := s.instances[key]
	if !exists {
		return nil, ErrInstanceNotFound
	}

	// Create a copy to modify
	updated := *instance

	// Apply patch fields if they are provided (not nil)
	if patch.IP != nil {
		updated.IP = *patch.IP
	}
	if patch.Name != nil {
		// If name is being changed, we need to update the key
		oldKey := key
		newKey := *patch.Name
		if newKey == "" {
			newKey = updated.IP // fallback to IP if name becomes empty
		}

		updated.Name = *patch.Name

		// If the key changed, remove old entry and add with new key
		if oldKey != newKey {
			delete(s.instances, oldKey)
			s.instances[newKey] = &updated
		} else {
			s.instances[key] = &updated
		}
	} else {
		s.instances[key] = &updated
	}

	if patch.Labels != nil {
		// For labels, we do a merge - existing labels are preserved unless overridden
		if updated.Labels == nil {
			updated.Labels = make(map[string]string)
		}
		for k, v := range patch.Labels {
			updated.Labels[k] = v
		}
		// Update the stored instance with merged labels
		if patch.Name != nil && *patch.Name != key {
			// Key was changed above, instance is already updated
		} else {
			s.instances[key] = &updated
		}
	}

	if patch.LastPing != nil {
		updated.LastPing = *patch.LastPing
		if patch.Name != nil && *patch.Name != key {
			// Key was changed above, instance is already updated
		} else {
			s.instances[key] = &updated
		}
	}

	if patch.Status != nil {
		updated.Status = *patch.Status
		if patch.Name != nil && *patch.Name != key {
			// Key was changed above, instance is already updated
		} else {
			s.instances[key] = &updated
		}
	}

	// Return a copy of the updated instance
	result := updated
	return &result, nil
}

// Get retrieves an instance by name or IP
func (s *InventoryStore) Get(key string) (*inventory.Instance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instance, exists := s.instances[key]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent external modifications
	instanceCopy := *instance
	return &instanceCopy, true
}

// GetAll returns all stored instances
func (s *InventoryStore) GetAll() []*inventory.Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instances := make([]*inventory.Instance, 0, len(s.instances))
	for _, instance := range s.instances {
		// Return copies to prevent external modifications
		instanceCopy := *instance
		instances = append(instances, &instanceCopy)
	}

	return instances
}

// Delete removes an instance by name or IP
func (s *InventoryStore) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.instances[key]
	if exists {
		delete(s.instances, key)
	}

	return exists
}

// Count returns the number of stored instances
func (s *InventoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.instances)
}

// GetByIP finds an instance by its IP address
func (s *InventoryStore) GetByIP(ip string) (*inventory.Instance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, instance := range s.instances {
		if instance.IP == ip {
			instanceCopy := *instance
			return &instanceCopy, true
		}
	}

	return nil, false
}

// GetByLabels finds instances that match all provided labels
func (s *InventoryStore) GetByLabels(labels map[string]string) []*inventory.Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []*inventory.Instance

	for _, instance := range s.instances {
		if s.matchesLabels(instance, labels) {
			instanceCopy := *instance
			matches = append(matches, &instanceCopy)
		}
	}

	return matches
}

// UpdateLabels provides more granular control over label updates
// Use empty string value to remove a label
func (s *InventoryStore) UpdateLabels(key string, labelUpdates map[string]string) (*inventory.Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, exists := s.instances[key]
	if !exists {
		return nil, ErrInstanceNotFound
	}

	// Create a copy to modify
	updated := *instance
	if updated.Labels == nil {
		updated.Labels = make(map[string]string)
	} else {
		// Copy existing labels
		newLabels := make(map[string]string)
		for k, v := range updated.Labels {
			newLabels[k] = v
		}
		updated.Labels = newLabels
	}

	// Apply label updates
	for k, v := range labelUpdates {
		if v == "" {
			// Empty string means remove the label
			delete(updated.Labels, k)
		} else {
			updated.Labels[k] = v
		}
	}

	s.instances[key] = &updated

	// Return a copy of the updated instance
	result := updated
	return &result, nil
}

// matchesLabels checks if an instance has all the specified labels
func (s *InventoryStore) matchesLabels(instance *inventory.Instance, labels map[string]string) bool {
	if instance.Labels == nil {
		return len(labels) == 0
	}

	for key, value := range labels {
		if instanceValue, exists := instance.Labels[key]; !exists || instanceValue != value {
			return false
		}
	}

	return true
}
