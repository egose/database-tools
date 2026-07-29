package storage

type Storage interface {
	Upload(string, string) (string, error)
	Download(string, string) error
	GetTargetObjectName(string) (string, error)
	DeleteOldObjects() error
}
