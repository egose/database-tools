package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type smtpDialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

type SMTP struct {
	Host                            string
	Port                            string
	Username                        string
	Password                        string
	From                            string
	To                              []string
	SubjectPrefix                   string
	AllowInsecureNoTLSInDevelopment bool
	DialTimeout                     time.Duration
	ReadTimeout                     time.Duration
	WriteTimeout                    time.Duration
	Dialer                          smtpDialer
	TLSConfig                       *tls.Config
	notifyOnFailureOnly             bool
}

func (s *SMTP) Init(host, port, username, password, from, to, subjectPrefix string, notifyOnFailureOnly bool, allowInsecureNoTLSInDevelopment bool) error {
	recipients, err := ValidateSMTPOptions(host, port, username, password, from, to, subjectPrefix)
	if err != nil {
		return err
	}
	if port == "" {
		port = "587"
	}

	s.Host = host
	s.Port = port
	s.Username = username
	s.Password = password // pragma: allowlist secret
	s.From = from
	s.To = recipients
	s.SubjectPrefix = subjectPrefix
	s.AllowInsecureNoTLSInDevelopment = allowInsecureNoTLSInDevelopment
	s.notifyOnFailureOnly = notifyOnFailureOnly
	return nil
}

func ValidateSMTPOptions(host, port, username, password, from, to, subjectPrefix string) ([]string, error) {
	if host == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}
	if port == "" {
		port = "587"
	}
	if from == "" {
		return nil, fmt.Errorf("SMTP from address is required")
	}
	if _, err := mail.ParseAddress(from); err != nil {
		return nil, fmt.Errorf("invalid SMTP from address: %w", err)
	}
	if err := validateEmailHeaderValue("SMTP subject prefix", subjectPrefix); err != nil {
		return nil, err
	}

	recipients, err := parseRecipientList(to)
	if err != nil {
		return nil, err
	}
	if password != "" && username == "" {
		return nil, fmt.Errorf("SMTP username is required when password is set")
	}

	return recipients, nil
}

