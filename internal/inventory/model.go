package inventory

import "time"

type Status int

const (
	UNKNOWN Status = iota // UNKNOWN is the default on when instance is registered/not used
	HEALTHY
	DEGRADED
	FAILED
)

// Instance represent a server or workload manages by the system
type Instance struct {
	// IP is the address of the server
	IP string
	// Name is the designation or host name of the server
	Name string
	// Labels are the key-value pairs associated with the server
	Labels map[string]string
	// LastPing is the timestamp when the instance first registered
	LastPing time.Time
	// Status represent the health check status of the instance
	Status Status
	// CodeVersion is the version of the code running on the instance
	CodeVersion string
	// ConfigurationVersion is the version of the configuration applied to the instance
	ConfigurationVersion string
}

// RegistrationRequest represents the request body for instance registration
type RegistrationRequest struct {
	Instance Instance `json:"instance"`
	Token    string   `json:"token"`
}

// InstancePatch represents partial updates to an instance, not all fields are updatable
type InstancePatch struct {
	Labels               map[string]string `json:"labels,omitempty"`
	LastPing             *time.Time        `json:"last_ping,omitempty"`
	Status               *Status           `json:"status,omitempty"`
	CodeVersion          *string           `json:"code_version,omitempty"`
	ConfigurationVersion *string           `json:"configuration_version,omitempty"`
}

// ListResponse represents the response for listing instances
type ListResponse struct {
	Instances []*Instance `json:"instances"`
	Count     int         `json:"count"`
}

type Store interface {
	Save(instance *Instance) error
	Update(key string, patch InstancePatch) (*Instance, error)
	GetAll() []*Instance
}
