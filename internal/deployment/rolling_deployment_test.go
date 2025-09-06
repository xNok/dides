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

		result, err := service.GetDeploymentStatus(context.Background())
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

		result, err := service.GetDeploymentStatus(context.Background())
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
		mockInventory.EXPECT().CountByLabels(gomock.Any(), record.Request.Labels).Return(5, nil).Times(1)
		mockInventory.EXPECT().GetNeedingUpdate(gomock.Any(), record.Request.Labels, desiredState, gomock.Any()).Return([]*inventory.Instance{}, nil).Times(1)
		mockStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(r *deployment.DeploymentRecord) error {
			// Verify the deployment is marked as completed
			if r.Status != deployment.Completed {
				t.Errorf("Expected status to be Completed, got %v", r.Status)
			}
			if r.Progress.TotalMatchingInstances != 5 {
				t.Errorf("Expected 5 completed instances, got %d", r.Progress.TotalMatchingInstances)
			}
			return nil
		}).Times(1)

		err := rollingDeployment.StartDeployment(context.Background(), record)
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
		mockInventory.EXPECT().CountByLabels(gomock.Any(), record.Request.Labels).Return(10, nil).Times(1)
		mockInventory.EXPECT().GetNeedingUpdate(gomock.Any(), record.Request.Labels, desiredState, gomock.Any()).Return(instances, nil).Times(1)
		mockInventory.EXPECT().UpdateDesiredState(gomock.Any(), "instance-1", desiredState).Return(nil).Times(1)
		mockInventory.EXPECT().UpdateDesiredState(gomock.Any(), "instance-2", desiredState).Return(nil).Times(1)
		mockStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(r *deployment.DeploymentRecord) error {
			// Verify the progress is updated correctly
			if r.Progress.TotalMatchingInstances != 10 {
				t.Errorf("Expected 10 total instances, got %d", r.Progress.TotalMatchingInstances)
			}
			if r.Progress.InProgressInstances != 2 {
				t.Errorf("Expected 2 in-progress instances, got %d", r.Progress.InProgressInstances)
			}
			return nil
		}).Times(1)

		err := rollingDeployment.StartDeployment(context.Background(), record)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	// Test case: CountByLabels error
	t.Run("count_by_labels_error", func(t *testing.T) {
		record := &deployment.DeploymentRecord{
			Request: deployment.DeploymentRequest{
				CodeVersion:          "v1.0.0",
				ConfigurationVersion: "config-v1.0",
				Labels:               map[string]string{"env": "prod"},
			},
		}

		countErr := errors.New("count error")
		mockInventory.EXPECT().CountByLabels(gomock.Any(), record.Request.Labels).Return(0, countErr).Times(1)

		err := rollingDeployment.StartDeployment(context.Background(), record)
		if err != countErr {
			t.Errorf("Expected count error, got %v", err)
		}
	})
}

