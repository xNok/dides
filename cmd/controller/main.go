package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	inmemory "github.com/xnok/dides/internal/infra/in-memory"
	"github.com/xnok/dides/internal/inventory"
)

// TODO: rework handler instanciation and move to DI
var (
	registrationService *inventory.RegistrationService
	updateService       *inventory.UpdateService
)

func main() {
	// Initialize the in-memory store and registration service
	store := inmemory.NewInventoryStore()
	registrationService = inventory.NewRegistrationService(store)
	updateService = inventory.NewUpdateService(store)

	// Setup REST Router
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Inventory manages the list of instances
	r.Route("/inventory", func(r chi.Router) {
		// Register an instance to the system
		r.Post("/register", registerInstance)
		// Instance status update - typically instance health-check reporting
		r.Patch("/{instanceID}", updateInstance)
	})

	http.ListenAndServe(":3333", r)
}

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

func updateInstance(w http.ResponseWriter, r *http.Request) {
	var req inventory.UpdateRequest

	// Parse the request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update the instance using the registration service
	instance, err := updateService.UpdateInstance(req)
	if err != nil {
		http.Error(w, "Failed to update instance", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"message":  "Instance updated successfully",
		"instance": instance,
	}

	json.NewEncoder(w).Encode(response)
}
