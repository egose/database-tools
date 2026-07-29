package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const webhookTimeout = 10 * time.Second

type RocketChat struct {
	WebhookUrl          string
	WebhookPrefix       string
	notifyOnFailureOnly bool
}

func (this *RocketChat) Init(webhookUrl, webhookPrefix string, notifyOnFailureOnly bool) error {
	this.WebhookUrl = webhookUrl
	this.WebhookPrefix = webhookPrefix
	this.notifyOnFailureOnly = notifyOnFailureOnly
	return nil
}

func (this *RocketChat) Send(success bool, loc *time.Location, filenameOrError string) error {
	if success && this.notifyOnFailureOnly {
		return nil
	}

	msg := BuildMessage(success, loc, this.WebhookPrefix, filenameOrError)
	attachments := []map[string]interface{}{
		{
			"title": "Details",
			"text":  "",
			"color": msg.Color,
			"fields": []map[string]interface{}{
				{
					"title": "Status",
					"value": msg.Status,
					"short": false,
				},
				{
					"title": "Time",
					"value": msg.CurrentTime,
					"short": false,
				},
				{
					"title": msg.FilenameOrErrorLabel,
					"value": msg.FilenameOrError,
					"short": false,
				},
			},
		},
	}

	payload := map[string]interface{}{
		"text":        msg.Text,
		"attachments": attachments,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Error encoding JSON: %v", err)
	}

	req, err := http.NewRequest("POST", this.WebhookUrl, bytes.NewBuffer(jsonPayload))
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
