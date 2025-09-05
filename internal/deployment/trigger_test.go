package deployment_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/xnok/dides/internal/deployment"
	"github.com/xnok/dides/internal/deployment/mocks"
)

func TestTriggerService_TriggerDeployment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	service := deployment.NewTriggerService(mockStore)

	req := deployment.DeploymentRequest{
		CodeVersion:          "v1.2.3",
		ConfigurationVersion: "config-v1.0",
		Labels:               map[string]string{"env": "prod"},
	}

	// Set expectation: Save should be called once with the request and return nil
	mockStore.EXPECT().Save(&req).Return(nil).Times(1)

	err := service.TriggerDeployment(&req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestTriggerService_TriggerDeployment_EmptyCodeVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	service := deployment.NewTriggerService(mockStore)

	req := deployment.DeploymentRequest{
		CodeVersion: "", // Empty code version should trigger validation error
		Labels:      map[string]string{"env": "prod"},
	}

	// Save should NOT be called since validation should fail
	mockStore.EXPECT().Save(gomock.Any()).Times(0)

	err := service.TriggerDeployment(&req)
	if err != deployment.ErrInvalidDeploymentRequest {
		t.Errorf("Expected ErrInvalidDeploymentRequest, got %v", err)
	}
}

func TestTriggerService_TriggerDeployment_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	service := deployment.NewTriggerService(mockStore)

	req := deployment.DeploymentRequest{
		CodeVersion: "v1.0.0",
		Labels:      map[string]string{"env": "test"},
	}

	// Set expectation: Save should be called and return an error
	mockStore.EXPECT().Save(&req).Return(deployment.ErrInvalidDeploymentRequest).Times(1)

	err := service.TriggerDeployment(&req)
	if err != deployment.ErrInvalidDeploymentRequest {
		t.Errorf("Expected ErrInvalidDeploymentRequest from store, got %v", err)
	}
}
