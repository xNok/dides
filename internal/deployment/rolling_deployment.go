package deployment

import (
	"context"
	"errors"

	"github.com/xnok/dides/internal/inventory"
)

var (
	ErrMoreThanOneInflightDeployment = errors.New("more than one inflight deployment found")
)

// startDeployment initialize the deployment with the number of instance matching the batch size
func (s *TriggerService) startDeployment(record *DeploymentRecord) error {
	// 1. Determine desired state from the deployment request
	desiredState := inventory.State{
		CodeVersion:          record.Request.CodeVersion,
		ConfigurationVersion: record.Request.ConfigurationVersion,
	}

	// 2. Get total count of instances needing update for progress tracking
	totalInstances, err := s.inventory.CountNeedingUpdate(record.Request.Labels, desiredState)
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
	instances, err := s.inventory.GetNeedingUpdate(record.Request.Labels, desiredState, opts)
	if err != nil {
		return err
	}

	// 5. If no instances need updates, mark deployment as completed
	if len(instances) == 0 {
		record.Status = Completed
		record.Progress.CompletedInstances = totalInstances
		return s.store.Update(record)
	}

	// 6. Start the first batch of deployments
	currentBatch := instances
	record.Progress.CurrentBatch = make([]string, len(currentBatch))

	// 7. Update desired state for instances in the current batch
	for i, instance := range currentBatch {
		if err := s.inventory.UpdateDesiredState(instance.Name, desiredState); err != nil {
			return err
		}

		// update record if state was updated
		record.Progress.CurrentBatch[i] = instance.Name
		record.Progress.InProgressInstances++
	}

	// Update the deployment record with progress
	return s.store.Update(record)
}

// ProgressDeployment checks instance states and progresses the deployment
func (s *TriggerService) ProgressDeployment(ctx context.Context) (*DeploymentRecord, error) {
	if err := s.lock.Lock(ctx, lockKey); err != nil {
		return nil, err
	}
	defer s.lock.Unlock(ctx, lockKey)

	// 1. Get the deployment record
	record, err := s.store.GetByStatus(Running)
	if err != nil {
		return nil, err
	}

	if len(record) == 0 {
		return nil, nil
	}

	if len(record) != 1 {
		return nil, ErrMoreThanOneInflightDeployment
	}

	// 2. Compute the next update

	// 2.0 How many have failed if failed > max_failures then store and rollback

	// 2.1 How many of the current batch are done

	// 2.2 Get the next batch batch_size - inflight

	// 2.3 Update the state for next batch

	return record[0], nil
}

// GetDeploymentStatus returns all currently running deployments
func (s *TriggerService) GetDeploymentStatus() ([]*DeploymentRecord, error) {
	return s.store.GetByStatus(Running)
}
