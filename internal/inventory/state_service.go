package inventory

import "context"

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
func (s *StateService) GetInstancesByLabels(ctx context.Context, labels map[string]string) ([]*Instance, error) {
	if len(labels) == 0 {
		return s.store.GetAll(), nil
	}

	matches := s.store.GetByLabels(labels)
	return matches, nil
}

// UpdateDesiredState sets the desired state for an instance
func (s *StateService) UpdateDesiredState(ctx context.Context, instanceKey string, state State) error {
	patch := InstancePatch{
		DesiredState: &state,
	}

	_, err := s.store.Update(instanceKey, patch)
	return err
}

// CountByLabels returns the count of instances matching the given labels
func (s *StateService) CountByLabels(ctx context.Context, labels map[string]string) (int, error) {
	return s.store.CountByLabels(labels)
}

// GetNeedingUpdate returns instances that match labels and need state updates
func (s *StateService) GetNeedingUpdate(ctx context.Context, labels map[string]string, desiredState State, opts *GetNeedingUpdateOptions) ([]*Instance, error) {
	return s.store.GetNeedingUpdate(labels, desiredState, opts)
}

// CountNeedingUpdate returns the count of instances that match labels and need state updates
func (s *StateService) CountNeedingUpdate(ctx context.Context, labels map[string]string, desiredState State) (int, error) {
	return s.store.CountNeedingUpdate(labels, desiredState)
}

// CountCompleted returns the count of instances that match labels and have completed the update to desired state
func (s *StateService) CountCompleted(ctx context.Context, labels map[string]string, desiredState State) (int, error) {
	return s.store.CountCompleted(labels, desiredState)
}

// CountFailed returns the count of instances that match labels and have failed the update to desired state
func (s *StateService) CountFailed(ctx context.Context, labels map[string]string, desiredState State) (int, error) {
	return s.store.CountFailed(labels, desiredState)
}

// CountInProgress returns the count of instances that match labels and are currently being updated
func (s *StateService) CountInProgress(ctx context.Context, labels map[string]string, desiredState State) (int, error) {
	return s.store.CountInProgress(labels, desiredState)
}

// ResetFailedInstances resets the status of failed instances matching the labels to UNKNOWN
func (s *StateService) ResetFailedInstances(ctx context.Context, labels map[string]string) error {
	return s.store.ResetFailedInstances(labels)
}
