package common

type Storage interface {
	Upload(string, []byte) (string, error)
	Download(string, string) error
	GetTargetObjectName(string) (string, error)
}
