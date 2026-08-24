package postgresarchiveformat

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"
)

const (
	FormatVersion             = 1
	DatabaseFamily            = "postgresql"
	DumpFormat                = "custom"
	ManifestPath              = "manifest.json"
	DumpPath                  = "database.dump"
	PostgreSQLBackupPrefix    = "postgres-archive/"
	maxManifestBytes          = 64 << 10
	postgresCustomFormatMagic = "PGDMP"
)

var ErrInvalidPayload = errors.New("invalid PostgreSQL archive payload")

type Manifest struct {
	FormatVersion           int       `json:"format_version"`
	DatabaseFamily          string    `json:"database_family"`
	DumpFormat              string    `json:"dump_format"`
	CreatedAt               time.Time `json:"created_at"`
	SourceDatabase          string    `json:"source_database"`
	PostgreSQLClientVersion string    `json:"postgresql_client_version"`
}

func NewManifest(createdAt time.Time, sourceDatabase string, clientVersion string) (Manifest, error) {
	manifest := Manifest{
		FormatVersion:           FormatVersion,
		DatabaseFamily:          DatabaseFamily,
		DumpFormat:              DumpFormat,
		CreatedAt:               createdAt.UTC(),
		SourceDatabase:          sourceDatabase,
		PostgreSQLClientVersion: clientVersion,
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func WriteManifest(w io.Writer, manifest Manifest) error {
	if err := ValidateManifest(manifest); err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("encode PostgreSQL archive manifest: %w", err)
	}
	return nil
}

func ValidateManifest(manifest Manifest) error {
	if manifest.FormatVersion != FormatVersion {
		return invalid("unsupported format version %d", manifest.FormatVersion)
	}
	if manifest.DatabaseFamily != DatabaseFamily {
		return invalid("database family must be %q", DatabaseFamily)
	}
	if manifest.DumpFormat != DumpFormat {
		return invalid("dump format must be %q", DumpFormat)
	}
	if manifest.CreatedAt.IsZero() {
		return invalid("creation time is required")
	}
	_, offset := manifest.CreatedAt.Zone()
	if offset != 0 {
		return invalid("creation time must use UTC")
	}
	if err := validateTextField("source database", manifest.SourceDatabase); err != nil {
		return err
	}
	if looksLikeConnectionString(manifest.SourceDatabase) {
		return invalid("source database must be a database name, not a connection string")
	}
	if err := validateTextField("PostgreSQL client version", manifest.PostgreSQLClientVersion); err != nil {
		return err
	}
	if looksLikeConnectionString(manifest.PostgreSQLClientVersion) {
		return invalid("PostgreSQL client version must not contain a connection string")
	}
	return nil
}

// ValidateArchive verifies the complete inner payload contract without extracting it.
func ValidateArchive(r io.Reader) (Manifest, error) {
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return Manifest{}, invalid("open gzip stream: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	seen := make(map[string]struct{}, 2)
	var manifest Manifest
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Manifest{}, invalid("read tar stream: %v", err)
		}

		entryName, err := validateEntryName(header.Name)
		if err != nil {
			return Manifest{}, err
		}
		if _, ok := seen[entryName]; ok {
			return Manifest{}, invalid("duplicate payload entry %q", entryName)
		}
		seen[entryName] = struct{}{}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return Manifest{}, invalid("payload entry %q must be a regular file", entryName)
		}

		switch entryName {
		case ManifestPath:
			manifest, err = readManifest(tarReader, header.Size)
			if err != nil {
				return Manifest{}, err
			}
		case DumpPath:
			if err := validateDump(tarReader, header.Size); err != nil {
				return Manifest{}, err
			}
		}
	}

	for _, required := range []string{ManifestPath, DumpPath} {
		if _, ok := seen[required]; !ok {
			return Manifest{}, invalid("missing payload entry %q", required)
		}
	}
	return manifest, nil
}

func validateEntryName(name string) (string, error) {
	cleanName := path.Clean(name)
	if name != cleanName {
		return "", invalid("payload entry path %q conflicts with canonical path %q", name, cleanName)
	}
	if cleanName != ManifestPath && cleanName != DumpPath {
		for _, required := range []string{ManifestPath, DumpPath} {
			if strings.HasPrefix(cleanName, required+"/") || strings.HasPrefix(required, cleanName+"/") {
				return "", invalid("payload entry path %q conflicts with required path %q", name, required)
			}
		}
		return "", invalid("unexpected payload entry %q", name)
	}
	return cleanName, nil
}

func readManifest(r io.Reader, size int64) (Manifest, error) {
	if size < 0 || size > maxManifestBytes {
		return Manifest{}, invalid("manifest size %d exceeds limit %d", size, maxManifestBytes)
	}

	decoder := json.NewDecoder(io.LimitReader(r, size))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, invalid("decode manifest: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return Manifest{}, invalid("manifest must contain exactly one JSON object")
		}
		return Manifest{}, invalid("decode manifest: %v", err)
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func validateDump(r io.Reader, size int64) error {
	if size < int64(len(postgresCustomFormatMagic)) {
		return invalid("dump is not PostgreSQL custom format")
	}
	magic := make([]byte, len(postgresCustomFormatMagic))
	if _, err := io.ReadFull(r, magic); err != nil {
		return invalid("read dump header: %v", err)
	}
	if string(magic) != postgresCustomFormatMagic {
		return invalid("dump is not PostgreSQL custom format")
	}
	return nil
}

func validateTextField(name string, value string) error {
	if strings.TrimSpace(value) == "" {
		return invalid("%s is required", name)
	}
	if len(value) > 1024 {
		return invalid("%s is too long", name)
	}
	if strings.ContainsAny(value, "\x00\r\n") {
		return invalid("%s contains control characters", name)
	}
	return nil
}

func looksLikeConnectionString(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "postgres://") ||
		strings.HasPrefix(lower, "postgresql://") ||
		strings.Contains(lower, "password=")
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidPayload, fmt.Sprintf(format, args...))
}
