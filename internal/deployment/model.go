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

// DeploymentRecord represents an record for the deployment history
type DeploymentRecord struct {
	ID      string            `json:"id"`
	Request DeploymentRequest `json:"request"`
	Status  DeploymentStatus  `json:"status"`
	// Progress tracking for rolling deployments
	Progress DeploymentProgress `json:"progress"`
}

// DeploymentProgress tracks the progress of a deployment
type DeploymentProgress struct {
	// Total number of instances that match the deployment labels
	TotalInstances int `json:"total_instances"`
	// Number of instances currently being updated (in progress)
	InProgressInstances int `json:"in_progress_instances"`
	// Number of instances successfully updated
	CompletedInstances int `json:"completed_instances"`
	// Number of instances that failed to update
	FailedInstances int `json:"failed_instances"`
	// Instances currently in the active batch
	CurrentBatch []string `json:"current_batch"`
}

// DeploymentProgressResponse represents the response from progressing a deployment
type DeploymentProgressResponse struct {
	Message    string             `json:"message"`
	Deployment *DeploymentRecord  `json:"deployment"`
	Status     DeploymentStatus   `json:"status"`
	Progress   DeploymentProgress `json:"progress"`
}

// DeploymentStatusResponse represents the response from getting deployment status
type DeploymentStatusResponse struct {
	Deployments []*DeploymentRecord `json:"deployments"`
	Count       int                 `json:"count"`
}
