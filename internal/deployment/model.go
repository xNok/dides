package deployment

type DeploymentStatus int

const (
	Unknown DeploymentStatus = iota
	Running
	Completed
	Failed
)

// DeploymentRequest represents a request to deploy a new version to set of instance
type DeploymentRequest struct {
	CodeVersion          string `json:"code_version"`
	ConfigurationVersion string `json:"configuration_version"`

	// Labels to filter deployments
	Labels map[string]string `json:"labels"`
	// Configuration for deployment
	Configuration Configuration `json:"configuration"`
}

type Configuration struct {
	// BatchSize indicate how many updates run concurrently across nodes, but the batch size must be respected
	BatchSize int `json:"batch_size"`
	// FailureThreshold Abort the rollout if failures exceed a limit (either total or percentage)
	FailureThreshold int `json:"failure_threshold"`
}

type DeploymentRecord struct {
	Request DeploymentRequest `json:"request"`
	Status  DeploymentStatus  `json:"status"`
}
