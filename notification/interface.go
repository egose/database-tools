package notification

import (
	"time"
)

type Notification interface {
	Send(bool, *time.Location, string) error
}
