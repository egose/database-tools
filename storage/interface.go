package storage

import "context"

type Storage interface {
	Upload(context.Context, string, string) (string, error)
	Download(context.Context, string, string) error
	GetTargetObjectName(context.Context, string) (string, error)
	DeleteOldObjects(context.Context, string) error
	Close() error
}
