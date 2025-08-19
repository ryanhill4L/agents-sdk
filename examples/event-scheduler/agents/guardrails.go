package agents

import (
	"fmt"
	"strings"
)

// PrivacyGuardrail prevents access to sensitive information
type PrivacyGuardrail struct{}

func NewPrivacyGuardrail() *PrivacyGuardrail {
	return &PrivacyGuardrail{}
}

func (p *PrivacyGuardrail) Name() string {
	return "privacy_check"
}

func (p *PrivacyGuardrail) Description() string {
	return "Prevents access to sensitive personal information and unauthorized data modifications"
}

func (p *PrivacyGuardrail) Validate(content string) error {
	lowercaseContent := strings.ToLower(content)

	// Check for potentially sensitive requests
	sensitiveKeywords := []string{
		"password", "ssn", "social security", "credit card", "bank account",
		"delete all", "drop table", "truncate", "update users set",
		"personal phone", "home address", "salary", "wages",
	}

	for _, keyword := range sensitiveKeywords {
		if strings.Contains(lowercaseContent, keyword) {
			return fmt.Errorf("request contains potentially sensitive information or unsafe operations: %s", keyword)
		}
	}

	return nil
}
