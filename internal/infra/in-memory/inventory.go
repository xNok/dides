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
	if patch.Labels != nil {
		// For labels, we do a merge - existing labels are preserved unless overridden
		if updated.Labels == nil {
			updated.Labels = make(map[string]string)
		}
		for k, v := range patch.Labels {
			updated.Labels[k] = v
		}
	}

	if patch.LastPing != nil {
		updated.LastPing = *patch.LastPing
	}

	if patch.Status != nil {
		updated.Status = *patch.Status
	}

	if patch.CurrentState != nil {
		updated.CurrentState = *patch.CurrentState
	}

	if patch.DesiredState != nil {
		updated.DesiredState = *patch.DesiredState
	}

	// Update the stored instance
	s.instances[key] = &updated

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

// CountByLabels returns the count of instances matching the given labels
func (s *InventoryStore) CountByLabels(labels map[string]string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, instance := range s.instances {
		if s.matchesLabels(instance, labels) {
			count++
		}
	}

	return count
}

// GetNeedingUpdate returns instances that match labels and need state updates
func (s *InventoryStore) GetNeedingUpdate(labels map[string]string, desiredState inventory.State, opts *inventory.GetNeedingUpdateOptions) ([]*inventory.Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []*inventory.Instance
	maxResults := -1 // Default: no limit
	if opts != nil && opts.Limit > 0 {
		maxResults = opts.Limit
	}

	for _, instance := range s.instances {
		if s.matchesLabels(instance, labels) && s.needsUpdate(instance, desiredState) {
			instanceCopy := *instance
			matches = append(matches, &instanceCopy)

			// Check if we've reached the limit
			if maxResults > 0 && len(matches) >= maxResults {
				break
			}
		}
	}

	return matches, nil
}

// CountNeedingUpdate returns the count of instances that match labels and need state updates
func (s *InventoryStore) CountNeedingUpdate(labels map[string]string, desiredState inventory.State) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, instance := range s.instances {
		if s.matchesLabels(instance, labels) && s.needsUpdate(instance, desiredState) {
			count++
		}
	}

	return count, nil
}

// CountCompleted returns the count of instances that match labels and have completed the update to desired state
func (s *InventoryStore) CountCompleted(labels map[string]string, desiredState inventory.State) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, instance := range s.instances {
		if s.matchesLabels(instance, labels) && s.isCompleted(instance, desiredState) {
			count++
		}
	}

	return count, nil
}

// CountFailed returns the count of instances that match labels and have failed the update to desired state
func (s *InventoryStore) CountFailed(labels map[string]string, desiredState inventory.State) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, instance := range s.instances {
		if s.matchesLabels(instance, labels) && s.isFailed(instance, desiredState) {
			count++
		}
	}

	return count, nil
}

// ResetFailedInstances resets the status of failed instances matching the labels to UNKNOWN
func (s *InventoryStore) ResetFailedInstances(labels map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, instance := range s.instances {
		if s.matchesLabels(instance, labels) && instance.Status == inventory.FAILED {
			// Create a copy to modify
			updated := *instance
			updated.Status = inventory.UNKNOWN

			// Update the stored instance
			s.instances[key] = &updated
		}
	}

	return nil
}

// CountInProgress returns the count of instances that match labels and are currently being updated
// (desiredState == targetState but currentState != desiredState)
func (s *InventoryStore) CountInProgress(labels map[string]string, desiredState inventory.State) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, instance := range s.instances {
		if s.matchesLabels(instance, labels) && s.isInProgress(instance, desiredState) {
			count++
		}
	}

	return count, nil
}

// isCompleted checks if an instance has completed the update to the desired state
// This includes both successful completions and failed attempts
func (s *InventoryStore) isCompleted(instance *inventory.Instance, desiredState inventory.State) bool {
	// Successfully completed: current state matches desired state
	successfullyCompleted := instance.CurrentState.CodeVersion == desiredState.CodeVersion &&
		instance.CurrentState.ConfigurationVersion == desiredState.ConfigurationVersion &&
		instance.Status == inventory.HEALTHY

	return successfullyCompleted
}

// isFailed checks if an instance has failed the update to the desired state
// An instance is considered failed if it has the desired state but status is FAILED
func (s *InventoryStore) isFailed(instance *inventory.Instance, desiredState inventory.State) bool {
	return instance.DesiredState.CodeVersion == desiredState.CodeVersion &&
		instance.DesiredState.ConfigurationVersion == desiredState.ConfigurationVersion &&
		instance.Status == inventory.FAILED
}

// isInProgress checks if an instance is currently being updated
// An instance is in progress if: desiredState == targetState but currentState != desiredState
func (s *InventoryStore) isInProgress(instance *inventory.Instance, desiredState inventory.State) bool {
	// Check if the instance has been told to update to the target state
	hasDesiredState := instance.DesiredState.CodeVersion == desiredState.CodeVersion &&
		instance.DesiredState.ConfigurationVersion == desiredState.ConfigurationVersion

	// Check if the instance hasn't finished updating yet
	needsUpdate := instance.CurrentState.CodeVersion != desiredState.CodeVersion ||
		instance.CurrentState.ConfigurationVersion != desiredState.ConfigurationVersion

	// Instance is in progress if it has the desired state but still needs to update its current state
	return hasDesiredState && needsUpdate
}

// needsUpdate checks if an instance needs an update based on desired state
func (s *InventoryStore) needsUpdate(instance *inventory.Instance, desiredState inventory.State) bool {
	return instance.CurrentState.CodeVersion != desiredState.CodeVersion ||
		instance.CurrentState.ConfigurationVersion != desiredState.ConfigurationVersion
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
