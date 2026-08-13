package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSlackSendPostsPayload(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	slack := &Slack{Client: server.Client()}
	if err := slack.Init(server.URL, "[backup]", false, false); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if err := slack.Send(context.Background(), false, time.UTC, "boom"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if payload["text"] == nil {
		t.Fatal("Send() missing text payload")
	}
}

func TestSlackSendReturnsHTTPError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer server.Close()

	slack := &Slack{Client: server.Client()}
	_ = slack.Init(server.URL, "", false, false)
	if err := slack.Send(context.Background(), false, time.UTC, "boom"); err == nil {
		t.Fatal("Send() expected HTTP error")
	}
}

func TestSlackInitRejectsPlaintextWebhookURL(t *testing.T) {
	slack := &Slack{}
	if err := slack.Init("http://example.com/webhook", "", false, false); err == nil {
		t.Fatal("Init() expected plaintext URL rejection")
	}
}

func TestSlackSendTruncatesAndRedactsErrorBody(t *testing.T) {
	largeBody := "mongodb://user:secret@db.example.com/test?sig=secret " + strings.Repeat("x", maxErrorSnippetBytes) // pragma: allowlist secret
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, largeBody, http.StatusBadGateway)
	}))
	defer server.Close()

	slack := &Slack{Client: server.Client()}
	if err := slack.Init(server.URL, "", false, false); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	err := slack.Send(context.Background(), false, time.UTC, "boom")
	if err == nil {
		t.Fatal("Send() expected HTTP error")
	}
	errText := err.Error()
	if strings.Contains(errText, "secret") {
		t.Fatalf("Send() error = %q, leaked secret", errText)
	}
	if !strings.Contains(errText, "mongodb://db.example.com/test?sig=REDACTED") {
		t.Fatalf("Send() error = %q, want redacted URI", errText)
	}
	if !strings.Contains(errText, "... (truncated)") {
		t.Fatalf("Send() error = %q, want truncation marker", errText)
	}
	if len(errText) > maxErrorSnippetBytes+200 {
		t.Fatalf("Send() error length = %d, want capped snippet", len(errText))
	}
}
