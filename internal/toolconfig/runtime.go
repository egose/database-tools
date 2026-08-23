package toolconfig

import (
	"fmt"
	"strconv"
	"time"
)

func ReadWorkspaceBase(env EnvReader, key string, fallback string) string {
	return envValue(env, key, fallback)
}

func ReadOptionalDuration(env EnvReader, key string) (time.Duration, error) {
	raw := envValue(env, key)
	if raw == "" {
		return 0, nil
	}

	duration, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}

	return duration, nil
}

func ReadPositiveInt(env EnvReader, key string, fallback int) (int, error) {
	raw := envValue(env, key)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}

	return value, nil
}

func ReadPositiveInt64(env EnvReader, key string, fallback int64) (int64, error) {
	raw := envValue(env, key)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}

	return value, nil
}
