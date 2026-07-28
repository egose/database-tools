package notification

import (
	"fmt"
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
		FilenameOrError: filenameOrError,
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
