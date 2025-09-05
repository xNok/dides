package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xnok/dides/internal/deployment"
	inmemory "github.com/xnok/dides/internal/infra/in-memory"
	"github.com/xnok/dides/internal/inventory"
	"github.com/xnok/dides/pkg/simulator"
)

func setupTestServer() *httptest.Server {
	// Initialize the in-memory stores and services (same as main)
	inventoryStore := inmemory.NewInventoryStore()
	registrationService = inventory.NewRegistrationService(inventoryStore)
	updateService = inventory.NewUpdateService(inventoryStore)

	deploymentStore := inmemory.NewDeploymentStore()
	deploymentLock := inmemory.NewInMemoryLocker()
	inventoryStateService := inventory.NewStateService(inventoryStore)

	// Create rolling deployment strategy and inject it into the trigger service
	rollingStrategy := deployment.NewRollingDeployment(deploymentStore, inventoryStateService)
	triggerService = deployment.NewTriggerService(deploymentStore, deploymentLock, rollingStrategy)

	// Setup the router (same as main)
	r := setupRouter()

	return httptest.NewServer(r)
}

func TestController_RegisterInstancesFromConfig(t *testing.T) {
	// Start test server
	server := setupTestServer()
	defer server.Close()

	// Load simulator config using the utility
	configFile := "../../testdata/simulator.config.yaml"
	config, err := simulator.LoadConfigFromFile(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(config.Instances) == 0 {
		t.Fatal("No instances found in config file")
	}

	// Create test utilities
	testUtils := simulator.NewTestUtilities(server, config)
	testData := simulator.NewTestDataGenerator()

	// 1. Register all instances using the utility
	registeredNames := testUtils.RegisterAllInstances(t, "test-token")
	t.Logf("Successfully registered %d instances from config: %v", len(registeredNames), registeredNames)

	// 1.1 validate the registration
	instances := testUtils.GetAllInstances(t)
	if len(instances) != len(registeredNames) {
		t.Fatalf("Expected %d instances, got %d", len(registeredNames), len(instances))
	}

	// 2. Simulate the instances sending updates
	for i := range registeredNames {
		testUtils.UpdateInstance(t, registeredNames[i], testData.CreateUnknownUpdate())
	}

	// 3. Validate we can retrieve all instances after updates
	updatedInstances := testUtils.GetAllInstances(t)
	if len(updatedInstances) != len(registeredNames) {
		t.Fatalf("Expected %d instances after updates, got %d", len(registeredNames), len(updatedInstances))
	}
	t.Logf("Successfully retrieved %d instances after updates", len(updatedInstances))

	// 4. Trigger a deployment
	deploymentRequest := testData.CreateDeploymentRequest(
		"v2.0.0",
		"config-v2",
		map[string]string{
			"env": "production",
		},
	)

	deployResp := testUtils.TriggerDeployment(t, deploymentRequest)
	assert.Equal(t, http.StatusCreated, deployResp.StatusCode)
	t.Logf("Successfully triggered deployment for version %s", deploymentRequest.CodeVersion)

	// 5. Triggering a deployment while pending is not possible
	deploymentRequest = testData.CreateDeploymentRequest(
		"v2.0.1",
		"config-v2",
		map[string]string{
			"env": "production",
		},
	)
	deployResp = testUtils.TriggerDeployment(t, deploymentRequest)
	assert.Equal(t, http.StatusConflict, deployResp.StatusCode)

	// Wait for the deployment to progress
	time.Sleep(500 * time.Millisecond)

	// 6. Deployment is in progress - batch size need to be respected
	deployments, deploymentResp := testUtils.GetAllDeployments(t)
	assert.Equal(t, http.StatusOK, deploymentResp.StatusCode)
	assert.Equal(t, 1, deployments.Count)

	updatedInstances = testUtils.GetAllInstances(t)
	var updatedCount int
	for _, instance := range updatedInstances {
		if instance.DesiredState.CodeVersion == "v2.0.0" {
			updatedCount++
		}
	}
	assert.Equal(t, 2, updatedCount)

	// 7. Progress the deployment - nothing should happen yet (instances haven't updated their current state)
	progress, progressResp := testUtils.ProgressDeployment(t)
	assert.Equal(t, http.StatusOK, progressResp.StatusCode)
	assert.Equal(t, deployment.Running, progress.Status)
	t.Logf("Successfully progressed deployment. Status: %v, Progress: %+v", progress.Status, progress.Progress)

	// 8. instance report their status
	for i := range progress.Progress.CurrentBatch {
		testUtils.UpdateInstance(t, registeredNames[i], testData.CreateHealthyUpdate("v2.0.0", "config-v2"))
	}

	// Wait for the deployment to progress
	time.Sleep(500 * time.Millisecond)
	_, progressResp = testUtils.ProgressDeployment(t)
	assert.Equal(t, http.StatusOK, progressResp.StatusCode)

}
