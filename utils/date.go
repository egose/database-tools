package utils

import (
	"fmt"
	"strings"
	"time"
)

func GetNewFilename() (string, string) {
	now := time.Now()
	timestamp := 9999999999999 - now.UnixNano()/int64(time.Millisecond)
	date := strings.ReplaceAll(now.Format("2006-01-02T15:04:05.000Z"), ":", "")
	name := fmt.Sprintf("%d-%s", timestamp, date)
	filename := fmt.Sprintf("%s.tar.gz", name)
	return filename, name
}
