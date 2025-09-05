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

	// 2. Get instances that match the deployment labels AND need updates (leverage database filtering)
	instances, err := s.inventory.GetNeedingUpdate(record.Request.Labels, desiredState)
	if err != nil {
		return err
	}

	// 3. Initialize deployment progress
	record.Progress = DeploymentProgress{
		TotalInstances:      len(instances),
		InProgressInstances: 0,
		CompletedInstances:  0,
		FailedInstances:     0,
		CurrentBatch:        make([]string, 0),
	}

	// 4. If no instances need updates, mark deployment as completed
	if len(instances) == 0 {
		record.Status = Completed
		record.Progress.CompletedInstances = len(instances)
		return s.store.Update(record)
	}

	// 5. Start the first batch of deployments
	batchSize := record.Request.Configuration.BatchSize
	if batchSize <= 0 {
		batchSize = 1 // Default to 1 if not specified
	}

	// 6. Start the first batch
	batchEnd := batchSize
	if batchEnd > len(instances) {
		batchEnd = len(instances)
	}

	currentBatch := instances[:batchEnd]
	record.Progress.CurrentBatch = make([]string, len(currentBatch))
	record.Progress.InProgressInstances = len(currentBatch)

	// 7. Update desired state for instances in the current batch
	for i, instance := range currentBatch {
		instanceKey := instance.Name
		if instanceKey == "" {
			instanceKey = instance.IP
		}
		record.Progress.CurrentBatch[i] = instanceKey

		if err := s.inventory.UpdateDesiredState(instanceKey, desiredState); err != nil {
			return err
		}
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

	return record[0], nil
}

// GetDeploymentStatus returns all currently running deployments
func (s *TriggerService) GetDeploymentStatus() ([]*DeploymentRecord, error) {
	return s.store.GetByStatus(Running)
}
