package simulator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xnok/dides/internal/deployment"
	"github.com/xnok/dides/internal/inventory"
)

// TestUtilities provides common testing utilities for simulator tests
type TestUtilities struct {
	Server *httptest.Server
	Config *Config
}

// NewTestUtilities creates a new TestUtilities instance
func NewTestUtilities(server *httptest.Server, config *Config) *TestUtilities {
	return &TestUtilities{
		Server: server,
		Config: config,
	}
}

// RegisterInstance registers a single instance with the test server
func (tu *TestUtilities) RegisterInstance(t *testing.T, instanceConfig InstanceConfig, token string) *http.Response {
	t.Helper()

	jsonData, err := instanceConfig.ToJSON(token)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(
		tu.Server.URL+"/inventory/instances/register",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	return resp
}

// RegisterAllInstances registers all instances from the config
func (tu *TestUtilities) RegisterAllInstances(t *testing.T, token string) []string {
	t.Helper()

	var registeredNames []string
	for i, instanceConfig := range tu.Config.Instances {
		t.Run(fmt.Sprintf("Register_Instance_%d_%s", i+1, instanceConfig.Name), func(t *testing.T) {
			resp := tu.RegisterInstance(t, instanceConfig, token)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("Expected status %d, got %d. Body: %s",
					http.StatusCreated, resp.StatusCode, string(body))
			}

			// Verify response
			var response map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["message"] != "Instance registered successfully" {
				t.Errorf("Unexpected message: %v", response["message"])
			}

			// Verify instance data in response
			instanceData, ok := response["instance"].(map[string]interface{})
			if !ok {
				t.Fatalf("Instance data not found in response. Full response: %+v", response)
			}

			if instanceData["IP"] != instanceConfig.IP {
				t.Errorf("Expected IP %s, got %v", instanceConfig.IP, instanceData["IP"])
			}

			if instanceData["Name"] != instanceConfig.Name {
				t.Errorf("Expected Name %s, got %v", instanceConfig.Name, instanceData["Name"])
			}

			registeredNames = append(registeredNames, instanceConfig.Name)
		})
	}

	return registeredNames
}

// UpdateInstance updates an instance with the given patch
func (tu *TestUtilities) UpdateInstance(t *testing.T, instanceName string, updates inventory.InstancePatch) *http.Response {
	t.Helper()

	updateRequest := inventory.UpdateRequest{
		Updates: updates,
	}

	jsonData, err := json.Marshal(updateRequest)
	if err != nil {
		t.Fatalf("Failed to marshal update request: %v", err)
	}

	req, err := http.NewRequest(
		"PATCH",
		tu.Server.URL+"/inventory/instances/"+instanceName,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to update instance: %v", err)
	}

	return resp
}

// GetAllInstances retrieves all instances from the inventory
func (tu *TestUtilities) GetAllInstances(t *testing.T) []*inventory.Instance {
	t.Helper()

	resp, err := http.Get(tu.Server.URL + "/inventory/instances")
	if err != nil {
		t.Fatalf("Failed to get instances: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status %d, got %d. Body: %s",
			http.StatusOK, resp.StatusCode, string(body))
	}

	var response inventory.ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return response.Instances
}

// TriggerDeployment triggers a deployment request
func (tu *TestUtilities) TriggerDeployment(t *testing.T, deploymentRequest deployment.DeploymentRequest) *http.Response {
	t.Helper()

	jsonData, err := json.Marshal(deploymentRequest)
	if err != nil {
		t.Fatalf("Failed to marshal deployment request: %v", err)
	}

	resp, err := http.Post(
		tu.Server.URL+"/deploy/",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		t.Fatalf("Failed to trigger deployment: %v", err)
	}

	return resp
}

// MakeHTTPRequest is a generic helper for making HTTP requests
func (tu *TestUtilities) MakeHTTPRequest(t *testing.T, method, path string, body interface{}) *http.Response {
	t.Helper()

	var requestBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		requestBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, tu.Server.URL+path, requestBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	return resp
}

// AssertResponseStatus asserts that the response has the expected status code
func (tu *TestUtilities) AssertResponseStatus(t *testing.T, resp *http.Response, expectedStatus int) {
	t.Helper()

	if resp.StatusCode != expectedStatus {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status %d, got %d. Body: %s",
			expectedStatus, resp.StatusCode, string(body))
	}
}

// DecodeResponse decodes a JSON response into the given interface
func (tu *TestUtilities) DecodeResponse(t *testing.T, resp *http.Response, target interface{}) {
	t.Helper()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

// GetInstanceByName finds an instance config by name
func (tu *TestUtilities) GetInstanceByName(name string) (*InstanceConfig, bool) {
	for _, instance := range tu.Config.Instances {
		if instance.Name == name {
			return &instance, true
		}
	}
	return nil, false
}

// GetInstancesByLabel finds instance configs that have the specified label
func (tu *TestUtilities) GetInstancesByLabel(key, value string) []InstanceConfig {
	var matches []InstanceConfig
	for _, instance := range tu.Config.Instances {
		if labelValue, exists := instance.Labels[key]; exists && labelValue == value {
			matches = append(matches, instance)
		}
	}
	return matches
}
