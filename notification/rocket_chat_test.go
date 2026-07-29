package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRocketChatSendPostsPayload(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rc := &RocketChat{}
	if err := rc.Init(server.URL, "[backup]", false); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if err := rc.Send(false, time.UTC, "boom"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if payload["text"] == nil {
		t.Fatal("Send() missing text payload")
	}
}

func TestRocketChatSendReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer server.Close()

	rc := &RocketChat{}
	_ = rc.Init(server.URL, "", false)
	if err := rc.Send(false, time.UTC, "boom"); err == nil {
		t.Fatal("Send() expected HTTP error")
	}
}
