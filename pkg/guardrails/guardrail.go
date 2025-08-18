package guardrails

// Guardrail represents a safety or validation check
type Guardrail interface {
	// Validate checks if the content passes the guardrail
	Validate(content string) error

	// Name returns the guardrail name for identification
	Name() string

	// Description returns a description of what this guardrail validates
	Description() string
}