func TestRollingDeployment_CompleteDeploymentScenario(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockInventory := mocks.NewMockInventoryService(ctrl)
	rollingDeployment := deployment.NewRollingDeployment(mockStore, mockInventory)

	ctx := context.Background()

	// Test case: Complete deployment scenario with 5 instances, BatchSize 2, FailureThreshold 1
	t.Run("complete_deployment_scenario", func(t *testing.T) {
		// Initial deployment record
		record := &deployment.DeploymentRecord{
			ID: "deployment-complete",
			Request: deployment.DeploymentRequest{
				CodeVersion:          "v2.0.0",
				ConfigurationVersion: "config-v2.0",
				Labels:               map[string]string{"env": "prod", "service": "api"},
				Configuration: deployment.Configuration{
					BatchSize:        2,
					FailureThreshold: 1,
				},
			},
			Status: deployment.Running,
		}

		desiredState := inventory.State{
			CodeVersion:          "v2.0.0",
			ConfigurationVersion: "config-v2.0",
		}

		// Define all 5 instances
		allInstances := []*inventory.Instance{
			{Name: "instance-1", Labels: map[string]string{"env": "prod", "service": "api"}},
			{Name: "instance-2", Labels: map[string]string{"env": "prod", "service": "api"}},
			{Name: "instance-3", Labels: map[string]string{"env": "prod", "service": "api"}},
			{Name: "instance-4", Labels: map[string]string{"env": "prod", "service": "api"}},
			{Name: "instance-5", Labels: map[string]string{"env": "prod", "service": "api"}},
		}

		// Expected progress states at each step
		step1Progress := deployment.DeploymentProgress{
			TotalMatchingInstances: 5,
			InProgressInstances:    2, // instances 1, 2 started
			CompletedInstances:     0,
			FailedInstances:        0,
		}

		step2Progress := deployment.DeploymentProgress{
			TotalMatchingInstances: 5,
			InProgressInstances:    2, // instances 1, 2 still updating
			CompletedInstances:     0,
			FailedInstances:        0,
		}

		step3Progress := deployment.DeploymentProgress{
			TotalMatchingInstances: 5,
			InProgressInstances:    0, // No instances in progress (instances 1,2 completed, 3,4 not started yet)
			CompletedInstances:     2, // instances 1, 2 completed
			FailedInstances:        0,
		}

		step4Progress := deployment.DeploymentProgress{
			TotalMatchingInstances: 5,
			InProgressInstances:    2, // instances 3, 4 in progress (respects batch size)
			CompletedInstances:     2, // instances 1, 2 completed
			FailedInstances:        0,
		}

		step5Progress := deployment.DeploymentProgress{
			TotalMatchingInstances: 5,
			InProgressInstances:    1,
			CompletedInstances:     4, // instances 1, 2, 3, 4 completed
			FailedInstances:        0,
		}

		finalProgress := deployment.DeploymentProgress{
			TotalMatchingInstances: 5,
			InProgressInstances:    0, // all instances completed
			CompletedInstances:     5, // all instances completed
			FailedInstances:        0,
		}

		// Helper function to validate progress
		validateProgress := func(actual deployment.DeploymentProgress, expected deployment.DeploymentProgress, step string) {
			if actual.TotalMatchingInstances != expected.TotalMatchingInstances {
				t.Errorf("%s: Expected %d total instances, got %d", step, expected.TotalMatchingInstances, actual.TotalMatchingInstances)
			}
			if actual.InProgressInstances != expected.InProgressInstances {
				t.Errorf("%s: Expected %d in-progress instances, got %d", step, expected.InProgressInstances, actual.InProgressInstances)
			}
			if actual.CompletedInstances != expected.CompletedInstances {
				t.Errorf("%s: Expected %d completed instances, got %d", step, expected.CompletedInstances, actual.CompletedInstances)
			}
			if actual.FailedInstances != expected.FailedInstances {
				t.Errorf("%s: Expected %d failed instances, got %d", step, expected.FailedInstances, actual.FailedInstances)
			}
		}

		// Step 1: Start deployment with 5 instances, BatchSize 2, FailureThreshold 1
		t.Log("Step 1: Starting deployment")

		// Mock expectations for StartDeployment
		mockInventory.EXPECT().CountByLabels(gomock.Any(), record.Request.Labels).Return(5, nil).Times(1)
		mockInventory.EXPECT().GetNeedingUpdate(gomock.Any(), record.Request.Labels, desiredState, gomock.Any()).DoAndReturn(
			func(ctx context.Context, labels map[string]string, state inventory.State, opts *inventory.GetNeedingUpdateOptions) ([]*inventory.Instance, error) {
				// Return first 2 instances for the first batch
				return allInstances[:2], nil
			}).Times(1)

		// Expect UpdateDesiredState calls for first batch (instances 1 and 2)
		mockInventory.EXPECT().UpdateDesiredState(gomock.Any(), "instance-1", desiredState).Return(nil).Times(1)
		mockInventory.EXPECT().UpdateDesiredState(gomock.Any(), "instance-2", desiredState).Return(nil).Times(1)

		// Expect store update with initial progress
		mockStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(r *deployment.DeploymentRecord) error {
			validateProgress(r.Progress, step1Progress, "Step 1")
			return nil
		}).Times(1)

		err := rollingDeployment.StartDeployment(context.Background(), record)
		if err != nil {
			t.Fatalf("Expected no error in StartDeployment, got %v", err)
		}

		// Step 2: Simulate instances reporting being updated (first progress check)
		t.Log("Step 2: First progress check - instances 1 and 2 still updating")

		// Mock expectations for first ProgressDeployment call
		mockInventory.EXPECT().CountFailed(gomock.Any(), record.Request.Labels, desiredState).Return(0, nil).Times(1)
		mockInventory.EXPECT().CountCompleted(gomock.Any(), record.Request.Labels, desiredState).Return(0, nil).Times(1)  // Still updating
		mockInventory.EXPECT().CountInProgress(gomock.Any(), record.Request.Labels, desiredState).Return(2, nil).Times(1) // 2 instances in progress

		// No new instances to start (batch limit reached)
		mockInventory.EXPECT().GetNeedingUpdate(gomock.Any(), record.Request.Labels, desiredState, gomock.Any()).DoAndReturn(
			func(ctx context.Context, labels map[string]string, state inventory.State, opts *inventory.GetNeedingUpdateOptions) ([]*inventory.Instance, error) {
				// No new instances since we're at batch limit
				return []*inventory.Instance{}, nil
			}).Times(1)

		mockStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(r *deployment.DeploymentRecord) error {
			validateProgress(r.Progress, step2Progress, "Step 2")
			return nil
		}).Times(1)

		updatedRecord, err := rollingDeployment.ProgressDeployment(ctx, record)
		if err != nil {
			t.Fatalf("Expected no error in first ProgressDeployment, got %v", err)
		}
		if updatedRecord.Status != deployment.Running {
			t.Errorf("Expected status to still be Running, got %v", updatedRecord.Status)
		}

		// Step 3: Instances 1 and 2 complete, but algorithm might not start new instances yet
		t.Log("Step 3: Instances 1 and 2 complete, checking progress")

		mockInventory.EXPECT().CountFailed(gomock.Any(), record.Request.Labels, desiredState).Return(0, nil).Times(1)
		mockInventory.EXPECT().CountCompleted(gomock.Any(), record.Request.Labels, desiredState).Return(2, nil).Times(1)  // 2 completed
		mockInventory.EXPECT().CountInProgress(gomock.Any(), record.Request.Labels, desiredState).Return(0, nil).Times(1) // 0 in progress
		mockStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(r *deployment.DeploymentRecord) error {
			validateProgress(r.Progress, step3Progress, "Step 3")
			return nil
		}).Times(1)

		updatedRecord, err = rollingDeployment.ProgressDeployment(ctx, updatedRecord)
		if err != nil {
			t.Fatalf("Expected no error in second ProgressDeployment, got %v", err)
		}

		// Step 4: Progress deployment - instances 3 and 4 still updating
		t.Log("Step 4: Progress check - instances 3 and 4 still updating")

		mockInventory.EXPECT().CountFailed(gomock.Any(), record.Request.Labels, desiredState).Return(0, nil).Times(1)
		mockInventory.EXPECT().CountCompleted(gomock.Any(), record.Request.Labels, desiredState).Return(2, nil).Times(1)  // Still 2 completed
		mockInventory.EXPECT().CountInProgress(gomock.Any(), record.Request.Labels, desiredState).Return(2, nil).Times(1) // 2 instances in progress
		mockStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(r *deployment.DeploymentRecord) error {
			validateProgress(r.Progress, step4Progress, "Step 4")
			return nil
		}).Times(1)

		updatedRecord, err = rollingDeployment.ProgressDeployment(ctx, updatedRecord)
		if err != nil {
			t.Fatalf("Expected no error in third ProgressDeployment, got %v", err)
		}

		// Step 5: Instances 3 and 4 complete, start final instance (instance 5)
		t.Log("Step 5: Instances 3 and 4 complete, starting final instance 5")

		mockInventory.EXPECT().CountFailed(gomock.Any(), record.Request.Labels, desiredState).Return(0, nil).Times(1)
		mockInventory.EXPECT().CountCompleted(gomock.Any(), record.Request.Labels, desiredState).Return(4, nil).Times(1)  // 4 completed
		mockInventory.EXPECT().CountInProgress(gomock.Any(), record.Request.Labels, desiredState).Return(1, nil).Times(1) // 1 instance in progress
		mockInventory.EXPECT().GetNeedingUpdate(gomock.Any(), record.Request.Labels, desiredState, gomock.Any()).DoAndReturn(
			func(ctx context.Context, labels map[string]string, state inventory.State, opts *inventory.GetNeedingUpdateOptions) ([]*inventory.Instance, error) {
				// No new instances we are waiting for the last one to finish
				return []*inventory.Instance{}, nil
			}).Times(1)
		mockStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(r *deployment.DeploymentRecord) error {
			validateProgress(r.Progress, step5Progress, "Step 5")
			return nil
		}).Times(1)

		updatedRecord, err = rollingDeployment.ProgressDeployment(ctx, updatedRecord)
		if err != nil {
			t.Fatalf("Expected no error in fourth ProgressDeployment, got %v", err)
		}

		// Step 6: All instances complete - deployment finished
		t.Log("Step 6: All instances complete - deployment finished")

		mockInventory.EXPECT().CountFailed(gomock.Any(), record.Request.Labels, desiredState).Return(0, nil).Times(1)
		mockInventory.EXPECT().CountCompleted(gomock.Any(), record.Request.Labels, desiredState).Return(5, nil).Times(1)  // All 5 completed
		mockInventory.EXPECT().CountInProgress(gomock.Any(), record.Request.Labels, desiredState).Return(0, nil).Times(1) // None in progress

		mockStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(r *deployment.DeploymentRecord) error {
			if r.Status != deployment.Completed {
				t.Errorf("Expected status to be Completed, got %v", r.Status)
			}
			validateProgress(r.Progress, finalProgress, "Step 6")
			return nil
		}).Times(1)

		finalRecord, err := rollingDeployment.ProgressDeployment(ctx, updatedRecord)
		if err != nil {
			t.Fatalf("Expected no error in final ProgressDeployment, got %v", err)
		}

		// Verify final state
		if finalRecord.Status != deployment.Completed {
			t.Errorf("Expected final status to be Completed, got %v", finalRecord.Status)
		}
		validateProgress(finalRecord.Progress, finalProgress, "Final verification")

		t.Log("Deployment completed successfully!")
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
				TotalMatchingInstances: 10,
				InProgressInstances:    2,
				CompletedInstances:     8,
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
