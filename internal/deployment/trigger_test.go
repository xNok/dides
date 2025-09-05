package deployment_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/xnok/dides/internal/deployment"
	"github.com/xnok/dides/internal/deployment/mocks"
)

func TestTriggerService_TriggerDeployment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockLocker := mocks.NewMockLocker(ctrl)
	mockStrategy := mocks.NewMockDeploymentStrategy(ctrl)
	service := deployment.NewTriggerService(mockStore, mockLocker, mockStrategy)

	ctx := context.Background()
	req := deployment.DeploymentRequest{
		CodeVersion:          "v1.2.3",
		ConfigurationVersion: "config-v1.0",
		Labels:               map[string]string{"env": "prod"},
		Configuration: deployment.Configuration{
			BatchSize:        2,
			FailureThreshold: 1,
		},
	}

	// Set expectations for locker
	mockLocker.EXPECT().Lock(ctx, "deployment").Return(nil).Times(1)
	mockLocker.EXPECT().Unlock(ctx, "deployment").Return(nil).Times(1)

	// Set expectations: GetByStatus should be called to check for running deployments
	mockStore.EXPECT().GetByStatus(deployment.Running).Return([]*deployment.DeploymentRecord{}, nil).Times(1)
	// Save should be called once with a deployment record and return nil
	mockStore.EXPECT().Save(gomock.Any()).DoAndReturn(func(record *deployment.DeploymentRecord) error {
		// Simulate ID assignment
		record.ID = "deployment-001"
		return nil
	}).Times(1)
	// Mock strategy StartDeployment call
	mockStrategy.EXPECT().StartDeployment(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := service.TriggerDeployment(ctx, &req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestTriggerService_TriggerDeployment_EmptyCodeVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockLocker := mocks.NewMockLocker(ctrl)
	mockStrategy := mocks.NewMockDeploymentStrategy(ctrl)
	service := deployment.NewTriggerService(mockStore, mockLocker, mockStrategy)

	ctx := context.Background()
	req := deployment.DeploymentRequest{
		CodeVersion: "", // Empty code version should trigger validation error
		Labels:      map[string]string{"env": "prod"},
	}

	// No expectations since validation should fail before any store or locker calls

	err := service.TriggerDeployment(ctx, &req)
	if err != deployment.ErrInvalidDeploymentRequest {
		t.Errorf("Expected ErrInvalidDeploymentRequest, got %v", err)
	}
}

func TestTriggerService_TriggerDeployment_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockLocker := mocks.NewMockLocker(ctrl)
	mockStrategy := mocks.NewMockDeploymentStrategy(ctrl)
	service := deployment.NewTriggerService(mockStore, mockLocker, mockStrategy)

	ctx := context.Background()
	req := deployment.DeploymentRequest{
		CodeVersion: "v1.0.0",
		Labels:      map[string]string{"env": "test"},
		Configuration: deployment.Configuration{
			BatchSize:        2,
			FailureThreshold: 1,
		},
	}

	// Set expectations for locker
	mockLocker.EXPECT().Lock(ctx, "deployment").Return(nil).Times(1)
	mockLocker.EXPECT().Unlock(ctx, "deployment").Return(nil).Times(1)

	// Set expectations: GetByStatus should be called to check for running deployments
	mockStore.EXPECT().GetByStatus(deployment.Running).Return([]*deployment.DeploymentRecord{}, nil).Times(1)
	// Save should be called and return an error
	mockStore.EXPECT().Save(gomock.Any()).Return(deployment.ErrInvalidDeploymentRequest).Times(1)

	err := service.TriggerDeployment(ctx, &req)
	if err != deployment.ErrInvalidDeploymentRequest {
		t.Errorf("Expected ErrInvalidDeploymentRequest from store, got %v", err)
	}
}

func TestTriggerService_TriggerDeployment_RolloutInProgress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockLocker := mocks.NewMockLocker(ctrl)
	mockStrategy := mocks.NewMockDeploymentStrategy(ctrl)
	service := deployment.NewTriggerService(mockStore, mockLocker, mockStrategy)

	ctx := context.Background()
	req := deployment.DeploymentRequest{
		CodeVersion: "v1.0.0",
		Labels:      map[string]string{"env": "test"},
		Configuration: deployment.Configuration{
			BatchSize:        2,
			FailureThreshold: 1,
		},
	}

	// Simulate a running deployment already exists
	runningDeployment := &deployment.DeploymentRecord{
		Status: deployment.Running,
	}

	// Set expectations for locker
	mockLocker.EXPECT().Lock(ctx, "deployment").Return(nil).Times(1)
	mockLocker.EXPECT().Unlock(ctx, "deployment").Return(nil).Times(1)

	// GetByStatus should return a running deployment
	mockStore.EXPECT().GetByStatus(deployment.Running).Return([]*deployment.DeploymentRecord{runningDeployment}, nil).Times(1)

	err := service.TriggerDeployment(ctx, &req)
	if err != deployment.ErrRolloutInProgress {
		t.Errorf("Expected ErrRolloutInProgress, got %v", err)
	}
}

func TestTriggerService_TriggerDeployment_LockError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	mockLocker := mocks.NewMockLocker(ctrl)
	mockStrategy := mocks.NewMockDeploymentStrategy(ctrl)
	service := deployment.NewTriggerService(mockStore, mockLocker, mockStrategy)

	ctx := context.Background()
	req := deployment.DeploymentRequest{
		CodeVersion: "v1.0.0",
		Labels:      map[string]string{"env": "test"},
		Configuration: deployment.Configuration{
			BatchSize:        2,
			FailureThreshold: 1,
		},
	}

	// Lock should fail
	lockErr := errors.New("failed to acquire lock")
	mockLocker.EXPECT().Lock(ctx, "deployment").Return(lockErr).Times(1)

	err := service.TriggerDeployment(ctx, &req)
	if err != lockErr {
		t.Errorf("Expected lock error, got %v", err)
	}
}
