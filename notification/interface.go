package notification

import (
	"context"
	"time"
)

type Notification interface {
	Send(context.Context, bool, *time.Location, string) error
}
