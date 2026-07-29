package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSlackSendPostsPayload(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	slack := &Slack{}
	if err := slack.Init(server.URL, "[backup]", false); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if err := slack.Send(false, time.UTC, "boom"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if payload["text"] == nil {
		t.Fatal("Send() missing text payload")
	}
}

func TestSlackSendReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer server.Close()

	slack := &Slack{}
	_ = slack.Init(server.URL, "", false)
	if err := slack.Send(false, time.UTC, "boom"); err == nil {
		t.Fatal("Send() expected HTTP error")
	}
}
