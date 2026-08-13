package notification

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Message struct {
	Text                 string
	Status               string
	Color                string
	FilenameOrErrorLabel string
	FilenameOrError      string
	CurrentTime          string
}

func BuildMessage(success bool, loc *time.Location, prefix string, filenameOrError string) Message {
	timestamp := time.Now()
	if loc != nil {
		timestamp = timestamp.In(loc)
	}

	msg := Message{
		FilenameOrError: redactSensitiveText(filenameOrError),
		CurrentTime:     timestamp.Format("2006-01-02 15:04:05"),
	}

	if success {
		msg.Text = joinPrefix(prefix, "Database archiving completed successfully")
		msg.Status = "Success"
		msg.Color = "#00AA00"
		msg.FilenameOrErrorLabel = "Filename"
		return msg
	}

	msg.Text = joinPrefix(prefix, "Database archiving failed")
	msg.Status = "Failure"
	msg.Color = "#FF0000"
	msg.FilenameOrErrorLabel = "Error"
	return msg
}

func BuildPlainTextBody(msg Message) string {
	return fmt.Sprintf("%s\n\nStatus: %s\nTime: %s\n%s: %s\n", msg.Text, msg.Status, msg.CurrentTime, msg.FilenameOrErrorLabel, msg.FilenameOrError)
}

func joinPrefix(prefix string, text string) string {
	if strings.TrimSpace(prefix) == "" {
		return text
	}

	return strings.TrimSpace(prefix) + " " + text
}

var uriPattern = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://[^\s]+`)

func redactSensitiveText(raw string) string {
	return uriPattern.ReplaceAllStringFunc(raw, redactSensitiveURI)
}

func redactSensitiveURI(raw string) string {
	uri, suffix := trimTrailingURIJunk(raw)
	sanitized := pruneURIUserInfo(uri)

	parsed, err := url.Parse(sanitized)
	if err == nil && parsed.RawQuery != "" {
		query := parsed.Query()
		changed := false
		for key := range query {
			if isSensitiveQueryKey(key) {
				query.Set(key, "REDACTED")
				changed = true
			}
		}
		if changed {
			parsed.RawQuery = query.Encode()
			sanitized = parsed.String()
		}
	}

	return sanitized + suffix
}

func trimTrailingURIJunk(raw string) (string, string) {
	trimmed := strings.TrimRight(raw, ".,;:!?)")
	return trimmed, raw[len(trimmed):]
}

func pruneURIUserInfo(raw string) string {
	schemeIndex := strings.Index(raw, "://")
	if schemeIndex == -1 {
		return raw
	}

	authorityStart := schemeIndex + len("://")
	remainder := raw[authorityStart:]
	authorityEnd := strings.IndexAny(remainder, "/?#")
	if authorityEnd == -1 {
		authorityEnd = len(remainder)
	}

	authority := remainder[:authorityEnd]
	atIndex := strings.LastIndex(authority, "@")
	if atIndex == -1 {
		return raw
	}

	return raw[:authorityStart] + authority[atIndex+1:] + remainder[authorityEnd:]
}

func isSensitiveQueryKey(key string) bool {
	switch strings.ToLower(key) {
	case "access_token", "accountkey", "awsaccesskeyid", "password", "passwd", "pwd", "sig", "signature", "token", "x-amz-security-token", "x-amz-signature", "x-goog-credential", "x-goog-security-token", "x-goog-signature":
		return true
	default:
		return false
	}
}
