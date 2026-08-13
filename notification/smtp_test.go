package notification

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

type smtpDialerFunc func(context.Context, string, string) (net.Conn, error)

func (f smtpDialerFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return f(ctx, network, address)
}

func TestSMTPInitParsesRecipients(t *testing.T) {
	smtpNotification := new(SMTP)
	err := smtpNotification.Init("smtp.example.com", "587", "user", "pass", "from@example.com", "a@example.com, b@example.com", "[backup]", false, false)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	if len(smtpNotification.To) != 2 {
		t.Fatalf("Init() recipients len = %d, want 2", len(smtpNotification.To))
	}
}

func TestSMTPInitRejectsMissingRecipients(t *testing.T) {
	smtpNotification := new(SMTP)
	err := smtpNotification.Init("smtp.example.com", "587", "user", "pass", "from@example.com", "", "", false, false)
	if err == nil {
		t.Fatal("Init() expected error for missing recipients")
	}
}

func TestSMTPInitRejectsSubjectHeaderInjection(t *testing.T) {
	smtpNotification := new(SMTP)
	err := smtpNotification.Init("smtp.example.com", "587", "", "", "from@example.com", "to@example.com", "ok\r\nBcc: victim@example.com", false, false)
	if err == nil {
		t.Fatal("Init() expected CRLF rejection")
	}
}

func TestBuildEmailMessageContainsHeaders(t *testing.T) {
	msg, err := buildEmailMessage("from@example.com", []string{"to@example.com"}, "subject", "body")
	if err != nil {
		t.Fatalf("buildEmailMessage() error = %v", err)
	}
	wants := []string{"From: from@example.com", "To: to@example.com", "Subject: subject", "body"}
	for _, want := range wants {
		if !strings.Contains(msg, want) {
			t.Fatalf("buildEmailMessage() = %q, missing %q", msg, want)
		}
	}
}

func TestBuildEmailMessageEncodesNonASCIISubject(t *testing.T) {
	msg, err := buildEmailMessage("from@example.com", []string{"to@example.com"}, "[backup] d\u00e9j\u00e0 vu", "body")
	if err != nil {
		t.Fatalf("buildEmailMessage() error = %v", err)
	}
	if !strings.Contains(msg, "Subject: =?utf-8?") {
		t.Fatalf("buildEmailMessage() = %q, want MIME-encoded subject", msg)
	}
}

func TestSMTPSendRejectsServerWithoutSTARTTLS(t *testing.T) {
	addr, shutdown := serveSMTP(t, func(conn net.Conn) {
		reader := bufio.NewReader(conn)
		mustWriteSMTPLine(t, conn, "220 smtp.test ready")
		expectSMTPCommand(t, reader, "EHLO")
		mustWriteSMTPMultiline(t, conn, []string{"250-smtp.test", "250 AUTH PLAIN"})
	})
	defer shutdown()

	smtpNotification := &SMTP{Host: hostFromAddr(addr), Port: portFromAddr(addr), From: "from@example.com", To: []string{"to@example.com"}}
	err := smtpNotification.Send(context.Background(), false, time.UTC, "boom")
	if err == nil || !strings.Contains(err.Error(), "does not support STARTTLS") {
		t.Fatalf("Send() error = %v, want STARTTLS rejection", err)
	}
}

