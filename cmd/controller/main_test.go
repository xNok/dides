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

	// ------------------------------------------------------
	// Simulate Deployment Logic
	// ------------------------------------------------------

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
	time.Sleep(100 * time.Millisecond)

	// 6. Deployment is in progress - batch size need to be respected
	deployments, deploymentResp := testUtils.GetAllDeployments(t)
	assert.Equal(t, http.StatusOK, deploymentResp.StatusCode)
	assert.Equal(t, 1, deployments.Count)

	updatedInstances = testUtils.GetAllInstances(t)
	inflight := getInflightInstances(updatedInstances, "v2.0.0")
	assert.Equal(t, 2, len(inflight))

	// 7. Progress the deployment - nothing should happen yet (instances haven't updated their current state)
	progress, progressResp := testUtils.ProgressDeployment(t)
	assert.Equal(t, http.StatusOK, progressResp.StatusCode)
	assert.Equal(t, deployment.Running, progress.Status)
	t.Logf("Successfully progressed deployment. Status: %v, Progress: %+v", progress.Status, progress.Progress)

	// 8. instance report their status
	for i := range inflight {
		testUtils.UpdateInstance(t, inflight[i], testData.CreateHealthyUpdate("v2.0.0", "config-v2"))
	}

	// 8.1 Progress the deployment to the next step
	progress, progressResp = testUtils.ProgressDeployment(t)
	assert.Equal(t, http.StatusOK, progressResp.StatusCode)
	assert.Equal(t, deployment.Running, progress.Status)
	assert.Equal(t, deployment.DeploymentProgress{
		TotalInstances:      3,
		CompletedInstances:  2,
		InProgressInstances: 1,
		FailedInstances:     0,
	}, progress.Progress)

	// Wait for the deployment to progress
	time.Sleep(100 * time.Millisecond)

	updatedInstances = testUtils.GetAllInstances(t)
	inflight = getInflightInstances(updatedInstances, "v2.0.0")
	assert.Equal(t, 1, len(inflight))

	// 9. instance report their status
	for i := range inflight {
		testUtils.UpdateInstance(t, inflight[i], testData.CreateHealthyUpdate("v2.0.0", "config-v2"))
	}

	// Wait for the deployment to progress
	progress, progressResp = testUtils.ProgressDeployment(t)
	assert.Equal(t, http.StatusOK, progressResp.StatusCode)
	assert.Equal(t, deployment.Completed, progress.Status)
	assert.Equal(t, deployment.DeploymentProgress{
		TotalInstances:      3,
		CompletedInstances:  3,
		InProgressInstances: 0,
		FailedInstances:     0,
	}, progress.Progress)

	// ------------------------------------------------------
	// Simulate Deployment With Failures
	// ------------------------------------------------------

	// 10. Trigger a deployment with one instance failing
	deploymentRequest = testData.CreateDeploymentRequest(
		"v3.0.0",
		"config-v3",
		map[string]string{
			"env": "production",
		},
	)

	deployResp = testUtils.TriggerDeployment(t, deploymentRequest)
	assert.Equal(t, http.StatusCreated, deployResp.StatusCode)
	t.Logf("Successfully triggered deployment for version %s", deploymentRequest.CodeVersion)

	// Wait for the deployment to progress
	time.Sleep(100 * time.Millisecond)

	deployments, deploymentResp = testUtils.GetAllDeployments(t)
	assert.Equal(t, http.StatusOK, deploymentResp.StatusCode)
	assert.Equal(t, 1, deployments.Count)

	updatedInstances = testUtils.GetAllInstances(t)
	inflight = getInflightInstances(updatedInstances, "v3.0.0")
	assert.Equal(t, 2, len(inflight))

	// 11. Progress the deployment - nothing should happen yet (instances haven't updated their current state)
	progress, progressResp = testUtils.ProgressDeployment(t)
	assert.Equal(t, http.StatusOK, progressResp.StatusCode)
	assert.Equal(t, deployment.Running, progress.Status)
	t.Logf("Successfully progressed deployment. Status: %v, Progress: %+v", progress.Status, progress.Progress)

	// 12. instance report their status - one fails
	testUtils.UpdateInstance(t, inflight[0], testData.CreateHealthyUpdate("v3.0.0", "config-v3"))
	testUtils.UpdateInstance(t, inflight[1], testData.CreateFailedUpdate("v3.0.0", "config-v3"))

	// 12.1 Progress the deployment to the next step - should trigger a rollback
	progress, progressResp = testUtils.ProgressDeployment(t)
	assert.Equal(t, http.StatusOK, progressResp.StatusCode)
	assert.Equal(t, deployment.Failed, progress.Status)
	assert.Equal(t, deployment.DeploymentProgress{
		TotalInstances:      3,
		CompletedInstances:  1, // 1 successful
		InProgressInstances: 0, // Deployment failed, no more instances being processed
		FailedInstances:     1,
	}, progress.Progress)

}

func getInflightInstances(instances []*inventory.Instance, targetVersion string) []string {
	var inflight []string
	for _, instance := range instances {
		if instance.DesiredState.CodeVersion == targetVersion && instance.CurrentState.CodeVersion != targetVersion {
			inflight = append(inflight, instance.Name)
		}
	}
	return inflight
}
