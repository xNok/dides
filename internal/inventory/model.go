package inventory

import "time"

type Status int

const (
	UNKNOWN Status = iota // UNKNOWN is the default on when instance is registered/not used
	HEALTHY
	FAILED
)

// Instance represent a server or workload manages by the system
type Instance struct {
	// ------------------------------------------------------
	// Instance Metadata
	// ------------------------------------------------------
	// IP is the address of the server
	IP string
	// Name is the designation or host name of the server
	Name string
	// Labels are the key-value pairs associated with the server
	Labels map[string]string
	// LastPing is the timestamp when the instance first registered

	// ------------------------------------------------------
	// Instance State/Status
	// ------------------------------------------------------
	// LastPing is the timestamp when the instance last pinged the server
	LastPing time.Time
	// Status represent the health check status of the instance
	Status Status

	CurrentState State `json:"current_state"`
	DesiredState State `json:"desired_state"`
}

// State represent the version deployed on the instance
type State struct {
	CodeVersion          string
	ConfigurationVersion string
}

// RegistrationRequest represents the request body for instance registration
type RegistrationRequest struct {
	Instance Instance `json:"instance"`
	Token    string   `json:"token"`
}

// InstancePatch represents partial updates to an instance, not all fields are updatable
type InstancePatch struct {
	Labels       map[string]string `json:"labels,omitempty"`
	LastPing     *time.Time        `json:"last_ping,omitempty"`
	Status       *Status           `json:"status,omitempty"`
	CurrentState *State            `json:"current_state,omitempty"`
	DesiredState *State            `json:"desired_state,omitempty"`
}

// ListResponse represents the response for listing instances
type ListResponse struct {
	Instances []*Instance `json:"instances"`
	Count     int         `json:"count"`
}

// GetNeedingUpdateOptions contains options for GetNeedingUpdate query
type GetNeedingUpdateOptions struct {
	Limit int // Maximum number of instances to return. 0 or negative means no limit
}

type Store interface {
	// CRUD
	Save(instance *Instance) error
	Update(key string, patch InstancePatch) (*Instance, error)
	GetAll() []*Instance
	GetByLabels(labels map[string]string) []*Instance
	CountByLabels(labels map[string]string) (int, error)

	// Search for update
	GetNeedingUpdate(labels map[string]string, desiredState State, opts *GetNeedingUpdateOptions) ([]*Instance, error)
	CountNeedingUpdate(labels map[string]string, desiredState State) (int, error)
	// CountInProgress returns the total number of instances currently being updated (desiredState == targetState but currentState != desiredState)
	CountInProgress(labels map[string]string, desiredState State) (int, error)
	// CountCompleted returns the total number of instances that have completed the update to the desired state
	CountCompleted(labels map[string]string, desiredState State) (int, error)
	// CountFailed returns the total number of instances that have failed the update to the desired state
	CountFailed(labels map[string]string, desiredState State) (int, error)
	// ResetFailedInstances resets the status of failed instances matching the labels to UNKNOWN
	ResetFailedInstances(labels map[string]string) error
}
