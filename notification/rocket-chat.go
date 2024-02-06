package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

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
	var text string
	var color string
	var status string
	var filenameOrErrorLabel string

	if success {
		if this.notifyOnFailureOnly {
			return nil
		}

		msg := "Database archiving completed successfully"
		if this.WebhookPrefix != "" {
			text = fmt.Sprintf("%s %s", this.WebhookPrefix, msg)
		} else {
			text = msg
		}
		color = "#00AA00"
		status = "Success"
		filenameOrErrorLabel = "Filename"
	} else {
		msg := "Database archiving failed"
		if this.WebhookPrefix != "" {
			text = fmt.Sprintf("%s %s", this.WebhookPrefix, msg)
		} else {
			text = msg
		}
		color = "#FF0000"
		status = "Failure"
		filenameOrErrorLabel = "Error"
	}

	currentTime := time.Now().In(loc).Format("2006-01-02 15:04:05")
	attachments := []map[string]interface{}{
		{
			"title": "Details",
			"text":  "",
			"color": color,
			"fields": []map[string]interface{}{
				{
					"title": "Status",
					"value": status,
					"short": false,
				},
				{
					"title": "Time",
					"value": currentTime,
					"short": false,
				},
				{
					"title": filenameOrErrorLabel,
					"value": filenameOrError,
					"short": false,
				},
			},
		},
	}

	payload := map[string]interface{}{
		"text":        text,
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	return nil
}
