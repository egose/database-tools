package storage

import "context"

type ArchiveStorage interface {
	Upload(context.Context, string, string) (string, error)
	DeleteOldObjects(context.Context, string) error
}

type RestoreStorage interface {
	Download(context.Context, string, string) error
	GetTargetObjectName(context.Context, string) (string, error)
}

type Lifecycle interface {
	Close() error
}

type BackendIdentifier interface {
	BackendName() string
}

type ArchiveBackend interface {
	ArchiveStorage
	Lifecycle
}

type RestoreBackend interface {
	RestoreStorage
	Lifecycle
	BackendIdentifier
}
