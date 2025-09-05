package deployment

import "context"

// DeploymentStrategy defines the interface for different deployment strategies
type DeploymentStrategy interface {
	// StartDeployment initializes a deployment with the given record
	StartDeployment(record *DeploymentRecord) error

	// ProgressDeployment advances the deployment to the next stage
	ProgressDeployment(ctx context.Context, record *DeploymentRecord) (*DeploymentRecord, error)

	// RollbackDeployment reverts a failed deployment
	RollbackDeployment(record *DeploymentRecord) error
}
