package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Slack struct {
	WebhookURL          string
	WebhookPrefix       string
	notifyOnFailureOnly bool
}

func (s *Slack) Init(webhookURL, webhookPrefix string, notifyOnFailureOnly bool) error {
	s.WebhookURL = webhookURL
	s.WebhookPrefix = webhookPrefix
	s.notifyOnFailureOnly = notifyOnFailureOnly
	return nil
}

func (s *Slack) Send(success bool, loc *time.Location, filenameOrError string) error {
	if success && s.notifyOnFailureOnly {
		return nil
	}

	msg := BuildMessage(success, loc, s.WebhookPrefix, filenameOrError)
	payload := map[string]interface{}{
		"text": msg.Text,
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": "*" + msg.Text + "*",
				},
			},
			{
				"type": "section",
				"fields": []map[string]string{
					{"type": "mrkdwn", "text": "*Status*\n" + msg.Status},
					{"type": "mrkdwn", "text": "*Time*\n" + msg.CurrentTime},
					{"type": "mrkdwn", "text": "*" + msg.FilenameOrErrorLabel + "*\n" + msg.FilenameOrError},
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Error encoding JSON: %v", err)
	}

	req, err := http.NewRequest("POST", s.WebhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("Error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: webhookTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected webhook response status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
