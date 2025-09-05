package inventory

import (
	"errors"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid registration token")
)

type RegistrationService struct {
	store Store
}

func NewRegistrationService(store Store) *RegistrationService {
	return &RegistrationService{
		store: store,
	}
}

func (s *RegistrationService) RegisterInstance(req RegistrationRequest) (*Instance, error) {

	// TODO: Validate registration token - we accept anthing for now to keep things simple
	if req.Token == "" {
		return nil, ErrInvalidToken
	}

	// Set the first connected timestamp
	req.Instance.LastPing = time.Now()

	// persist the instance
	s.store.Save(&req.Instance)

	return &req.Instance, nil
}
