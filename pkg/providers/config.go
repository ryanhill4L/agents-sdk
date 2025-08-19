package providers

import (
	"os"
	"time"
)

// ProviderType represents the type of LLM provider
type ProviderType int

const (
	ProviderTypeOpenAI ProviderType = iota
	ProviderTypeAnthropic
	ProviderTypeGemini
)

func (p ProviderType) String() string {
	switch p {
	case ProviderTypeOpenAI:
		return "openai"
	case ProviderTypeAnthropic:
		return "anthropic"
	case ProviderTypeGemini:
		return "gemini"
	default:
		return "unknown"
	}
}

// ProviderConfig holds common configuration for all providers
type ProviderConfig struct {
	// APIKey for the provider service
	APIKey string

	// BaseURL for custom API endpoints (optional)
	BaseURL string

	// Timeout for API requests
	Timeout time.Duration

	// MaxRetries for failed requests
	MaxRetries int

	// Debug enables detailed logging
	Debug bool
}

// OpenAIConfig holds OpenAI-specific configuration
type OpenAIConfig struct {
	ProviderConfig
	
	// Organization ID (optional)
	Organization string
	
	// Project ID (optional)
	Project string
}

// AnthropicConfig holds Anthropic-specific configuration
type AnthropicConfig struct {
	ProviderConfig
	
	// Version specifies the API version (optional)
	Version string
	
	// Beta features to enable (optional)
	Beta []string
}

// GeminiConfig holds Google Gemini-specific configuration
type GeminiConfig struct {
	ProviderConfig
	
	// Project ID for Google Cloud (optional if using API key auth)
	ProjectID string
	
	// Location for the model (optional)
	Location string
}

// DefaultConfig returns a default configuration with common settings
func DefaultConfig() ProviderConfig {
	return ProviderConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		Debug:      false,
	}
}

// NewOpenAIConfig creates OpenAI configuration with defaults
func NewOpenAIConfig(apiKey string) *OpenAIConfig {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	
	return &OpenAIConfig{
		ProviderConfig: ProviderConfig{
			APIKey:     apiKey,
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			Debug:      false,
		},
		Organization: os.Getenv("OPENAI_ORG_ID"),
		Project:      os.Getenv("OPENAI_PROJECT_ID"),
	}
}

// NewAnthropicConfig creates Anthropic configuration with defaults
func NewAnthropicConfig(apiKey string) *AnthropicConfig {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	
	return &AnthropicConfig{
		ProviderConfig: ProviderConfig{
			APIKey:     apiKey,
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			Debug:      false,
		},
		Version: "2023-06-01",
	}
}

// NewGeminiConfig creates Gemini configuration with defaults
func NewGeminiConfig(apiKey string) *GeminiConfig {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	
	return &GeminiConfig{
		ProviderConfig: ProviderConfig{
			APIKey:     apiKey,
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			Debug:      false,
		},
		ProjectID: os.Getenv("GOOGLE_CLOUD_PROJECT"),
		Location:  "us-central1", // Default location
	}
}

// Validate checks if the configuration is valid
func (c *ProviderConfig) Validate() error {
	if c.APIKey == "" {
		return ErrMissingAPIKey
	}
	if c.Timeout <= 0 {
		return ErrInvalidTimeout
	}
	if c.MaxRetries < 0 {
		return ErrInvalidMaxRetries
	}
	return nil
}

// Validate checks if the OpenAI configuration is valid
func (c *OpenAIConfig) Validate() error {
	return c.ProviderConfig.Validate()
}

// Validate checks if the Anthropic configuration is valid
func (c *AnthropicConfig) Validate() error {
	return c.ProviderConfig.Validate()
}

// Validate checks if the Gemini configuration is valid
func (c *GeminiConfig) Validate() error {
	return c.ProviderConfig.Validate()
}