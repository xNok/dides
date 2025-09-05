package deployment

import "context"

// DeploymentStrategy defines the interface for different deployment strategies
type DeploymentStrategy interface {
	// StartDeployment initializes a deployment with the given record
	StartDeployment(ctx context.Context, record *DeploymentRecord) error

	// ProgressDeployment advances the deployment to the next stage
	ProgressDeployment(ctx context.Context, record *DeploymentRecord) (*DeploymentRecord, error)

	// ResetFailedInstances resets the status of failed instances matching the labels to UNKNOWN
	ResetFailedInstances(ctx context.Context, labels map[string]string) error
}
