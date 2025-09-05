package deployment_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/xnok/dides/internal/deployment"
	"github.com/xnok/dides/internal/deployment/mocks"
	"github.com/xnok/dides/internal/inventory"
)

func TestTriggerService_GetDeploymentStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockLocker := mocks.NewMockLocker(ctrl)
	mockStrategy := mocks.NewMockDeploymentStrategy(ctrl)
	service := deployment.NewTriggerService(mockStore, mockLocker, mockStrategy)

	// Test successful case
	t.Run("success", func(t *testing.T) {
		expectedDeployments := []*deployment.DeploymentRecord{
			{
				ID:     "deployment-1",
				Status: deployment.Running,
			},
			{
				ID:     "deployment-2",
				Status: deployment.Running,
			},
		}

		mockStore.EXPECT().GetByStatus(deployment.Running).Return(expectedDeployments, nil).Times(1)

		result, err := service.GetDeploymentStatus()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 deployments, got %d", len(result))
		}

		if result[0].ID != "deployment-1" {
			t.Errorf("Expected first deployment ID to be 'deployment-1', got %s", result[0].ID)
		}
	})

	// Test store error
	t.Run("store_error", func(t *testing.T) {
		storeErr := errors.New("store error")
		mockStore.EXPECT().GetByStatus(deployment.Running).Return(nil, storeErr).Times(1)

		result, err := service.GetDeploymentStatus()
		if err != storeErr {
			t.Errorf("Expected store error, got %v", err)
		}

		if result != nil {
			t.Errorf("Expected nil result on error, got %v", result)
		}
	})
}

