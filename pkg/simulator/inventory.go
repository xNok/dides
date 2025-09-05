package simulator

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xnok/dides/internal/inventory"
	"gopkg.in/yaml.v2"
)

// Config represents the structure of simulator.config.yaml
type Config struct {
	Instances []InstanceConfig `yaml:"instances"`
}

// InstanceConfig represents an instance configuration from YAML
type InstanceConfig struct {
	IP     string            `yaml:"ip"`
	Name   string            `yaml:"name"`
	Labels map[string]string `yaml:"labels"`
}

// LoadConfigFromFile loads simulator configuration from a YAML file
func LoadConfigFromFile(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// ToInventoryInstance converts an InstanceConfig to an inventory.Instance
func (ic InstanceConfig) ToInventoryInstance() inventory.Instance {
	return inventory.Instance{
		IP:     ic.IP,
		Name:   ic.Name,
		Labels: ic.Labels,
	}
}

// ToRegistrationRequest converts an InstanceConfig to a registration request
func (ic InstanceConfig) ToRegistrationRequest(token string) inventory.RegistrationRequest {
	return inventory.RegistrationRequest{
		Instance: ic.ToInventoryInstance(),
		Token:    token,
	}
}

// ToJSON converts an InstanceConfig to JSON bytes for HTTP requests
func (ic InstanceConfig) ToJSON(token string) ([]byte, error) {
	req := ic.ToRegistrationRequest(token)
	return json.Marshal(req)
}
