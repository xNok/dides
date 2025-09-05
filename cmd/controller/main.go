package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/xnok/dides/internal/deployment"
	inmemory "github.com/xnok/dides/internal/infra/in-memory"
	"github.com/xnok/dides/internal/inventory"
)

// TODO: rework handler instanciation and move to DI
var (
	registrationService *inventory.RegistrationService
	updateService       *inventory.UpdateService
	triggerService      *deployment.TriggerService
)

func main() {
	// Initialize the in-memory store and services
	InventoryStore := inmemory.NewInventoryStore()
	registrationService = inventory.NewRegistrationService(InventoryStore)
	updateService = inventory.NewUpdateService(InventoryStore)

	// Initialize the deployment store and trigger service
	deploymentStore := inmemory.NewDeploymentStore()
	deploymentLock := inmemory.NewInMemoryLocker()
	inventoryStateService := inventory.NewStateService(InventoryStore)

	// Create rolling deployment strategy and inject it into the trigger service
	rollingStrategy := deployment.NewRollingDeployment(deploymentStore, inventoryStateService)
	triggerService = deployment.NewTriggerService(deploymentStore, deploymentLock, rollingStrategy)

	// Setup REST Router
	r := setupRouter()

	http.ListenAndServe(":3333", r)
}

// setupRouter creates and configures the HTTP router
func setupRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Inventory manages the list of instances
	r.Route("/inventory", func(r chi.Router) {
		// List all instances
		r.Get("/instances", listInstances)
		// Register an instance to the system
		r.Post("/instances/register", registerInstance)
		// Instance status update - typically instance health-check reporting
		r.Patch("/instances/{instanceID}", updateInstance)
	})

	// Interact with the deployment process
	// Since only one in-flight deployment is allowed, we skip the need for an id
	r.Route("/deploy", func(r chi.Router) {
		// Trigger a deploment
		r.Post("/", deployTrigger)
		// Get all running deployments
		r.Get("/status", deploymentStatus)
		// force the deployment to progress (mostly for testing)
		r.Post("/progress", deploymentProgress)
	})

	return r
}

// listInstances returns all instances in the inventory
func listInstances(w http.ResponseWriter, r *http.Request) {
	// Get all instances using the registration service
	instances, err := registrationService.ListAllInstances()
	if err != nil {
		http.Error(w, "Failed to retrieve instances", http.StatusInternalServerError)
		return
	}

	// Create response
	response := inventory.ListResponse{
		Instances: instances,
		Count:     len(instances),
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// registerInstance adds an instance to the inventory
func registerInstance(w http.ResponseWriter, r *http.Request) {
	var req inventory.RegistrationRequest

	// Parse the request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Register the instance using the registration service
	instance, err := registrationService.RegisterInstance(req)
	if err != nil {
		if err == inventory.ErrInvalidToken {
			http.Error(w, "Invalid registration token", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Failed to register instance", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := map[string]interface{}{
		"message":  "Instance registered successfully",
		"instance": instance,
	}

	json.NewEncoder(w).Encode(response)
}

// updateInstance modify the state of the instance record and return desired state
func updateInstance(w http.ResponseWriter, r *http.Request) {
	// Extract instance ID from URL path parameter
	instanceID := chi.URLParam(r, "instanceID")
	if instanceID == "" {
		http.Error(w, "Instance ID is required", http.StatusBadRequest)
		return
	}

	var req inventory.UpdateRequest

	// Parse the request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update the instance using the update service
	instance, err := updateService.UpdateInstance(instanceID, req)
	if err != nil {
		if err == inmemory.ErrInstanceNotFound {
			http.Error(w, "Instance not found", http.StatusNotFound)
			return
		}
		if err == inventory.ErrUpdateValidation {
			http.Error(w, "Invalid update request", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to update instance", http.StatusInternalServerError)
		return
	}

	// Return the updated instance with desired state for the instance to act upon
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"message":       "Instance updated successfully",
		"instance":      instance,
		"desired_state": instance.DesiredState,
		"current_state": instance.CurrentState,
		"update_needed": instance.CurrentState.CodeVersion != instance.DesiredState.CodeVersion ||
			instance.CurrentState.ConfigurationVersion != instance.DesiredState.ConfigurationVersion,
	}

	json.NewEncoder(w).Encode(response)
}

// deployTrigger starts a deployment to a given set of instances
func deployTrigger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req deployment.DeploymentRequest

	// Parse the request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Trigger the deployment using the trigger service
	err := triggerService.TriggerDeployment(ctx, &req)
	if err != nil {
		if errors.Is(err, deployment.ErrRolloutInProgress) {
			http.Error(w, "Deployment is already in progress", http.StatusConflict)
			return
		}

		http.Error(w, "Failed to trigger deployment", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := map[string]interface{}{
		"message": "Deployment triggered successfully",
		"request": req,
	}

	json.NewEncoder(w).Encode(response)
}

// deploymentStatus returns the status and progress of a deployment
func deploymentStatus(w http.ResponseWriter, r *http.Request) {
	// Get all running deployments
	deployments, err := triggerService.GetDeploymentStatus()
	if err != nil {
		http.Error(w, "Failed to get deployment status", http.StatusInternalServerError)
		return
	}

	// Create structured response
	response := deployment.DeploymentStatusResponse{
		Deployments: deployments,
		Count:       len(deployments),
	}

	// Return deployment status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// deploymentProgress manually progresses a deployment (normally done by background process)
func deploymentProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Progress the deployment
	record, err := triggerService.ProgressDeployment(ctx)
	if err != nil {
		http.Error(w, "Failed to progress deployment", http.StatusInternalServerError)
		return
	}

	// Create structured response
	response := deployment.DeploymentProgressResponse{
		Message:    "Deployment progressed successfully",
		Deployment: record,
		Status:     record.Status,
		Progress:   record.Progress,
	}

	// Return updated status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