func TestSMTPSendAllowsDevelopmentNoTLSOptOut(t *testing.T) {
	messageCh := make(chan string, 1)
	addr, shutdown := serveSMTP(t, func(conn net.Conn) {
		reader := bufio.NewReader(conn)
		mustWriteSMTPLine(t, conn, "220 smtp.test ready")
		expectSMTPCommand(t, reader, "EHLO")
		mustWriteSMTPMultiline(t, conn, []string{"250-smtp.test", "250 OK"})
		expectSMTPCommand(t, reader, "MAIL FROM:")
		mustWriteSMTPLine(t, conn, "250 sender ok")
		expectSMTPCommand(t, reader, "RCPT TO:")
		mustWriteSMTPLine(t, conn, "250 recipient ok")
		expectSMTPCommand(t, reader, "DATA")
		mustWriteSMTPLine(t, conn, "354 go ahead")
		messageCh <- readSMTPData(t, reader)
		mustWriteSMTPLine(t, conn, "250 queued")
		expectSMTPCommand(t, reader, "QUIT")
		mustWriteSMTPLine(t, conn, "221 bye")
	})
	defer shutdown()

	smtpNotification := &SMTP{
		Host:                            hostFromAddr(addr),
		Port:                            portFromAddr(addr),
		From:                            "from@example.com",
		To:                              []string{"to@example.com"},
		AllowInsecureNoTLSInDevelopment: true,
	}
	if err := smtpNotification.Send(context.Background(), false, time.UTC, "boom"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if msg := <-messageCh; !strings.Contains(msg, "Database archiving failed") {
		t.Fatalf("Send() message = %q, want body content", msg)
	}
}

func TestSMTPSendHonorsContextCancellation(t *testing.T) {
	smtpNotification := &SMTP{
		Host: "smtp.example.com",
		Port: "587",
		From: "from@example.com",
		To:   []string{"to@example.com"},
		Dialer: smtpDialerFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := smtpNotification.Send(ctx, false, time.UTC, "boom")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Send() error = %v, want context.Canceled", err)
	}
}

func TestSMTPSendTimesOutOnSlowServer(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	go func() {
		defer clientConn.Close()
		mustWriteSMTPLine(t, serverConn, "220 smtp.test ready")
		reader := bufio.NewReader(serverConn)
		_, _ = reader.ReadString('\n')
		time.Sleep(250 * time.Millisecond)
	}()

	smtpNotification := &SMTP{
		Host:                            "smtp.example.com",
		Port:                            "587",
		From:                            "from@example.com",
		To:                              []string{"to@example.com"},
		AllowInsecureNoTLSInDevelopment: true,
		ReadTimeout:                     50 * time.Millisecond,
		WriteTimeout:                    50 * time.Millisecond,
		Dialer: smtpDialerFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
			return clientConn, nil
		}),
	}

	err := smtpNotification.Send(context.Background(), false, time.UTC, "boom")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "timeout") {
		t.Fatalf("Send() error = %v, want timeout", err)
	}
}

func TestSMTPSendRejectsInvalidTLSCertificate(t *testing.T) {
	addr, shutdown := serveSMTP(t, func(conn net.Conn) {
		reader := bufio.NewReader(conn)
		mustWriteSMTPLine(t, conn, "220 smtp.test ready")
		expectSMTPCommand(t, reader, "EHLO")
		mustWriteSMTPMultiline(t, conn, []string{"250-smtp.test", "250 STARTTLS"})
		expectSMTPCommand(t, reader, "STARTTLS")
		mustWriteSMTPLine(t, conn, "220 ready to start tls")

		tlsConn := tls.Server(conn, testServerTLSConfig(t))
		defer tlsConn.Close()
		_ = tlsConn.Handshake()
	})
	defer shutdown()

	smtpNotification := &SMTP{Host: hostFromAddr(addr), Port: portFromAddr(addr), From: "from@example.com", To: []string{"to@example.com"}}
	err := smtpNotification.Send(context.Background(), false, time.UTC, "boom")
	if err == nil {
		t.Fatal("Send() expected TLS validation failure")
	}
	errText := strings.ToLower(err.Error())
	if !strings.Contains(errText, "tls") && !strings.Contains(errText, "x509") && !strings.Contains(errText, "certificate") {
		t.Fatalf("Send() error = %v, want TLS validation failure", err)
	}
}

func serveSMTP(t *testing.T, handle func(net.Conn)) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		handle(conn)
	}()

	return listener.Addr().String(), func() {
		_ = listener.Close()
		<-done
	}
}

func expectSMTPCommand(t *testing.T, reader *bufio.Reader, prefix string) string {
	t.Helper()
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, prefix) {
		t.Fatalf("SMTP command = %q, want prefix %q", line, prefix)
	}
	return line
}

func readSMTPData(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	var builder strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("ReadString() error = %v", err)
		}
		if line == ".\r\n" {
			return builder.String()
		}
		builder.WriteString(line)
	}
}

func mustWriteSMTPLine(t *testing.T, writer io.Writer, line string) {
	t.Helper()
	if _, err := fmt.Fprintf(writer, "%s\r\n", line); err != nil {
		t.Fatalf("Fprintf() error = %v", err)
	}
}

func mustWriteSMTPMultiline(t *testing.T, writer io.Writer, lines []string) {
	t.Helper()
	for _, line := range lines {
		mustWriteSMTPLine(t, writer, line)
	}
}

func hostFromAddr(addr string) string {
	host, _, _ := net.SplitHostPort(addr)
	return host
}

func portFromAddr(addr string) string {
	_, port, _ := net.SplitHostPort(addr)
	return port
}

func testServerTLSConfig(t *testing.T) *tls.Config {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		DNSNames:              []string{"localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}

	return &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{certificateDER}, PrivateKey: privateKey}}}
}