func (s *SMTP) Send(ctx context.Context, success bool, loc *time.Location, filenameOrError string) error {
	ctx = contextOrBackground(ctx)

	if success && s.notifyOnFailureOnly {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	msg := BuildMessage(success, loc, s.SubjectPrefix, filenameOrError)
	subject := msg.Text
	body := BuildPlainTextBody(msg)
	message, err := buildEmailMessage(s.From, s.To, subject, body)
	if err != nil {
		return err
	}

	var auth smtp.Auth
	if s.Username != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}

	conn, err := s.smtpDialer().DialContext(ctx, "tcp", net.JoinHostPort(s.Host, s.Port))
	if err != nil {
		return fmt.Errorf("failed to dial SMTP server: %w", err)
	}
	stopWatchingContext := closeConnOnContextDone(ctx, conn)
	defer stopWatchingContext()
	defer conn.Close()

	if err := s.setConnDeadlines(conn, ctx); err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if err := s.startTLSIfAvailable(ctx, conn, client); err != nil {
		return err
	}

	if auth != nil {
		if err := s.withConnDeadlines(conn, ctx, func() error { return client.Auth(auth) }); err != nil {
			return fmt.Errorf("failed to authenticate with SMTP server: %w", err)
		}
	}
	if err := s.withConnDeadlines(conn, ctx, func() error { return client.Mail(s.From) }); err != nil {
		return fmt.Errorf("failed to set SMTP sender: %w", err)
	}
	for _, recipient := range s.To {
		if err := s.withConnDeadlines(conn, ctx, func() error { return client.Rcpt(recipient) }); err != nil {
			return fmt.Errorf("failed to set SMTP recipient %q: %w", recipient, err)
		}
	}

	var writer io.WriteCloser
	if err := s.withConnDeadlines(conn, ctx, func() error {
		var dataErr error
		writer, dataErr = client.Data()
		return dataErr
	}); err != nil {
		return fmt.Errorf("failed to open SMTP message body: %w", err)
	}

	if err := s.withConnDeadlines(conn, ctx, func() error {
		if _, writeErr := io.WriteString(writer, message); writeErr != nil {
			return writeErr
		}
		return writer.Close()
	}); err != nil {
		return fmt.Errorf("failed to send SMTP email: %w", err)
	}

	if err := s.withConnDeadlines(conn, ctx, client.Quit); err != nil {
		return fmt.Errorf("failed to finish SMTP session: %w", err)
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

func buildEmailMessage(from string, to []string, subject string, body string) (string, error) {
	if err := validateEmailHeaderValue("SMTP from address", from); err != nil {
		return "", err
	}
	for _, recipient := range to {
		if err := validateEmailHeaderValue("SMTP recipient address", recipient); err != nil {
			return "", err
		}
	}
	encodedSubject, err := formatEmailSubject(subject)
	if err != nil {
		return "", err
	}

	headers := []string{
		"From: " + from,
		"To: " + strings.Join(to, ", "),
		"Subject: " + encodedSubject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}

	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body, nil
}

func validateEmailHeaderValue(fieldName, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%s must not contain CR or LF characters", fieldName)
	}

	return nil
}

func formatEmailSubject(subject string) (string, error) {
	if err := validateEmailHeaderValue("SMTP subject", subject); err != nil {
		return "", err
	}
	if isASCII(subject) {
		return subject, nil
	}

	return mime.QEncoding.Encode("utf-8", subject), nil
}

func isASCII(value string) bool {
	for _, r := range value {
		if r > 127 {
			return false
		}
	}

	return true
}

func (s *SMTP) smtpDialer() smtpDialer {
	if s.Dialer != nil {
		return s.Dialer
	}

	return &net.Dialer{Timeout: s.dialTimeout()}
}

func (s *SMTP) dialTimeout() time.Duration {
	if s.DialTimeout > 0 {
		return s.DialTimeout
	}

	return defaultSMTPDialTimeout
}

func (s *SMTP) readTimeout() time.Duration {
	if s.ReadTimeout > 0 {
		return s.ReadTimeout
	}

	return defaultSMTPReadTimeout
}

func (s *SMTP) writeTimeout() time.Duration {
	if s.WriteTimeout > 0 {
		return s.WriteTimeout
	}

	return defaultSMTPWriteTimeout
}

func (s *SMTP) setConnDeadlines(conn net.Conn, ctx context.Context) error {
	readDeadline := time.Now().Add(s.readTimeout())
	writeDeadline := time.Now().Add(s.writeTimeout())
	if ctxDeadline, ok := ctx.Deadline(); ok {
		if ctxDeadline.Before(readDeadline) {
			readDeadline = ctxDeadline
		}
		if ctxDeadline.Before(writeDeadline) {
			writeDeadline = ctxDeadline
		}
	}
	if err := conn.SetReadDeadline(readDeadline); err != nil {
		return fmt.Errorf("failed to set SMTP read deadline: %w", err)
	}
	if err := conn.SetWriteDeadline(writeDeadline); err != nil {
		return fmt.Errorf("failed to set SMTP write deadline: %w", err)
	}

	return nil
}

func (s *SMTP) withConnDeadlines(conn net.Conn, ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.setConnDeadlines(conn, ctx); err != nil {
		return err
	}
	if err := fn(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}

	return nil
}

func (s *SMTP) startTLSIfAvailable(ctx context.Context, conn net.Conn, client *smtp.Client) error {
	var hasStartTLS bool
	if err := s.withConnDeadlines(conn, ctx, func() error {
		hasStartTLS, _ = client.Extension("STARTTLS")
		return nil
	}); err != nil {
		return fmt.Errorf("failed to negotiate SMTP capabilities: %w", err)
	}
	if !hasStartTLS {
		if s.AllowInsecureNoTLSInDevelopment {
			return nil
		}
		return fmt.Errorf("SMTP server does not support STARTTLS; enable the development-only insecure no-TLS override only for local development")
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.Host}
	if s.TLSConfig != nil {
		tlsConfig = s.TLSConfig.Clone()
		if tlsConfig.MinVersion == 0 {
			tlsConfig.MinVersion = tls.VersionTLS12
		}
		if tlsConfig.ServerName == "" {
			tlsConfig.ServerName = s.Host
		}
	}

	if err := s.withConnDeadlines(conn, ctx, func() error { return client.StartTLS(tlsConfig) }); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("failed to start TLS for SMTP connection: %w", err)
	}

	return nil
}

func closeConnOnContextDone(ctx context.Context, conn net.Conn) func() {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	return func() {
		close(done)
	}
}
