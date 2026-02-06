// Package agent provides an AI-powered agent for analyzing MOS6502 disassembly.
package agent

import (
	"errors"
	"os"
)

// Provider represents a supported LLM provider.
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
	ProviderGemini    Provider = "gemini"
)

// Default values for agent configuration.
const (
	DefaultProvider    = ProviderAnthropic
	DefaultModel       = ""    // Use provider's default
	DefaultTemperature = 0.3   // Lower temperature for more focused analysis
	DefaultMaxTokens   = 16384 // Generous limit for detailed analysis
)

// Config holds the configuration for the agent.
type Config struct {
	// Provider is the LLM provider to use ("anthropic", "openai", or "gemini").
	Provider Provider

	// APIKey is the API key for the provider.
	// If empty, will be read from environment variables.
	APIKey string

	// Model is the model name to use (e.g., "claude-sonnet-4-5-20250929", "gpt-4.1", "gemini-2.5-flash").
	// If empty, uses the provider's default model.
	Model string

	// BaseURL is the base URL for OpenAI-compatible API endpoints.
	// If empty, uses the provider's default endpoint.
	BaseURL string

	// Temperature controls randomness in the model's output (0.0-1.0).
	Temperature float64

	// MaxTokens is the maximum number of tokens in the model's response.
	MaxTokens int

	// StartAddr is the start address for analysis (0 = binary start).
	StartAddr uint16

	// EndAddr is the end address for analysis (0 = binary end).
	EndAddr uint16

	// DryRun if true, shows changes without saving to the state file.
	DryRun bool

	// Verbose if true, enables verbose output during analysis.
	Verbose bool

	// StatePath is the path to the state file, used for periodic saving.
	StatePath string

	// SaveInterval is the number of tool responses between periodic saves.
	// 0 disables periodic saving.
	SaveInterval int
}

// NewConfig creates a new Config with default values.
func NewConfig() *Config {
	return &Config{
		Provider:    DefaultProvider,
		Temperature: DefaultTemperature,
		MaxTokens:   DefaultMaxTokens,
	}
}

// Validate checks that the configuration is valid and fills in defaults.
func (c *Config) Validate() error {
	// Validate provider
	switch c.Provider {
	case ProviderAnthropic, ProviderOpenAI, ProviderGemini:
		// Valid
	case "":
		c.Provider = DefaultProvider
	default:
		return errors.New("invalid provider: must be 'anthropic', 'openai', or 'gemini'")
	}

	// Get API key from environment if not set
	if c.APIKey == "" {
		switch c.Provider {
		case ProviderAnthropic:
			c.APIKey = os.Getenv("ANTHROPIC_API_KEY")
			if c.APIKey == "" {
				c.APIKey = os.Getenv("CLAUDE_API_KEY")
			}
		case ProviderOpenAI:
			c.APIKey = os.Getenv("OPENAI_API_KEY")
		case ProviderGemini:
			c.APIKey = os.Getenv("GEMINI_API_KEY")
			if c.APIKey == "" {
				c.APIKey = os.Getenv("GOOGLE_API_KEY")
			}
		}
	}

	if c.APIKey == "" {
		return errors.New("API key required: set via --api-key flag or environment variable")
	}

	// Load OpenAI base URL from environment if not set via flag
	if c.Provider == ProviderOpenAI && c.BaseURL == "" {
		c.BaseURL = os.Getenv("OPENAI_BASE_URL")
	}

	// Set default temperature if not specified
	if c.Temperature == 0 {
		c.Temperature = DefaultTemperature
	}
	if c.Temperature < 0 || c.Temperature > 1 {
		return errors.New("temperature must be between 0.0 and 1.0")
	}

	// Set default max tokens if not specified
	if c.MaxTokens == 0 {
		c.MaxTokens = DefaultMaxTokens
	}

	return nil
}
