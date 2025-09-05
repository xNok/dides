package inmemory

import (
	"fmt"
	"sync"
	"time"

	"github.com/xnok/dides/internal/deployment"
)

// deploymentEntry represents an internal storage entry with metadata
type deploymentEntry struct {
	ID        string
	Record    *deployment.DeploymentRecord
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DeploymentRecordWithID extends deployment.DeploymentRecord with ID and timestamps
type DeploymentRecordWithID struct {
	ID        string                       `json:"id"`
	Request   deployment.DeploymentRequest `json:"request"`
	Status    deployment.DeploymentStatus  `json:"status"`
	CreatedAt time.Time                    `json:"created_at"`
	UpdatedAt time.Time                    `json:"updated_at"`
}

// DeploymentStore is an in-memory implementation of the deployment.Store interface
type DeploymentStore struct {
	mu          sync.RWMutex
	deployments map[string]*deploymentEntry // key is deployment ID
	nextID      int
}

// NewDeploymentStore creates a new in-memory deployment store
func NewDeploymentStore() *DeploymentStore {
	return &DeploymentStore{
		deployments: make(map[string]*deploymentEntry),
		nextID:      1,
	}
}

// Save stores a deployment request in memory and returns the deployment ID
func (s *DeploymentStore) Save(req *deployment.DeploymentRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate a unique ID for the deployment
	id := s.generateID()

	now := time.Now()
	entry := &deploymentEntry{
		ID: id,
		Record: &deployment.DeploymentRecord{
			Request: *req,
			Status:  deployment.Running,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.deployments[id] = entry
	return nil
}

// Get retrieves a deployment by ID
func (s *DeploymentStore) Get(id string) (*DeploymentRecordWithID, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.deployments[id]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent external modifications
	record := &DeploymentRecordWithID{
		ID:        entry.ID,
		Request:   entry.Record.Request,
		Status:    entry.Record.Status,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
	}
	return record, true
}

// GetAll returns all stored deployments
func (s *DeploymentStore) GetAll() []*DeploymentRecordWithID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*DeploymentRecordWithID, 0, len(s.deployments))
	for _, entry := range s.deployments {
		// Return copies to prevent external modifications
		record := &DeploymentRecordWithID{
			ID:        entry.ID,
			Request:   entry.Record.Request,
			Status:    entry.Record.Status,
			CreatedAt: entry.CreatedAt,
			UpdatedAt: entry.UpdatedAt,
		}
		records = append(records, record)
	}

	return records
}

// GetByStatus returns all deployments with the specified status
func (s *DeploymentStore) GetByStatus(status deployment.DeploymentStatus) ([]*deployment.DeploymentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []*deployment.DeploymentRecord
	for _, entry := range s.deployments {
		if entry.Record.Status == status {
			// Return a copy to prevent external modifications
			recordCopy := *entry.Record
			matches = append(matches, &recordCopy)
		}
	}

	return matches, nil
}

// GetByLabels returns deployments that match all provided labels
func (s *DeploymentStore) GetByLabels(labels map[string]string) []*deployment.DeploymentRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []*deployment.DeploymentRecord
	for _, entry := range s.deployments {
		if s.matchesLabels(entry, labels) {
			recordCopy := *entry.Record
			matches = append(matches, &recordCopy)
		}
	}

	return matches
}

// UpdateStatus updates the status of a deployment
func (s *DeploymentStore) UpdateStatus(id string, status deployment.DeploymentStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.deployments[id]
	if !exists {
		return deployment.ErrDeploymentNotFound
	}

	// Update status and timestamp
	entry.Record.Status = status
	entry.UpdatedAt = time.Now()

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
func (s *DeploymentStore) matchesLabels(entry *deploymentEntry, labels map[string]string) bool {
	if entry.Record.Request.Labels == nil {
		return len(labels) == 0
	}

	for key, value := range labels {
		if recordValue, exists := entry.Record.Request.Labels[key]; !exists || recordValue != value {
			return false
		}
	}

	return true
}
