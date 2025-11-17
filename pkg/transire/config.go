package transire

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the Transire application configuration
type Config struct {
	Name     string `yaml:"name"`
	Language string `yaml:"language"`
	Cloud    string `yaml:"cloud"`
	Runtime  string `yaml:"runtime"`
	IaC      string `yaml:"iac"`
	CI       string `yaml:"ci"`

	Lambda      LambdaConfig              `yaml:"lambda"`
	Functions   map[string]FunctionConfig `yaml:"functions"`
	Environment map[string]string         `yaml:"environment"`
	VPC         *VPCConfig                `yaml:"vpc,omitempty"`

	ExistingResources ExistingResourcesConfig `yaml:"existing_resources"`
	Queues           map[string]QueueConfig   `yaml:"queues"`
	Schedules        map[string]ScheduleConfig `yaml:"schedules"`

	CDKExtensions []ExtensionConfig `yaml:"cdk_extensions"`
	Development   DevelopmentConfig `yaml:"development"`
}

// LambdaConfig configures Lambda function defaults
type LambdaConfig struct {
	Architecture   string `yaml:"architecture"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	MemoryMB      int    `yaml:"memory_mb"`
}

// FunctionConfig defines a function group
type FunctionConfig struct {
	Include             []IncludeSpec     `yaml:"include"`
	MemoryMB           int               `yaml:"memory_mb,omitempty"`
	TimeoutSeconds     int               `yaml:"timeout_seconds,omitempty"`
	ReservedConcurrency *int              `yaml:"reserved_concurrency,omitempty"`
	Environment        map[string]string `yaml:"environment,omitempty"`
}

// VPCConfig configures VPC settings
type VPCConfig struct {
	SubnetIDs        []string `yaml:"subnet_ids"`
	SecurityGroupIDs []string `yaml:"security_group_ids"`
}

// ExistingResourcesConfig references existing AWS resources
type ExistingResourcesConfig struct {
	DynamoDBTables []ExistingResource `yaml:"dynamodb_tables"`
	S3Buckets     []ExistingResource `yaml:"s3_buckets"`
	Secrets       []ExistingResource `yaml:"secrets"`
}

// ExistingResource represents a reference to existing infrastructure
type ExistingResource struct {
	Name        string   `yaml:"name"`
	ARN         string   `yaml:"arn"`
	Permissions []string `yaml:"permissions"`
}

// ExtensionConfig references CDK extension files
type ExtensionConfig struct {
	File string `yaml:"file"`
}

// DevelopmentConfig configures local development
type DevelopmentConfig struct {
	HTTPPort        int    `yaml:"http_port"`
	QueuePort       int    `yaml:"queue_port"`
	SchedulerPort   int    `yaml:"scheduler_port"`
	AutoReload      bool   `yaml:"auto_reload"`
	LogLevel        string `yaml:"log_level"`
	MockAWSServices bool   `yaml:"mock_aws_services"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// Default to transire.yaml in current directory if no path specified
	if path == "" {
		path = "transire.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Set defaults
	config.setDefaults()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to a YAML file
func (c *Config) SaveConfig(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SetDefaults sets default values for configuration (exported version)
func (c *Config) SetDefaults() {
	c.setDefaults()
}

// setDefaults sets default values for configuration
func (c *Config) setDefaults() {
	if c.Language == "" {
		c.Language = "go"
	}
	if c.Cloud == "" {
		c.Cloud = "aws"
	}
	if c.Runtime == "" {
		c.Runtime = "lambda"
	}
	if c.IaC == "" {
		c.IaC = "cdk"
	}
	if c.CI == "" {
		c.CI = "github"
	}

	// Lambda defaults
	if c.Lambda.Architecture == "" {
		c.Lambda.Architecture = "arm64"
	}
	if c.Lambda.TimeoutSeconds == 0 {
		c.Lambda.TimeoutSeconds = 30
	}
	if c.Lambda.MemoryMB == 0 {
		c.Lambda.MemoryMB = 128
	}

	// Development defaults
	if c.Development.HTTPPort == 0 {
		c.Development.HTTPPort = 3000
	}
	if c.Development.QueuePort == 0 {
		c.Development.QueuePort = 4000
	}
	if c.Development.SchedulerPort == 0 {
		c.Development.SchedulerPort = 5000
	}
	if c.Development.LogLevel == "" {
		c.Development.LogLevel = "info"
	}

	// Environment defaults
	if c.Environment == nil {
		c.Environment = make(map[string]string)
	}

	// Function defaults - create main function if none specified
	if c.Functions == nil || len(c.Functions) == 0 {
		c.Functions = map[string]FunctionConfig{
			"main": {
				Include: []IncludeSpec{
					{
						HTTPHandlers:     "*",
						QueueHandlers:    "*",
						ScheduleHandlers: "*",
					},
				},
			},
		}
	}
}

// Validate checks configuration for errors
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Validate supported values for MVP
	if c.Language != "go" {
		return fmt.Errorf("unsupported language: %s (only 'go' supported in MVP)", c.Language)
	}
	if c.Cloud != "aws" {
		return fmt.Errorf("unsupported cloud: %s (only 'aws' supported in MVP)", c.Cloud)
	}
	if c.Runtime != "lambda" {
		return fmt.Errorf("unsupported runtime: %s (only 'lambda' supported in MVP)", c.Runtime)
	}
	if c.IaC != "cdk" {
		return fmt.Errorf("unsupported IaC: %s (only 'cdk' supported in MVP)", c.IaC)
	}

	// Validate Lambda architecture
	if c.Lambda.Architecture != "arm64" {
		return fmt.Errorf("unsupported Lambda architecture: %s (only 'arm64' supported)", c.Lambda.Architecture)
	}

	// Validate function groups
	if len(c.Functions) == 0 {
		return fmt.Errorf("at least one function group must be defined")
	}

	for name, fn := range c.Functions {
		if len(fn.Include) == 0 {
			return fmt.Errorf("function group '%s' must include at least one handler type", name)
		}

		// Validate memory and timeout ranges
		if fn.MemoryMB != 0 && (fn.MemoryMB < 128 || fn.MemoryMB > 10240) {
			return fmt.Errorf("function group '%s' memory must be between 128 and 10240 MB", name)
		}
		if fn.TimeoutSeconds != 0 && (fn.TimeoutSeconds < 1 || fn.TimeoutSeconds > 900) {
			return fmt.Errorf("function group '%s' timeout must be between 1 and 900 seconds", name)
		}
	}

	return nil
}

// GetFunctionForHandler returns the function group name for a given handler
func (c *Config) GetFunctionForHandler(handlerType, handlerName string) string {
	for fnName, fn := range c.Functions {
		for _, include := range fn.Include {
			switch handlerType {
			case "http":
				if c.matchesInclude(include.HTTPHandlers, handlerName) {
					return fnName
				}
			case "queue":
				if c.matchesInclude(include.QueueHandlers, handlerName) {
					return fnName
				}
			case "schedule":
				if c.matchesInclude(include.ScheduleHandlers, handlerName) {
					return fnName
				}
			}
		}
	}

	// Default to first function if no specific match
	for fnName := range c.Functions {
		return fnName
	}

	return "main"
}

// matchesInclude checks if a handler matches an include specification
func (c *Config) matchesInclude(include interface{}, handlerName string) bool {
	if include == nil {
		return false
	}

	switch v := include.(type) {
	case string:
		return v == "*" || v == handlerName
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok && (str == "*" || str == handlerName) {
				return true
			}
		}
	case []string:
		for _, item := range v {
			if item == "*" || item == handlerName {
				return true
			}
		}
	}

	return false
}