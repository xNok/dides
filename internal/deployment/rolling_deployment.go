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

//go:generate mockgen -source=rolling_deployment.go -destination=mocks/mock_inventory.go -package=mocks

type InventoryService interface {
	// GetInstancesByLabels returns instances that match the given labels
	GetInstancesByLabels(labels map[string]string) ([]*inventory.Instance, error)
	// UpdateDesiredState sets the desired state for an instance
	UpdateDesiredState(instanceKey string, state inventory.State) error

	// CountByLabels returns the total number of instances that match the given labels
	CountByLabels(labels map[string]string) int
	// CountNeedingUpdate returns the total number of instances that need to be updated (haven't been started yet)
	CountNeedingUpdate(labels map[string]string, desiredState inventory.State) (int, error)
	// GetNeedingUpdate returns instances that need to be updated (options can limit the number of results for batching)
	GetNeedingUpdate(labels map[string]string, desiredState inventory.State, opts *inventory.GetNeedingUpdateOptions) ([]*inventory.Instance, error)
	// CountInProgress returns the total number of instances currently being updated (desiredState == targetState but currentState != desiredState)
	CountInProgress(labels map[string]string, desiredState inventory.State) (int, error)
	// CountCompleted returns the total number of instances that have completed the update
	CountCompleted(labels map[string]string, desiredState inventory.State) (int, error)
	// CountFailed returns the total number of instances that have failed the update
	CountFailed(labels map[string]string, desiredState inventory.State) (int, error)
	// ResetFailedInstances resets the status of failed instances matching the labels to UNKNOWN
	ResetFailedInstances(labels map[string]string) error
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

	// 2. Get total count of instances matching labels for progress tracking
	totalInstances := rd.inventory.CountByLabels(record.Request.Labels)

	// 3. Initialize deployment progress
	record.Progress = DeploymentProgress{
		TotalMatchingInstances: totalInstances,
		InProgressInstances:    0,
		CompletedInstances:     0,
		FailedInstances:        0,
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

	// 7. Update desired state for instances in the current batch
	// TODO: Consider using UpdateMany such that it can be done in a transaction (so it can be rolled back)
	for _, instance := range currentBatch {
		if err := rd.inventory.UpdateDesiredState(instance.Name, desiredState); err != nil {
			return err
		}

		// update record if state was updated
		record.Progress.InProgressInstances++
	}

	// Update the deployment record with progress
	return rd.store.Update(record)
}

// ResetFailedInstances resets the status of failed instances matching the labels to UNKNOWN
func (rd *RollingDeployment) ResetFailedInstances(labels map[string]string) error {
	return rd.inventory.ResetFailedInstances(labels)
}

// ProgressDeployment checks instance states and progresses the deployment
func (rd *RollingDeployment) ProgressDeployment(ctx context.Context, record *DeploymentRecord) (*DeploymentRecord, error) {
	// 0. Determine desired state from the deployment request
	desiredState := inventory.State{
		CodeVersion:          record.Request.CodeVersion,
		ConfigurationVersion: record.Request.ConfigurationVersion,
	}

	// ------------------------------------------------------
	// State Refresh Logic
	// ------------------------------------------------------

	// 1. Check number of failed instances and update record
	failed, err := rd.inventory.CountFailed(record.Request.Labels, desiredState)
	if err != nil {
		return nil, err
	}

	// 2. How many of the current batch are done?
	completed, err := rd.inventory.CountCompleted(record.Request.Labels, desiredState)
	if err != nil {
		return nil, err
	}

	// 3. How many are still in progress (desiredState == targetState but currentState != desiredState)?
	inProgress, err := rd.inventory.CountInProgress(record.Request.Labels, desiredState)
	if err != nil {
		return nil, err
	}

	record.Progress.FailedInstances = failed
	record.Progress.CompletedInstances = completed
	record.Progress.InProgressInstances = inProgress

	// ------------------------------------------------------
	// State Update Logic
	// ------------------------------------------------------

	// 1. If failure threshold exceeded, return special error for automatic rollback handling
	if failed >= record.Request.Configuration.FailureThreshold {
		record.Status = Failed
		return record, ErrFailureThresholdExceeded
	}

	// 2. If there are still instances in progress, wait for them to complete
	if completed >= record.Progress.TotalMatchingInstances {
		record.Status = Completed
		record.Progress.CompletedInstances = completed
		return record, rd.store.Update(record)
	}

	// 3. If the current batch is still in progress, wait
	if inProgress == record.Request.Configuration.BatchSize {
		return record, rd.store.Update(record)
	}

	// 4. Get the next batch = batch_size - inflight
	opts := &inventory.GetNeedingUpdateOptions{
		Limit: record.Request.Configuration.BatchSize - record.Progress.InProgressInstances,
	}
	instances, err := rd.inventory.GetNeedingUpdate(record.Request.Labels, desiredState, opts)
	if err != nil {
		return record, err
	}

	// 5. Update the state for next batch
	// TODO: Consider using UpdateMany such that it can be done in a transaction (so it can be rolled back)
	for _, instance := range instances {
		if err := rd.inventory.UpdateDesiredState(instance.Name, desiredState); err != nil {
			return record, err
		}
		record.Progress.InProgressInstances++
	}

	return record, rd.store.Update(record)
}
