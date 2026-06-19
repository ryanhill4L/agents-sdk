package providers

import (
	"fmt"
	"os"
	"strings"
)

// Resolve constructs a provider from a short name using credentials found in the
// environment. Supported names: "openai", "anthropic", "gemini", "ollama".
//
// An empty name falls back to PROVIDER, then auto-detects based on which API key
// is set, and finally defaults to "openai".
func Resolve(name string) (Provider, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = strings.ToLower(strings.TrimSpace(os.Getenv("PROVIDER")))
	}
	if name == "" {
		name = autodetect()
	}

	switch name {
	case "openai":
		return NewOpenAIProviderFromEnv()
	case "anthropic", "claude":
		return NewAnthropicProviderFromEnv()
	case "gemini", "google":
		return NewGeminiProviderFromEnv()
	case "ollama":
		return NewOllamaProviderFromEnv()
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedProvider, name)
	}
}

// autodetect picks a provider based on which credentials are present.
func autodetect() string {
	switch {
	case os.Getenv("OPENAI_API_KEY") != "":
		return "openai"
	case os.Getenv("ANTHROPIC_API_KEY") != "":
		return "anthropic"
	case os.Getenv("GEMINI_API_KEY") != "":
		return "gemini"
	case os.Getenv("OLLAMA_HOST") != "":
		return "ollama"
	default:
		return "openai"
	}
}
