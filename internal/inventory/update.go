package inventory

import (
	"errors"
	"time"
)

var (
	ErrUpdateValidation = errors.New("invalid update request")
)

// UpdateRequest represents a request to update an instance
type UpdateRequest struct {
	InstanceKey string        `json:"instance_key"`
	Updates     InstancePatch `json:"updates"`
}

// Validate checks if the update request is valid
func (r UpdateRequest) Validate() error {
	if r.InstanceKey == "" {
		return ErrUpdateValidation
	}
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

func (s *UpdateService) UpdateInstance(req UpdateRequest) (*Instance, error) {
	now := time.Now()

	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	req.Updates.LastPing = &now

	// Update the instance in the store
	instance, err := s.store.Update(req.InstanceKey, req.Updates)
	if err != nil {
		return nil, err
	}

	return instance, nil
}
