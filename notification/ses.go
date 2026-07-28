package notification

import (
	"fmt"
	"net/mail"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

type sesEmailSender interface {
	SendEmail(input *ses.SendEmailInput) (*ses.SendEmailOutput, error)
}

type SES struct {
	Endpoint            string
	Region              string
	AccessKeyID         string
	SecretAccessKey     string
	From                string
	To                  []string
	SubjectPrefix       string
	notifyOnFailureOnly bool
	client              sesEmailSender
}

func (s *SES) Init(endpoint, region, accessKeyID, secretAccessKey, from, to, subjectPrefix string, notifyOnFailureOnly bool) error {
	if region == "" {
		return fmt.Errorf("SES region is required")
	}
	if from == "" {
		return fmt.Errorf("SES from address is required")
	}
	if _, err := mail.ParseAddress(from); err != nil {
		return fmt.Errorf("invalid SES from address: %w", err)
	}
	if (accessKeyID == "") != (secretAccessKey == "") {
		return fmt.Errorf("SES access key ID and secret access key must be set together")
	}

	recipients, err := parseRecipientList(to)
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
	s.notifyOnFailureOnly = notifyOnFailureOnly
	s.client = ses.New(sess)
	return nil
}

func (s *SES) Send(success bool, loc *time.Location, filenameOrError string) error {
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

	if _, err := s.client.SendEmail(input); err != nil {
		return fmt.Errorf("failed to send SES email: %w", err)
	}

	return nil
}
