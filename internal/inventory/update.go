package inventory

import (
	"errors"
	"time"
)

var (
	ErrInstanceNotFound = errors.New("instance not found")
	ErrUpdateValidation = errors.New("invalid update request")
)

// UpdateRequest represents a request to update an instance
type UpdateRequest struct {
	Updates InstancePatch `json:"updates"`
}

// StateUpdateRequest represents a request to update instance state
type StateUpdateRequest struct {
	CurrentState *State `json:"current_state,omitempty"`
	DesiredState *State `json:"desired_state,omitempty"`
}

// Validate checks if the update request is valid
func (r UpdateRequest) Validate() error {
	// TODO: For now no validation I can think of
	return nil
}

type UpdateService struct {
	store Store
}

func NewUpdateService(store Store) *UpdateService {
	return &UpdateService{
		store: store,
	}
}

func (s *UpdateService) UpdateInstance(instanceKey string, req UpdateRequest) (*Instance, error) {
	now := time.Now()

	// Validate the instance key
	if instanceKey == "" {
		return nil, ErrUpdateValidation
	}

	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	req.Updates.LastPing = &now

	// Update the instance in the store
	instance, err := s.store.Update(instanceKey, req.Updates)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// UpdateInstanceState updates the current or desired state of an instance
func (s *UpdateService) UpdateInstanceState(instanceKey string, req StateUpdateRequest) (*Instance, error) {
	if instanceKey == "" {
		return nil, ErrUpdateValidation
	}

	// Get current instance
	instances := s.store.GetAll()
	var currentInstance *Instance
	for _, instance := range instances {
		key := instance.Name
		if key == "" {
			key = instance.IP
		}
		if key == instanceKey {
			currentInstance = instance
			break
		}
	}

	if currentInstance == nil {
		return nil, ErrInstanceNotFound
	}

	// Prepare patch
	patch := InstancePatch{}
	now := time.Now()
	patch.LastPing = &now

	// Update current state if provided
	if req.CurrentState != nil {
		patch.CurrentState = req.CurrentState
	}

	// Update the instance
	return s.store.Update(instanceKey, patch)
}

// GetDesiredState returns the desired state for an instance
func (s *UpdateService) GetDesiredState(instanceKey string) (*State, error) {
	instances := s.store.GetAll()
	for _, instance := range instances {
		key := instance.Name
		if key == "" {
			key = instance.IP
		}
		if key == instanceKey {
			return &instance.DesiredState, nil
		}
	}
	return nil, ErrInstanceNotFound
}
