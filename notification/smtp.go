package notification

import (
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type SMTP struct {
	Host                string
	Port                string
	Username            string
	Password            string
	From                string
	To                  []string
	SubjectPrefix       string
	notifyOnFailureOnly bool
}

func (s *SMTP) Init(host, port, username, password, from, to, subjectPrefix string, notifyOnFailureOnly bool) error {
	if host == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if port == "" {
		port = "587"
	}
	if from == "" {
		return fmt.Errorf("SMTP from address is required")
	}
	if _, err := mail.ParseAddress(from); err != nil {
		return fmt.Errorf("invalid SMTP from address: %w", err)
	}

	recipients, err := parseRecipientList(to)
	if err != nil {
		return err
	}
	if password != "" && username == "" {
		return fmt.Errorf("SMTP username is required when password is set")
	}

	s.Host = host
	s.Port = port
	s.Username = username
	s.Password = password // pragma: allowlist secret
	s.From = from
	s.To = recipients
	s.SubjectPrefix = subjectPrefix
	s.notifyOnFailureOnly = notifyOnFailureOnly
	return nil
}

func (s *SMTP) Send(success bool, loc *time.Location, filenameOrError string) error {
	if success && s.notifyOnFailureOnly {
		return nil
	}

	msg := BuildMessage(success, loc, s.SubjectPrefix, filenameOrError)
	subject := msg.Text
	body := BuildPlainTextBody(msg)
	message := buildEmailMessage(s.From, s.To, subject, body)

	var auth smtp.Auth
	if s.Username != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}

	if err := smtp.SendMail(s.Host+":"+s.Port, auth, s.From, s.To, []byte(message)); err != nil {
		return fmt.Errorf("failed to send SMTP email: %w", err)
	}

	return nil
}

func parseRecipientList(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	recipients := make([]string, 0, len(parts))
	for _, part := range parts {
		addr := strings.TrimSpace(part)
		if addr == "" {
			continue
		}
		if _, err := mail.ParseAddress(addr); err != nil {
			return nil, fmt.Errorf("invalid SMTP recipient address %q: %w", addr, err)
		}
		recipients = append(recipients, addr)
	}

	if len(recipients) == 0 {
		return nil, fmt.Errorf("at least one SMTP recipient is required")
	}

	return recipients, nil
}

func buildEmailMessage(from string, to []string, subject string, body string) string {
	headers := []string{
		"From: " + from,
		"To: " + strings.Join(to, ", "),
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}

	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body
}
