package utils

import (
	"os"
)

// Env represents an environment variable manager with prefixes.
type Env struct {
	Prefixes []string
}

// NewEnv creates a new Env instance with the specified prefixes.
func NewEnv(prefixes ...string) *Env {
	return &Env{Prefixes: prefixes}
}

// GetValue retrieves the value of an environment variable with fallback prefixes and an optional default value.
func (e *Env) GetValue(key string, defaultValues ...string) string {
	// Iterate over prefixes and try to retrieve the value
	for _, prefix := range e.Prefixes {
		fullKey := prefix + key
		value := os.Getenv(fullKey)
		if value != "" {
			return value
		}
	}

	// If no value is found and default values are provided, use the first non-empty default value
	for _, defaultValue := range defaultValues {
		if defaultValue != "" {
			return defaultValue
		}
	}

	// If no value is found and no default value is provided, return an empty string
	return ""
}
