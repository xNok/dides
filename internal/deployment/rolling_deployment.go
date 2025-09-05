package deployment

import (
	"context"

	"github.com/xnok/dides/internal/inventory"
)

// RollingDeployment implements the rolling deployment strategy
type RollingDeployment struct {
	store     Store
	inventory InventoryService
}

type InventoryService interface {
	// GetInstancesByLabels returns instances that match the given labels
	GetInstancesByLabels(labels map[string]string) ([]*inventory.Instance, error)
	// UpdateDesiredState sets the desired state for an instance
	UpdateDesiredState(instanceKey string, state inventory.State) error

	// CountByLabels returns the total number of instances that match the given labels
	CountByLabels(labels map[string]string) int
	// CountNeedingUpdate returns the total number of instances that need to be updated
	CountNeedingUpdate(labels map[string]string, desiredState inventory.State) (int, error)
	// GetNeedingUpdate returns instances that need to be updated (options can limit the number of results for batching)
	GetNeedingUpdate(labels map[string]string, desiredState inventory.State, opts *inventory.GetNeedingUpdateOptions) ([]*inventory.Instance, error)
	// CountCompleted returns the total number of instances that have completed the update
	CountCompleted(labels map[string]string, desiredState inventory.State) (int, error)
}

// NewRollingDeployment creates a new rolling deployment strategy
func NewRollingDeployment(store Store, inventory InventoryService) *RollingDeployment {
	return &RollingDeployment{
		store:     store,
		inventory: inventory,
	}
}

// StartDeployment initializes the deployment with the number of instances matching the batch size
func (rd *RollingDeployment) StartDeployment(record *DeploymentRecord) error {
	// 1. Determine desired state from the deployment request
	desiredState := inventory.State{
		CodeVersion:          record.Request.CodeVersion,
		ConfigurationVersion: record.Request.ConfigurationVersion,
	}

	// 2. Get total count of instances needing update for progress tracking
	totalInstances, err := rd.inventory.CountNeedingUpdate(record.Request.Labels, desiredState)
	if err != nil {
		return err
	}

	// 3. Initialize deployment progress
	record.Progress = DeploymentProgress{
		TotalInstances:      totalInstances,
		InProgressInstances: 0,
		CompletedInstances:  0,
		FailedInstances:     0,
		CurrentBatch:        make([]string, 0),
	}

	// 4. Get instances that match the deployment labels AND need updates
	opts := &inventory.GetNeedingUpdateOptions{
		Limit: record.Request.Configuration.BatchSize,
	}
	instances, err := rd.inventory.GetNeedingUpdate(record.Request.Labels, desiredState, opts)
	if err != nil {
		return err
	}

	// 5. If no instances need updates, mark deployment as completed
	if len(instances) == 0 {
		record.Status = Completed
		record.Progress.CompletedInstances = totalInstances
		return rd.store.Update(record)
	}

	// 6. Start the first batch of deployments
	currentBatch := instances
	record.Progress.CurrentBatch = make([]string, len(currentBatch))

	// 7. Update desired state for instances in the current batch
	// TODO: Consider using UpdateMany such that it can be done in a transaction (so it can be rolled back)
	for i, instance := range currentBatch {
		if err := rd.inventory.UpdateDesiredState(instance.Name, desiredState); err != nil {
			return err
		}

		// update record if state was updated
		record.Progress.CurrentBatch[i] = instance.Name
		record.Progress.InProgressInstances++
	}

	// Update the deployment record with progress
	return rd.store.Update(record)
}

// ProgressDeployment checks instance states and progresses the deployment
func (rd *RollingDeployment) ProgressDeployment(ctx context.Context, record *DeploymentRecord) (*DeploymentRecord, error) {
	// 1. Determine desired state from the deployment request
	desiredState := inventory.State{
		CodeVersion:          record.Request.CodeVersion,
		ConfigurationVersion: record.Request.ConfigurationVersion,
	}

	// 2. Check if failure threshold exceeded
	if rd.IsFailureThresholdExceeded(record) {
		if err := rd.RollbackDeployment(record); err != nil {
			return nil, err
		}
		record.Status = Failed
		return record, rd.store.Update(record)
	}

	// 3. How many of the current batch are done?
	completed, err := rd.inventory.CountCompleted(record.Request.Labels, desiredState)
	if err != nil {
		return nil, err
	}

	if completed == record.Progress.TotalInstances {
		record.Status = Completed
		record.Progress.CompletedInstances = completed
		record.Progress.InProgressInstances = 0
		record.Progress.CurrentBatch = nil
		return record, rd.store.Update(record)
	}

	// TODO: 4. Get the next batch batch_size - inflight
	// TODO: 5. Update the state for next batch

	return record, nil
}

// IsFailureThresholdExceeded checks if the deployment has exceeded failure limits
func (rd *RollingDeployment) IsFailureThresholdExceeded(record *DeploymentRecord) bool {
	// TODO: Implement actual failure threshold logic
	// For now, return false (placeholder implementation)
	return false
}

// RollbackDeployment reverts a failed deployment
func (rd *RollingDeployment) RollbackDeployment(record *DeploymentRecord) error {
	// TODO: Implement actual rollback logic
	// 1. Cancel the current deployment
	// 2. Trigger a new deployment to the previous version

	return nil
}
