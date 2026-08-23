package notification

import (
	"context"
	"fmt"
	"net/mail"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

type sesEmailSender interface {
	SendEmailWithContext(aws.Context, *ses.SendEmailInput, ...request.Option) (*ses.SendEmailOutput, error)
}

type SES struct {
	Endpoint                               string
	Region                                 string
	AccessKeyID                            string
	SecretAccessKey                        string
	From                                   string
	To                                     []string
	SubjectPrefix                          string
	AllowInsecureEndpointHTTPInDevelopment bool
	notifyOnFailureOnly                    bool
	client                                 sesEmailSender
}

func (s *SES) Init(endpoint, region, accessKeyID, secretAccessKey, from, to, subjectPrefix string, notifyOnFailureOnly bool, allowInsecureEndpointHTTPInDevelopment bool) error {
	recipients, err := ValidateSESOptions(endpoint, region, accessKeyID, secretAccessKey, from, to, allowInsecureEndpointHTTPInDevelopment)
	if err != nil {
		return err
	}

	config := &aws.Config{Region: aws.String(region)}
	if endpoint != "" {
		config.Endpoint = aws.String(endpoint)
	}
	if accessKeyID != "" {
		config.Credentials = credentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return fmt.Errorf("failed to create SES session: %w", err)
	}

	s.Endpoint = endpoint
	s.Region = region
	s.AccessKeyID = accessKeyID
	s.SecretAccessKey = secretAccessKey // pragma: allowlist secret
	s.From = from
	s.To = recipients
	s.SubjectPrefix = subjectPrefix
	s.AllowInsecureEndpointHTTPInDevelopment = allowInsecureEndpointHTTPInDevelopment
	s.notifyOnFailureOnly = notifyOnFailureOnly
	s.client = ses.New(sess)
	return nil
}

func ValidateSESOptions(endpoint, region, accessKeyID, secretAccessKey, from, to string, allowInsecureEndpointHTTPInDevelopment bool) ([]string, error) {
	if region == "" {
		return nil, fmt.Errorf("SES region is required")
	}
	if err := validateHTTPSURL(endpoint, allowInsecureEndpointHTTPInDevelopment, "SES endpoint override"); err != nil {
		return nil, err
	}
	if from == "" {
		return nil, fmt.Errorf("SES from address is required")
	}
	if _, err := mail.ParseAddress(from); err != nil {
		return nil, fmt.Errorf("invalid SES from address: %w", err)
	}
	if (accessKeyID == "") != (secretAccessKey == "") {
		return nil, fmt.Errorf("SES access key ID and secret access key must be set together")
	}

	recipients, err := parseRecipientList(to)
	if err != nil {
		return nil, err
	}

	return recipients, nil
}

func (s *SES) Send(ctx context.Context, success bool, loc *time.Location, filenameOrError string) error {
	if success && s.notifyOnFailureOnly {
		return nil
	}
	if s.client == nil {
		return fmt.Errorf("SES client is not initialized")
	}

	msg := BuildMessage(success, loc, s.SubjectPrefix, filenameOrError)
	body := BuildPlainTextBody(msg)
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{ToAddresses: aws.StringSlice(s.To)},
		Message: &ses.Message{
			Subject: &ses.Content{Charset: aws.String("UTF-8"), Data: aws.String(msg.Text)},
			Body: &ses.Body{
				Text: &ses.Content{Charset: aws.String("UTF-8"), Data: aws.String(body)},
			},
		},
		Source: aws.String(s.From),
	}

	if _, err := s.client.SendEmailWithContext(aws.Context(contextOrBackground(ctx)), input); err != nil {
		return fmt.Errorf("failed to send SES email: %w", err)
	}

	return nil
}
