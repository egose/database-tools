package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultSMTPDialTimeout  = 10 * time.Second
	defaultSMTPReadTimeout  = 10 * time.Second
	defaultSMTPWriteTimeout = 10 * time.Second
	maxErrorSnippetBytes    = 4096
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

var defaultHTTPClient httpDoer = &http.Client{}

func httpClientOrDefault(client httpDoer) httpDoer {
	if client != nil {
		return client
	}

	return defaultHTTPClient
}

func ValidateWebhookURL(raw string, allowInsecure bool, fieldName string) error {
	return validateHTTPSURL(raw, allowInsecure, fieldName)
}

func validateHTTPSURL(raw string, allowInsecure bool, fieldName string) error {
	if raw == "" {
		return nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", fieldName, err)
	}
	if parsed.Host == "" {
		return fmt.Errorf("invalid %s: host is required", fieldName)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "https":
		return nil
	case "http":
		if allowInsecure {
			return nil
		}
		return fmt.Errorf("%s must use HTTPS unless the development-only insecure HTTP override is enabled", fieldName)
	default:
		return fmt.Errorf("%s must use HTTP or HTTPS", fieldName)
	}
}

func sendWebhookJSON(ctx context.Context, client httpDoer, webhookURL string, payload any) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(contextOrBackground(ctx), http.MethodPost, webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClientOrDefault(client).Do(req)
	if err != nil {
		return fmt.Errorf("send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	snippet, readErr := readErrorSnippet(resp.Body)
	if readErr != nil {
		return fmt.Errorf("unexpected webhook response status %d and failed to read response body: %w", resp.StatusCode, readErr)
	}
	if snippet == "" {
		return fmt.Errorf("unexpected webhook response status %d", resp.StatusCode)
	}

	return fmt.Errorf("unexpected webhook response status %d: %s", resp.StatusCode, snippet)
}

func readErrorSnippet(r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxErrorSnippetBytes+1))
	if err != nil {
		return "", err
	}

	truncated := len(data) > maxErrorSnippetBytes
	if truncated {
		data = data[:maxErrorSnippetBytes]
	}

	snippet := strings.TrimSpace(redactSensitiveText(string(data)))
	if truncated {
		snippet += "... (truncated)"
	}

	return snippet, nil
}
