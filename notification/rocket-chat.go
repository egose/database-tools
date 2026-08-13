package notification

import (
	"context"
	"time"
)

type RocketChat struct {
	WebhookUrl                            string
	WebhookPrefix                         string
	AllowInsecureWebhookHTTPInDevelopment bool
	Client                                httpDoer
	notifyOnFailureOnly                   bool
}

func (this *RocketChat) Init(webhookUrl, webhookPrefix string, notifyOnFailureOnly bool, allowInsecureWebhookHTTPInDevelopment bool) error {
	if err := validateHTTPSURL(webhookUrl, allowInsecureWebhookHTTPInDevelopment, "Rocket.Chat webhook URL"); err != nil {
		return err
	}

	this.WebhookUrl = webhookUrl
	this.WebhookPrefix = webhookPrefix
	this.AllowInsecureWebhookHTTPInDevelopment = allowInsecureWebhookHTTPInDevelopment
	this.Client = httpClientOrDefault(this.Client)
	this.notifyOnFailureOnly = notifyOnFailureOnly
	return nil
}

func (this *RocketChat) Send(ctx context.Context, success bool, loc *time.Location, filenameOrError string) error {
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

	return sendWebhookJSON(ctx, this.Client, this.WebhookUrl, payload)
}
