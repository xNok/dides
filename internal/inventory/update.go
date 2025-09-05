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
