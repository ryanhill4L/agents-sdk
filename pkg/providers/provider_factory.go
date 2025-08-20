package providers

import (
	"fmt"
	"os"
)

// ProviderFactory creates providers based on configuration
type ProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// CreateProvider creates a provider based on type and configuration
func (f *ProviderFactory) CreateProvider(providerType ProviderType, options ...ProviderOption) (Provider, error) {
	switch providerType {
	case ProviderTypeOpenAI:
		return f.createOpenAIProvider(options...)
	case ProviderTypeAnthropic:
		return f.createAnthropicProvider(options...)
	case ProviderTypeGemini:
		return f.createGeminiProvider(options...)
	case ProviderTypeOllama:
		return f.createOllamaProvider(options...)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, providerType.String())
	}
}

// createOpenAIProvider creates an OpenAI provider with options
func (f *ProviderFactory) createOpenAIProvider(options ...ProviderOption) (Provider, error) {
	config := NewOpenAIConfig("")

	// Apply options
	for _, opt := range options {
		if err := opt.Apply(config); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	return NewOpenAIProvider(config)
}

// createAnthropicProvider creates an Anthropic provider with options
func (f *ProviderFactory) createAnthropicProvider(options ...ProviderOption) (Provider, error) {
	config := NewAnthropicConfig("")

	// Apply options
	for _, opt := range options {
		if err := opt.Apply(config); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	return NewAnthropicProvider(config)
}

// createGeminiProvider creates a Gemini provider with options
func (f *ProviderFactory) createGeminiProvider(options ...ProviderOption) (Provider, error) {
	config := NewGeminiConfig("")

	// Apply options
	for _, opt := range options {
		if err := opt.Apply(config); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	return NewGeminiProvider(config)
}

// ProviderOption represents configuration options for providers
type ProviderOption interface {
	Apply(config interface{}) error
}

// WithAPIKey sets the API key for the provider
type WithAPIKey string

func (w WithAPIKey) Apply(config interface{}) error {
	switch c := config.(type) {
	case *OpenAIConfig:
		c.APIKey = string(w)
	case *AnthropicConfig:
		c.APIKey = string(w)
	case *GeminiConfig:
		c.APIKey = string(w)
	default:
		return fmt.Errorf("unsupported config type for API key option")
	}
	return nil
}

// WithBaseURL sets the base URL for the provider
type WithBaseURL string

func (w WithBaseURL) Apply(config interface{}) error {
	switch c := config.(type) {
	case *OpenAIConfig:
		c.BaseURL = string(w)
	case *AnthropicConfig:
		c.BaseURL = string(w)
	case *GeminiConfig:
		c.BaseURL = string(w)
	case *OllamaConfig:
		c.Host = string(w)
	default:
		return fmt.Errorf("unsupported config type for base URL option")
	}
	return nil
}

// WithDebug enables debug logging for the provider
type WithDebug bool

func (w WithDebug) Apply(config interface{}) error {
	switch c := config.(type) {
	case *OpenAIConfig:
		c.Debug = bool(w)
	case *AnthropicConfig:
		c.Debug = bool(w)
	case *GeminiConfig:
		c.Debug = bool(w)
	case *OllamaConfig:
		c.Debug = bool(w)
	default:
		return fmt.Errorf("unsupported config type for debug option")
	}
	return nil
}

// WithOrganization sets the organization for OpenAI provider
type WithOrganization string

func (w WithOrganization) Apply(config interface{}) error {
	if c, ok := config.(*OpenAIConfig); ok {
		c.Organization = string(w)
		return nil
	}
	return fmt.Errorf("organization option only applies to OpenAI config")
}

// WithProject sets the project for OpenAI provider
type WithProject string

func (w WithProject) Apply(config interface{}) error {
	if c, ok := config.(*OpenAIConfig); ok {
		c.Project = string(w)
		return nil
	}
	return fmt.Errorf("project option only applies to OpenAI config")
}

// Convenience functions for creating providers

// NewOpenAIProviderFromEnv creates an OpenAI provider using environment variables
func NewOpenAIProviderFromEnv() (Provider, error) {
	factory := NewProviderFactory()
	return factory.CreateProvider(ProviderTypeOpenAI, WithAPIKey(os.Getenv("OPENAI_API_KEY")))
}

// NewAnthropicProviderFromEnv creates an Anthropic provider using environment variables
func NewAnthropicProviderFromEnv() (Provider, error) {
	factory := NewProviderFactory()
	return factory.CreateProvider(ProviderTypeAnthropic, WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")))
}

// NewOpenAIProviderWithKey creates an OpenAI provider with the given API key
func NewOpenAIProviderWithKey(apiKey string) (Provider, error) {
	factory := NewProviderFactory()
	return factory.CreateProvider(ProviderTypeOpenAI, WithAPIKey(apiKey))
}

// NewAnthropicProviderWithKey creates an Anthropic provider with the given API key
func NewAnthropicProviderWithKey(apiKey string) (Provider, error) {
	factory := NewProviderFactory()
	return factory.CreateProvider(ProviderTypeAnthropic, WithAPIKey(apiKey))
}

// NewGeminiProviderFromEnv creates a Gemini provider using environment variables
func NewGeminiProviderFromEnv() (Provider, error) {
	factory := NewProviderFactory()
	return factory.CreateProvider(ProviderTypeGemini, WithAPIKey(os.Getenv("GEMINI_API_KEY")))
}

// NewGeminiProviderWithKey creates a Gemini provider with the given API key
func NewGeminiProviderWithKey(apiKey string) (Provider, error) {
	factory := NewProviderFactory()
	return factory.CreateProvider(ProviderTypeGemini, WithAPIKey(apiKey))
}

// createOllamaProvider creates an Ollama provider with options
func (f *ProviderFactory) createOllamaProvider(options ...ProviderOption) (Provider, error) {
	config := NewOllamaConfig("")

	// Apply options
	for _, opt := range options {
		if err := opt.Apply(config); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	return NewOllamaProvider(config.Host)
}

// NewOllamaProviderFromEnv creates an Ollama provider using environment variables
func NewOllamaProviderFromEnv() (Provider, error) {
	factory := NewProviderFactory()
	return factory.CreateProvider(ProviderTypeOllama)
}

// NewOllamaProviderWithHost creates an Ollama provider with the given host
func NewOllamaProviderWithHost(host string) (Provider, error) {
	factory := NewProviderFactory()
	return factory.CreateProvider(ProviderTypeOllama, WithBaseURL(host))
}
