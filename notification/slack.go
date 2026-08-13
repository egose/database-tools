package notification

import (
	"context"
	"time"
)

type Slack struct {
	WebhookURL                            string
	WebhookPrefix                         string
	AllowInsecureWebhookHTTPInDevelopment bool
	Client                                httpDoer
	notifyOnFailureOnly                   bool
}

func (s *Slack) Init(webhookURL, webhookPrefix string, notifyOnFailureOnly bool, allowInsecureWebhookHTTPInDevelopment bool) error {
	if err := validateHTTPSURL(webhookURL, allowInsecureWebhookHTTPInDevelopment, "Slack webhook URL"); err != nil {
		return err
	}

	s.WebhookURL = webhookURL
	s.WebhookPrefix = webhookPrefix
	s.AllowInsecureWebhookHTTPInDevelopment = allowInsecureWebhookHTTPInDevelopment
	s.Client = httpClientOrDefault(s.Client)
	s.notifyOnFailureOnly = notifyOnFailureOnly
	return nil
}

func (s *Slack) Send(ctx context.Context, success bool, loc *time.Location, filenameOrError string) error {
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

	return sendWebhookJSON(ctx, s.Client, s.WebhookURL, payload)
}
