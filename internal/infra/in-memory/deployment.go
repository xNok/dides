package inmemory

import (
	"fmt"
	"sync"
	"time"

	"github.com/xnok/dides/internal/deployment"
)

// DeploymentStore is an in-memory implementation of the deployment.Store interface
type DeploymentStore struct {
	mu          sync.RWMutex
	deployments map[string]*DeploymentRecord // key is deployment ID
	nextID      int
}

// DeploymentRecord represents a stored deployment with metadata
type DeploymentRecord struct {
	ID        string                       `json:"id"`
	Request   deployment.DeploymentRequest `json:"request"`
	Status    deployment.DeploymentStatus  `json:"status"`
	CreatedAt time.Time                    `json:"created_at"`
	UpdatedAt time.Time                    `json:"updated_at"`
}

// NewDeploymentStore creates a new in-memory deployment store
func NewDeploymentStore() *DeploymentStore {
	return &DeploymentStore{
		deployments: make(map[string]*DeploymentRecord),
		nextID:      1,
	}
}

// Save stores a deployment request in memory and returns the deployment ID
func (s *DeploymentStore) Save(req deployment.DeploymentRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate a unique ID for the deployment
	id := s.generateID()

	now := time.Now()
	record := &DeploymentRecord{
		ID:        id,
		Request:   req,
		Status:    deployment.Pending, // Start with Pending status
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.deployments[id] = record
	return nil
}

// Get retrieves a deployment by ID
func (s *DeploymentStore) Get(id string) (*DeploymentRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.deployments[id]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent external modifications
	recordCopy := *record
	return &recordCopy, true
}

// GetAll returns all stored deployments
func (s *DeploymentStore) GetAll() []*DeploymentRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*DeploymentRecord, 0, len(s.deployments))
	for _, record := range s.deployments {
		// Return copies to prevent external modifications
		recordCopy := *record
		records = append(records, &recordCopy)
	}

	return records
}

// GetByStatus returns all deployments with the specified status
func (s *DeploymentStore) GetByStatus(status deployment.DeploymentStatus) []*DeploymentRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []*DeploymentRecord
	for _, record := range s.deployments {
		if record.Status == status {
			recordCopy := *record
			matches = append(matches, &recordCopy)
		}
	}

	return matches
}

// GetByLabels returns deployments that match all provided labels
func (s *DeploymentStore) GetByLabels(labels map[string]string) []*DeploymentRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []*DeploymentRecord
	for _, record := range s.deployments {
		if s.matchesLabels(record, labels) {
			recordCopy := *record
			matches = append(matches, &recordCopy)
		}
	}

	return matches
}

// UpdateStatus updates the status of a deployment
func (s *DeploymentStore) UpdateStatus(id string, status deployment.DeploymentStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.deployments[id]
	if !exists {
		return deployment.ErrDeploymentNotFound
	}

	// Update status and timestamp
	record.Status = status
	record.UpdatedAt = time.Now()

	return nil
}

// Delete removes a deployment by ID
func (s *DeploymentStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.deployments[id]
	if exists {
		delete(s.deployments, id)
	}

	return exists
}

// Count returns the number of stored deployments
func (s *DeploymentStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.deployments)
}

// generateID generates a unique deployment ID
func (s *DeploymentStore) generateID() string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("deployment-%03d", id)
}

// matchesLabels checks if a deployment has all the specified labels
func (s *DeploymentStore) matchesLabels(record *DeploymentRecord, labels map[string]string) bool {
	if record.Request.Labels == nil {
		return len(labels) == 0
	}

	for key, value := range labels {
		if recordValue, exists := record.Request.Labels[key]; !exists || recordValue != value {
			return false
		}
	}

	return true
}
