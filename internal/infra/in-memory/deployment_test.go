package inmemory

import (
	"testing"

	"github.com/xnok/dides/internal/deployment"
)

func TestDeploymentStore_Save(t *testing.T) {
	store := NewDeploymentStore()

	req := deployment.DeploymentRequest{
		CodeVersion:          "v1.2.3",
		ConfigurationVersion: "config-v1.0",
		Labels:               map[string]string{"env": "prod", "app": "web"},
	}

	err := store.Save(&req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the deployment was saved
	if store.Count() != 1 {
		t.Errorf("Expected 1 deployment, got %d", store.Count())
	}

	// Get all deployments and verify the content
	deployments := store.GetAll()
	if len(deployments) != 1 {
		t.Fatalf("Expected 1 deployment, got %d", len(deployments))
	}

	saved := deployments[0]
	if saved.Request.CodeVersion != req.CodeVersion {
		t.Errorf("Expected CodeVersion %s, got %s", req.CodeVersion, saved.Request.CodeVersion)
	}

	if saved.Request.ConfigurationVersion != req.ConfigurationVersion {
		t.Errorf("Expected ConfigurationVersion %s, got %s", req.ConfigurationVersion, saved.Request.ConfigurationVersion)
	}

	if saved.Status != deployment.Running {
		t.Errorf("Expected status %v, got %v", deployment.Running, saved.Status)
	}

	if saved.ID == "" {
		t.Error("Expected deployment ID to be generated")
	}
}

func TestDeploymentStore_Get(t *testing.T) {
	store := NewDeploymentStore()

	req := deployment.DeploymentRequest{
		CodeVersion: "v1.0.0",
		Labels:      map[string]string{"env": "test"},
	}

	err := store.Save(&req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Get the deployment ID from the saved deployments
	deployments := store.GetAll()
	if len(deployments) != 1 {
		t.Fatalf("Expected 1 deployment, got %d", len(deployments))
	}

	deploymentID := deployments[0].ID

	// Test Get
	retrieved, exists := store.Get(deploymentID)
	if !exists {
		t.Fatal("Expected deployment to exist")
	}

	if retrieved.Request.CodeVersion != req.CodeVersion {
		t.Errorf("Expected CodeVersion %s, got %s", req.CodeVersion, retrieved.Request.CodeVersion)
	}

	// Test Get with non-existent ID
	_, exists = store.Get("non-existent")
	if exists {
		t.Error("Expected non-existent deployment to not exist")
	}
}

func TestDeploymentStore_GetByStatus(t *testing.T) {
	store := NewDeploymentStore()

	// Save multiple deployments
	req1 := deployment.DeploymentRequest{CodeVersion: "v1.0.0"}
	req2 := deployment.DeploymentRequest{CodeVersion: "v1.1.0"}
	req3 := deployment.DeploymentRequest{CodeVersion: "v1.2.0"}

	store.Save(&req1)
	store.Save(&req2)
	store.Save(&req3)

	// All should be Pending initially
	pending, err := store.GetByStatus(deployment.Running)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(pending) != 3 {
		t.Errorf("Expected 3 pending deployments, got %d", len(pending))
	}

	// Update one to Running
	deployments := store.GetAll()
	err = store.UpdateStatus(deployments[0].ID, deployment.Completed)
	if err != nil {
		t.Fatalf("Expected no error updating status, got %v", err)
	}

	// Check status distribution
	completed, err := store.GetByStatus(deployment.Completed)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	running, err := store.GetByStatus(deployment.Running)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(completed) != 1 {
		t.Errorf("Expected 1 completed deployments, got %d", len(completed))
	}

	if len(running) != 2 {
		t.Errorf("Expected 2 running deployment, got %d", len(running))
	}
}

func TestDeploymentStore_GetByLabels(t *testing.T) {
	store := NewDeploymentStore()

	req1 := deployment.DeploymentRequest{
		CodeVersion: "v1.0.0",
		Labels:      map[string]string{"env": "prod", "app": "web"},
	}

	req2 := deployment.DeploymentRequest{
		CodeVersion: "v1.1.0",
		Labels:      map[string]string{"env": "test", "app": "web"},
	}

	req3 := deployment.DeploymentRequest{
		CodeVersion: "v1.2.0",
		Labels:      map[string]string{"env": "prod", "app": "api"},
	}

	store.Save(&req1)
	store.Save(&req2)
	store.Save(&req3)

	// Find by single label
	webDeployments := store.GetByLabels(map[string]string{"app": "web"})
	if len(webDeployments) != 2 {
		t.Errorf("Expected 2 web deployments, got %d", len(webDeployments))
	}

	// Find by multiple labels
	prodWebDeployments := store.GetByLabels(map[string]string{"env": "prod", "app": "web"})
	if len(prodWebDeployments) != 1 {
		t.Errorf("Expected 1 prod web deployment, got %d", len(prodWebDeployments))
	}

	if prodWebDeployments[0].Request.CodeVersion != "v1.0.0" {
		t.Errorf("Expected v1.0.0, got %s", prodWebDeployments[0].Request.CodeVersion)
	}
}

func TestDeploymentStore_UpdateStatus(t *testing.T) {
	store := NewDeploymentStore()

	req := deployment.DeploymentRequest{CodeVersion: "v1.0.0"}
	store.Save(&req)

	deployments := store.GetAll()
	deploymentID := deployments[0].ID

	// Update to Running
	err := store.UpdateStatus(deploymentID, deployment.Running)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the update
	updated, exists := store.Get(deploymentID)
	if !exists {
		t.Fatal("Expected deployment to exist")
	}

	if updated.Status != deployment.Running {
		t.Errorf("Expected status %v, got %v", deployment.Running, updated.Status)
	}

	// Test updating non-existent deployment
	err = store.UpdateStatus("non-existent", deployment.Completed)
	if err != deployment.ErrDeploymentNotFound {
		t.Errorf("Expected ErrDeploymentNotFound, got %v", err)
	}
}

func TestDeploymentStore_Delete(t *testing.T) {
	store := NewDeploymentStore()

	req := deployment.DeploymentRequest{CodeVersion: "v1.0.0"}
	store.Save(&req)

	if store.Count() != 1 {
		t.Fatalf("Expected 1 deployment, got %d", store.Count())
	}

	deployments := store.GetAll()
	deploymentID := deployments[0].ID

	// Delete the deployment
	deleted := store.Delete(deploymentID)
	if !deleted {
		t.Fatal("Expected deletion to return true")
	}

	if store.Count() != 0 {
		t.Errorf("Expected 0 deployments, got %d", store.Count())
	}

	// Verify it's gone
	_, exists := store.Get(deploymentID)
	if exists {
		t.Fatal("Expected deployment to not exist after deletion")
	}

	// Test deleting non-existent deployment
	deleted = store.Delete("non-existent")
	if deleted {
		t.Error("Expected deletion of non-existent deployment to return false")
	}
}
