package deployment

import (
	"context"
	"errors"
)

const (
	lockKey = "deployment"
)

//go:generate mockgen -source=trigger.go -destination=mocks/mock_store.go -package=mocks

var (
	ErrInvalidDeploymentRequest = errors.New("invalid deployment request")
	ErrRolloutInProgress        = errors.New("deployment rollout in progress")
	ErrDeploymentNotFound       = errors.New("deployment not found")
)

type Store interface {
	Save(req *DeploymentRequest) error
	GetByStatus(status DeploymentStatus) ([]*DeploymentRecord, error)
}

type Locker interface {
	Lock(ctx context.Context, key string) error
	Unlock(ctx context.Context, key string) error
}

type TriggerService struct {
	store Store
	lock  Locker
}

func NewTriggerService(store Store, lock Locker) *TriggerService {
	return &TriggerService{
		store: store,
		lock:  lock,
	}
}

// Validate the deployment request
func (r DeploymentRequest) Validate() error {
	// TODO: validate the content of the request (eg. is the version valid)
	if r.CodeVersion == "" {
		return ErrInvalidDeploymentRequest
	}
	return nil
}

// TriggerDeployment initiates a new deployment
func (s *TriggerService) TriggerDeployment(ctx context.Context, req *DeploymentRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	// Concurency check we need a lock here in case two or more requests has arrived
	if err := s.lock.Lock(ctx, lockKey); err != nil {
		return err
	}
	defer s.lock.Unlock(ctx, lockKey)

	// 1. Feature Request: If a deployment rollout is in progress, a new deployment rollout cannot start
	if s.isRolloutInProgress() {
		return ErrRolloutInProgress
	}

	// 2. Save the deployment request
	if err := s.store.Save(req); err != nil {
		return err
	}

	// 3. trigger the deployment


	return nil
}

func (s *TriggerService) isRolloutInProgress() bool {
	runningDeployments, err := s.store.GetByStatus(Running)
	if err != nil {
		return false
	}

	return len(runningDeployments) > 0
}
