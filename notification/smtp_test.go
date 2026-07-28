package notification

import (
	"strings"
	"testing"
)

func TestSMTPInitParsesRecipients(t *testing.T) {
	smtpNotification := new(SMTP)
	err := smtpNotification.Init("smtp.example.com", "587", "user", "pass", "from@example.com", "a@example.com, b@example.com", "[backup]", false)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	if len(smtpNotification.To) != 2 {
		t.Fatalf("Init() recipients len = %d, want 2", len(smtpNotification.To))
	}
}

func TestSMTPInitRejectsMissingRecipients(t *testing.T) {
	smtpNotification := new(SMTP)
	err := smtpNotification.Init("smtp.example.com", "587", "user", "pass", "from@example.com", "", "", false)
	if err == nil {
		t.Fatal("Init() expected error for missing recipients")
	}
}

func TestBuildEmailMessageContainsHeaders(t *testing.T) {
	msg := buildEmailMessage("from@example.com", []string{"to@example.com"}, "subject", "body")
	wants := []string{"From: from@example.com", "To: to@example.com", "Subject: subject", "body"}
	for _, want := range wants {
		if !strings.Contains(msg, want) {
			t.Fatalf("buildEmailMessage() = %q, missing %q", msg, want)
		}
	}
}
