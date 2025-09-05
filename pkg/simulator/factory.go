package simulator

import (
	"fmt"

	"github.com/xnok/dides/internal/inventory"
)

// ConfigBuilder helps build test configurations programmatically
type ConfigBuilder struct {
	config *Config
}

// NewConfigBuilder creates a new ConfigBuilder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: &Config{
			Instances: make([]InstanceConfig, 0),
		},
	}
}

// AddInstance adds a single instance to the config
func (cb *ConfigBuilder) AddInstance(ip, name string, labels map[string]string) *ConfigBuilder {
	if labels == nil {
		labels = make(map[string]string)
	}

	instance := InstanceConfig{
		IP:     ip,
		Name:   name,
		Labels: labels,
	}

	cb.config.Instances = append(cb.config.Instances, instance)
	return cb
}

// AddInstancesWithPattern adds multiple instances with a naming pattern
func (cb *ConfigBuilder) AddInstancesWithPattern(baseIP string, namePrefix string, count int, labels map[string]string) *ConfigBuilder {
	for i := 1; i <= count; i++ {
		ip := fmt.Sprintf("%s.%d", baseIP, i)
		name := fmt.Sprintf("%s-%d", namePrefix, i)

		instanceLabels := make(map[string]string)
		for k, v := range labels {
			instanceLabels[k] = v
		}

		cb.AddInstance(ip, name, instanceLabels)
	}
	return cb
}

// Build returns the built configuration
func (cb *ConfigBuilder) Build() *Config {
	return cb.config
}

// Predefined configurations for common test scenarios
func NewProductionConfig() *Config {
	return NewConfigBuilder().
		AddInstancesWithPattern("192.168.1", "web", 3, map[string]string{
			"env":  "production",
			"role": "web",
		}).
		AddInstancesWithPattern("192.168.2", "api", 2, map[string]string{
			"env":  "production",
			"role": "api",
		}).
		Build()
}

func NewDevConfig() *Config {
	return NewConfigBuilder().
		AddInstance("192.168.100.1", "dev-web-1", map[string]string{
			"env":  "dev",
			"role": "web",
		}).
		AddInstance("192.168.100.2", "dev-api-1", map[string]string{
			"env":  "dev",
			"role": "api",
		}).
		Build()
}

func NewMixedEnvironmentConfig() *Config {
	return NewConfigBuilder().
		AddInstancesWithPattern("192.168.1", "prod-web", 2, map[string]string{
			"env":  "production",
			"role": "web",
		}).
		AddInstancesWithPattern("192.168.100", "dev-web", 1, map[string]string{
			"env":  "dev",
			"role": "web",
		}).
		AddInstancesWithPattern("192.168.200", "staging-web", 1, map[string]string{
			"env":  "staging",
			"role": "web",
		}).
		Build()
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// NewTestDataGenerator creates a new TestDataGenerator
func NewTestDataGenerator() *TestDataGenerator {
	return &TestDataGenerator{}
}

// CreateUpdatePatch creates an InstancePatch for testing updates
func (tdg *TestDataGenerator) CreateUpdatePatch(status *inventory.Status, labels map[string]string) inventory.InstancePatch {
	return inventory.InstancePatch{
		Status: status,
		Labels: labels,
	}
}

// CreateHealthyUpdate creates an InstancePatch with HEALTHY status and version information
func (tdg *TestDataGenerator) CreateHealthyUpdate(codeVersion, configurationVersion string) inventory.InstancePatch {
	status := inventory.HEALTHY
	return inventory.InstancePatch{
		Status:               &status,
		CodeVersion:          &codeVersion,
		ConfigurationVersion: &configurationVersion,
	}
}

// CreateDegradedUpdate creates an InstancePatch with DEGRADED status and version information
func (tdg *TestDataGenerator) CreateDegradedUpdate(codeVersion, configurationVersion string) inventory.InstancePatch {
	status := inventory.DEGRADED
	return inventory.InstancePatch{
		Status:               &status,
		CodeVersion:          &codeVersion,
		ConfigurationVersion: &configurationVersion,
	}
}

// CreateFailedUpdate creates an InstancePatch with FAILED status and version information
func (tdg *TestDataGenerator) CreateFailedUpdate(codeVersion, configurationVersion string) inventory.InstancePatch {
	status := inventory.FAILED
	return inventory.InstancePatch{
		Status:               &status,
		CodeVersion:          &codeVersion,
		ConfigurationVersion: &configurationVersion,
	}
}
