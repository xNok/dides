package inventory

// StateService provides inventory state operations for searching and updating instance states
type StateService struct {
	store Store
}

// NewStateService creates a new state service for inventory state operations
func NewStateService(store Store) *StateService {
	return &StateService{
		store: store,
	}
}

// GetInstancesByLabels returns instances that match the given labels
func (s *StateService) GetInstancesByLabels(labels map[string]string) ([]*Instance, error) {
	if len(labels) == 0 {
		return s.store.GetAll(), nil
	}

	matches := s.store.GetByLabels(labels)
	return matches, nil
}

// UpdateDesiredState sets the desired state for an instance
func (s *StateService) UpdateDesiredState(instanceKey string, state State) error {
	patch := InstancePatch{
		DesiredState: &state,
	}

	_, err := s.store.Update(instanceKey, patch)
	return err
}

// GetInstancesWithState returns instances that need updates (current != desired)
// This is now delegated to the store for more efficient database-level filtering
func (s *StateService) GetInstancesWithState(currentState, desiredState State) ([]*Instance, error) {
	return s.store.GetInstancesWithState(currentState, desiredState)
}
