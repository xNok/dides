package deployment

import "errors"

//go:generate mockgen -source=trigger.go -destination=mocks/mock_store.go -package=mocks

var (
	ErrInvalidDeploymentRequest = errors.New("invalid deployment request")
	ErrRolloutInProgress        = errors.New("deployment rollout in progress")
	ErrDeploymentNotFound       = errors.New("deployment not found")
)

type Store interface {
	Save(req *DeploymentRequest) error
}

type TriggerService struct {
	store Store
}

func NewTriggerService(store Store) *TriggerService {
	return &TriggerService{
		store: store,
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
func (s *TriggerService) TriggerDeployment(req *DeploymentRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	// Feature Request: If a deployment rollout is in progress, a new deployment rollout cannot start
	if s.isRolloutInProgress(req) {
		return ErrRolloutInProgress
	}

	// Save the deployment request to the store
	return s.store.Save(req)
}

func (s *TriggerService) isRolloutInProgress(req *DeploymentRequest) bool {
	// TODO: Implement logic to check if a rollout is in progress for the given request
	return false
}
