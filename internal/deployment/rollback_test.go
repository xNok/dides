package deployment_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/xnok/dides/internal/deployment"
	"github.com/xnok/dides/internal/deployment/mocks"
	"github.com/xnok/dides/internal/inventory"
)

func TestTriggerService_TriggerRollback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockLocker := mocks.NewMockLocker(ctrl)
	mockStrategy := mocks.NewMockDeploymentStrategy(ctrl)
	service := deployment.NewTriggerService(mockStore, mockLocker, mockStrategy)

	ctx := context.Background()
	labels := map[string]string{"env": "prod", "service": "api"}
	config := deployment.Configuration{
		BatchSize:        2,
		FailureThreshold: 1,
	}

	// Mock previous completed deployment
	previousDeployment := &deployment.DeploymentRecord{
		ID: "deployment-previous",
		Request: deployment.DeploymentRequest{
			CodeVersion:          "v1.0.0",
			ConfigurationVersion: "config-v1.0",
			Labels:               labels,
			Configuration:        config,
		},
		Status: deployment.Completed,
	}

	// Set expectations for locker
	mockLocker.EXPECT().Lock(ctx, "deployment").Return(nil).Times(1)
	mockLocker.EXPECT().Unlock(ctx, "deployment").Return(nil).Times(1)

	// Check for running deployments (should be none)
	mockStore.EXPECT().GetByStatus(deployment.Running).Return([]*deployment.DeploymentRecord{}, nil).Times(1)

	// Reset failed instances before rollback
	mockStrategy.EXPECT().ResetFailedInstances(gomock.Any(), labels).Return(nil).Times(1)

	// Get previous completed deployments
	mockStore.EXPECT().GetByLabelsAndStatus(labels, deployment.Completed).Return([]*deployment.DeploymentRecord{previousDeployment}, nil).Times(1)

	// Save the rollback deployment record
	mockStore.EXPECT().Save(gomock.Any()).DoAndReturn(func(record *deployment.DeploymentRecord) error {
		// Verify the rollback request has the previous deployment's versions
		if record.Request.CodeVersion != "v1.0.0" {
			t.Errorf("Expected CodeVersion v1.0.0, got %s", record.Request.CodeVersion)
		}
		if record.Request.ConfigurationVersion != "config-v1.0" {
			t.Errorf("Expected ConfigurationVersion config-v1.0, got %s", record.Request.ConfigurationVersion)
		}
		record.ID = "rollback-deployment-001"
		return nil
	}).Times(1)

	// Start the rollback deployment
	mockStrategy.EXPECT().StartDeployment(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := service.TriggerRollback(ctx, labels, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestTriggerService_TriggerRollback_NoPreviousDeployment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockLocker := mocks.NewMockLocker(ctrl)
	mockStrategy := mocks.NewMockDeploymentStrategy(ctrl)
	service := deployment.NewTriggerService(mockStore, mockLocker, mockStrategy)

	ctx := context.Background()
	labels := map[string]string{"env": "prod", "service": "api"}
	config := deployment.Configuration{
		BatchSize:        2,
		FailureThreshold: 1,
	}

	// Set expectations for locker
	mockLocker.EXPECT().Lock(ctx, "deployment").Return(nil).Times(1)
	mockLocker.EXPECT().Unlock(ctx, "deployment").Return(nil).Times(1)

	// Check for running deployments (should be none)
	mockStore.EXPECT().GetByStatus(deployment.Running).Return([]*deployment.DeploymentRecord{}, nil).Times(1)

	// Reset failed instances before rollback
	mockStrategy.EXPECT().ResetFailedInstances(gomock.Any(), labels).Return(nil).Times(1)

	// Get previous completed deployments (none found)
	mockStore.EXPECT().GetByLabelsAndStatus(labels, deployment.Completed).Return([]*deployment.DeploymentRecord{}, nil).Times(1)

	err := service.TriggerRollback(ctx, labels, config)
	if err != deployment.ErrNoPreviousDeploymentFound {
		t.Errorf("Expected ErrNoPreviousDeploymentFound, got %v", err)
	}
}

func TestTriggerService_TriggerRollback_CancelsInProgressDeployment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockLocker := mocks.NewMockLocker(ctrl)
	mockStrategy := mocks.NewMockDeploymentStrategy(ctrl)
	service := deployment.NewTriggerService(mockStore, mockLocker, mockStrategy)

	ctx := context.Background()
	labels := map[string]string{"env": "prod", "service": "api"}
	config := deployment.Configuration{
		BatchSize:        2,
		FailureThreshold: 1,
	}

	// Mock running deployment that should be cancelled
	runningDeployment := &deployment.DeploymentRecord{
		ID: "deployment-running",
		Request: deployment.DeploymentRequest{
			CodeVersion:          "v2.0.0",
			ConfigurationVersion: "config-v2.0",
			Labels:               labels,
		},
		Status: deployment.Running,
	}

	// Mock previous completed deployment to rollback to
	previousDeployment := &deployment.DeploymentRecord{
		ID: "deployment-previous",
		Request: deployment.DeploymentRequest{
			CodeVersion:          "v1.0.0",
			ConfigurationVersion: "config-v1.0",
			Labels:               labels,
			Configuration:        config,
		},
		Status: deployment.Completed,
	}

	// Set expectations for locker
	mockLocker.EXPECT().Lock(ctx, "deployment").Return(nil).Times(1)
	mockLocker.EXPECT().Unlock(ctx, "deployment").Return(nil).Times(1)

	// Check for running deployments (should find one) - called twice by isRolloutInProgress and for cancellation
	mockStore.EXPECT().GetByStatus(deployment.Running).Return([]*deployment.DeploymentRecord{runningDeployment}, nil).Times(2)

	// Cancel the running deployment
	mockStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(record *deployment.DeploymentRecord) error {
		// Verify the running deployment is marked as failed
		if record.Status != deployment.Failed {
			t.Errorf("Expected status Failed, got %v", record.Status)
		}
		return nil
	}).Times(1)

	// Reset failed instances before rollback
	mockStrategy.EXPECT().ResetFailedInstances(gomock.Any(), labels).Return(nil).Times(1)

	// Get previous completed deployments
	mockStore.EXPECT().GetByLabelsAndStatus(labels, deployment.Completed).Return([]*deployment.DeploymentRecord{previousDeployment}, nil).Times(1)

	// Save the rollback deployment record
	mockStore.EXPECT().Save(gomock.Any()).DoAndReturn(func(record *deployment.DeploymentRecord) error {
		// Verify the rollback request has the previous deployment's versions
		if record.Request.CodeVersion != "v1.0.0" {
			t.Errorf("Expected CodeVersion v1.0.0, got %s", record.Request.CodeVersion)
		}
		if record.Request.ConfigurationVersion != "config-v1.0" {
			t.Errorf("Expected ConfigurationVersion config-v1.0, got %s", record.Request.ConfigurationVersion)
		}
		record.ID = "rollback-deployment-001"
		return nil
	}).Times(1)

	// Start the rollback deployment
	mockStrategy.EXPECT().StartDeployment(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := service.TriggerRollback(ctx, labels, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestRollingDeployment_FailureThresholdExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockInventory := mocks.NewMockInventoryService(ctrl)
	rollingDeployment := deployment.NewRollingDeployment(mockStore, mockInventory)

	ctx := context.Background()

	record := &deployment.DeploymentRecord{
		ID: "deployment-fail",
		Request: deployment.DeploymentRequest{
			CodeVersion:          "v2.0.0",
			ConfigurationVersion: "config-v2.0",
			Labels:               map[string]string{"env": "prod", "service": "api"},
			Configuration: deployment.Configuration{
				BatchSize:        2,
				FailureThreshold: 1, // Failure threshold is 1
			},
		},
		Status: deployment.Running,
	}

	desiredState := inventory.State{
		CodeVersion:          "v2.0.0",
		ConfigurationVersion: "config-v2.0",
	}

	// Mock expectations - simulate failure threshold exceeded
	mockInventory.EXPECT().CountFailed(gomock.Any(), record.Request.Labels, desiredState).Return(2, nil).Times(1) // 2 failures > threshold (1)
	mockInventory.EXPECT().CountCompleted(gomock.Any(), record.Request.Labels, desiredState).Return(0, nil).Times(1)
	mockInventory.EXPECT().CountInProgress(gomock.Any(), record.Request.Labels, desiredState).Return(0, nil).Times(1)

	updatedRecord, err := rollingDeployment.ProgressDeployment(ctx, record)

	// Should return ErrFailureThresholdExceeded
	if err != deployment.ErrFailureThresholdExceeded {
		t.Errorf("Expected ErrFailureThresholdExceeded, got %v", err)
	}

	// Record status should be set to Failed
	if updatedRecord.Status != deployment.Failed {
		t.Errorf("Expected status Failed, got %v", updatedRecord.Status)
	}
}
