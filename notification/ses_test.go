package notification

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/ses"
)

type fakeSESSender struct {
	input *ses.SendEmailInput
	err   error
}

func (f *fakeSESSender) SendEmail(input *ses.SendEmailInput) (*ses.SendEmailOutput, error) {
	f.input = input
	if f.err != nil {
		return nil, f.err
	}
	return &ses.SendEmailOutput{}, nil
}

func TestSESInitRejectsInvalidFrom(t *testing.T) {
	sesNotification := new(SES)
	err := sesNotification.Init("", "us-east-1", "", "", "bad", "to@example.com", "", false)
	if err == nil {
		t.Fatal("Init() expected invalid from error")
	}
}

func TestSESSendBuildsEmail(t *testing.T) {
	fake := &fakeSESSender{}
	sesNotification := &SES{
		From:          "from@example.com",
		To:            []string{"to@example.com"},
		SubjectPrefix: "[backup]",
		client:        fake,
	}

	if err := sesNotification.Send(false, time.UTC, "boom"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if fake.input == nil || fake.input.Message == nil || fake.input.Message.Subject == nil || fake.input.Message.Subject.Data == nil {
		t.Fatal("Send() did not populate SES input")
	}
	if got := *fake.input.Message.Subject.Data; got == "" {
		t.Fatal("Send() empty SES subject")
	}
}

func TestSESSendReturnsServiceError(t *testing.T) {
	sesNotification := &SES{
		From:   "from@example.com",
		To:     []string{"to@example.com"},
		client: &fakeSESSender{err: errors.New("send failed")},
	}

	if err := sesNotification.Send(false, time.UTC, "boom"); err == nil {
		t.Fatal("Send() expected service error")
	}
}
