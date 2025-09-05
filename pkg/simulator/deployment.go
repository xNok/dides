package simulator

import (
	"github.com/xnok/dides/internal/deployment"
)

// DeploymentTestUtilities provides utilities for testing deployment functionality
type DeploymentTestUtilities struct{}

// NewDeploymentTestUtilities creates a new DeploymentTestUtilities instance
func NewDeploymentTestUtilities() *DeploymentTestUtilities {
	return &DeploymentTestUtilities{}
}

// CreateBasicDeploymentRequest creates a basic deployment request for testing
func (dtu *DeploymentTestUtilities) CreateBasicDeploymentRequest(codeVersion, configVersion string, labels map[string]string) deployment.DeploymentRequest {
	if labels == nil {
		labels = make(map[string]string)
	}

	return deployment.DeploymentRequest{
		CodeVersion:          codeVersion,
		ConfigurationVersion: configVersion,
		Labels:               labels,
		Configuration: deployment.Configuration{
			BatchSize:        1,
			FailureThreshold: 0,
		},
	}
}

// CreateProductionDeploymentRequest creates a production-ready deployment request
func (dtu *DeploymentTestUtilities) CreateProductionDeploymentRequest(codeVersion, configVersion string) deployment.DeploymentRequest {
	return deployment.DeploymentRequest{
		CodeVersion:          codeVersion,
		ConfigurationVersion: configVersion,
		Labels: map[string]string{
			"env": "production",
		},
		Configuration: deployment.Configuration{
			BatchSize:        2,
			FailureThreshold: 1,
		},
	}
}

// CreateDevDeploymentRequest creates a development deployment request
func (dtu *DeploymentTestUtilities) CreateDevDeploymentRequest(codeVersion, configVersion string) deployment.DeploymentRequest {
	return deployment.DeploymentRequest{
		CodeVersion:          codeVersion,
		ConfigurationVersion: configVersion,
		Labels: map[string]string{
			"env": "dev",
		},
		Configuration: deployment.Configuration{
			BatchSize:        5,
			FailureThreshold: 2,
		},
	}
}

// CreateCanaryDeploymentRequest creates a canary deployment request
func (dtu *DeploymentTestUtilities) CreateCanaryDeploymentRequest(codeVersion, configVersion string, targetLabels map[string]string) deployment.DeploymentRequest {
	if targetLabels == nil {
		targetLabels = make(map[string]string)
	}

	return deployment.DeploymentRequest{
		CodeVersion:          codeVersion,
		ConfigurationVersion: configVersion,
		Labels:               targetLabels,
		Configuration: deployment.Configuration{
			BatchSize:        1,
			FailureThreshold: 0,
		},
	}
}