func TestRollingDeployment_StartDeployment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockInventory := mocks.NewMockInventoryService(ctrl)
	rollingDeployment := deployment.NewRollingDeployment(mockStore, mockInventory)

	// Test case: No instances need update - deployment completed immediately
	t.Run("no_instances_need_update", func(t *testing.T) {
		record := &deployment.DeploymentRecord{
			ID: "deployment-1",
			Request: deployment.DeploymentRequest{
				CodeVersion:          "v1.0.0",
				ConfigurationVersion: "config-v1.0",
				Labels:               map[string]string{"env": "prod"},
				Configuration: deployment.Configuration{
					BatchSize: 2,
				},
			},
			Status: deployment.Running,
		}

		desiredState := inventory.State{
			CodeVersion:          "v1.0.0",
			ConfigurationVersion: "config-v1.0",
		}

		// Mock expectations
		mockInventory.EXPECT().CountNeedingUpdate(record.Request.Labels, desiredState).Return(5, nil).Times(1)
		mockInventory.EXPECT().GetNeedingUpdate(record.Request.Labels, desiredState, gomock.Any()).Return([]*inventory.Instance{}, nil).Times(1)
		mockStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(r *deployment.DeploymentRecord) error {
			// Verify the deployment is marked as completed
			if r.Status != deployment.Completed {
				t.Errorf("Expected status to be Completed, got %v", r.Status)
			}
			if r.Progress.CompletedInstances != 5 {
				t.Errorf("Expected 5 completed instances, got %d", r.Progress.CompletedInstances)
			}
			return nil
		}).Times(1)

		err := rollingDeployment.StartDeployment(record)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	// Test case: Normal deployment with instances
	t.Run("normal_deployment_with_instances", func(t *testing.T) {
		record := &deployment.DeploymentRecord{
			ID: "deployment-2",
			Request: deployment.DeploymentRequest{
				CodeVersion:          "v1.1.0",
				ConfigurationVersion: "config-v1.1",
				Labels:               map[string]string{"env": "staging"},
				Configuration: deployment.Configuration{
					BatchSize: 2,
				},
			},
			Status: deployment.Running,
		}

		desiredState := inventory.State{
			CodeVersion:          "v1.1.0",
			ConfigurationVersion: "config-v1.1",
		}

		instances := []*inventory.Instance{
			{Name: "instance-1"},
			{Name: "instance-2"},
		}

		// Mock expectations
		mockInventory.EXPECT().CountNeedingUpdate(record.Request.Labels, desiredState).Return(10, nil).Times(1)
		mockInventory.EXPECT().GetNeedingUpdate(record.Request.Labels, desiredState, gomock.Any()).Return(instances, nil).Times(1)
		mockInventory.EXPECT().UpdateDesiredState("instance-1", desiredState).Return(nil).Times(1)
		mockInventory.EXPECT().UpdateDesiredState("instance-2", desiredState).Return(nil).Times(1)
		mockStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(r *deployment.DeploymentRecord) error {
			// Verify the progress is updated correctly
			if r.Progress.TotalInstances != 10 {
				t.Errorf("Expected 10 total instances, got %d", r.Progress.TotalInstances)
			}
			if r.Progress.InProgressInstances != 2 {
				t.Errorf("Expected 2 in-progress instances, got %d", r.Progress.InProgressInstances)
			}
			if len(r.Progress.CurrentBatch) != 2 {
				t.Errorf("Expected 2 instances in current batch, got %d", len(r.Progress.CurrentBatch))
			}
			return nil
		}).Times(1)

		err := rollingDeployment.StartDeployment(record)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	// Test case: CountNeedingUpdate error
	t.Run("count_needing_update_error", func(t *testing.T) {
		record := &deployment.DeploymentRecord{
			Request: deployment.DeploymentRequest{
				CodeVersion:          "v1.0.0",
				ConfigurationVersion: "config-v1.0",
				Labels:               map[string]string{"env": "prod"},
			},
		}

		desiredState := inventory.State{
			CodeVersion:          "v1.0.0",
			ConfigurationVersion: "config-v1.0",
		}

		countErr := errors.New("count error")
		mockInventory.EXPECT().CountNeedingUpdate(record.Request.Labels, desiredState).Return(0, countErr).Times(1)

		err := rollingDeployment.StartDeployment(record)
		if err != countErr {
			t.Errorf("Expected count error, got %v", err)
		}
	})
}

func TestTriggerService_ProgressDeployment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockLocker := mocks.NewMockLocker(ctrl)
	mockStrategy := mocks.NewMockDeploymentStrategy(ctrl)
	service := deployment.NewTriggerService(mockStore, mockLocker, mockStrategy)

	ctx := context.Background()

	// Test case: Lock error
	t.Run("lock_error", func(t *testing.T) {
		lockErr := errors.New("lock error")
		mockLocker.EXPECT().Lock(ctx, "deployment").Return(lockErr).Times(1)

		result, err := service.ProgressDeployment(ctx)
		if err != lockErr {
			t.Errorf("Expected lock error, got %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result on lock error, got %v", result)
		}
	})

	// Test case: No running deployments
	t.Run("no_running_deployments", func(t *testing.T) {
		mockLocker.EXPECT().Lock(ctx, "deployment").Return(nil).Times(1)
		mockLocker.EXPECT().Unlock(ctx, "deployment").Return(nil).Times(1)
		mockStore.EXPECT().GetByStatus(deployment.Running).Return([]*deployment.DeploymentRecord{}, nil).Times(1)

		result, err := service.ProgressDeployment(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result when no deployments, got %v", result)
		}
	})

	// Test case: Multiple running deployments error
	t.Run("multiple_running_deployments", func(t *testing.T) {
		deployments := []*deployment.DeploymentRecord{
			{ID: "deployment-1", Status: deployment.Running},
			{ID: "deployment-2", Status: deployment.Running},
		}

		mockLocker.EXPECT().Lock(ctx, "deployment").Return(nil).Times(1)
		mockLocker.EXPECT().Unlock(ctx, "deployment").Return(nil).Times(1)
		mockStore.EXPECT().GetByStatus(deployment.Running).Return(deployments, nil).Times(1)

		result, err := service.ProgressDeployment(ctx)
		if err != deployment.ErrMoreThanOneInflightDeployment {
			t.Errorf("Expected ErrMoreThanOneInflightDeployment, got %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result on error, got %v", result)
		}
	})

	// Test case: Successful progress
	t.Run("successful_progress", func(t *testing.T) {
		deploymentRecord := &deployment.DeploymentRecord{
			ID: "deployment-1",
			Request: deployment.DeploymentRequest{
				CodeVersion:          "v1.0.0",
				ConfigurationVersion: "config-v1.0",
				Labels:               map[string]string{"env": "prod"},
			},
			Status: deployment.Running,
			Progress: deployment.DeploymentProgress{
				TotalInstances:      10,
				InProgressInstances: 2,
				CompletedInstances:  8,
				CurrentBatch:        []string{"instance-9", "instance-10"},
			},
		}

		updatedRecord := &deployment.DeploymentRecord{
			ID:     "deployment-1",
			Status: deployment.Completed,
		}

		mockLocker.EXPECT().Lock(ctx, "deployment").Return(nil).Times(1)
		mockLocker.EXPECT().Unlock(ctx, "deployment").Return(nil).Times(1)
		mockStore.EXPECT().GetByStatus(deployment.Running).Return([]*deployment.DeploymentRecord{deploymentRecord}, nil).Times(1)
		mockStrategy.EXPECT().ProgressDeployment(ctx, deploymentRecord).Return(updatedRecord, nil).Times(1)

		result, err := service.ProgressDeployment(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Errorf("Expected result, got nil")
			return
		}
		if result.Status != deployment.Completed {
			t.Errorf("Expected status to be Completed, got %v", result.Status)
		}
	})
}

// Note: maxFailuresExceeded and rollbackDeployment are private methods and are tested indirectly
// through ProgressDeployment when they are properly implemented.
// Currently maxFailuresExceeded always returns false and rollbackDeployment is a no-op.
