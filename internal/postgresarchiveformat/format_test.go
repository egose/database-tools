package postgresarchiveformat

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

var testCreatedAt = time.Date(2026, time.August, 24, 12, 34, 56, 789000000, time.UTC)

func TestValidateArchiveAcceptsVersionedPostgreSQLPayload(t *testing.T) {
	if PostgreSQLBackupPrefix != "postgres-archive/" {
		t.Fatalf("PostgreSQLBackupPrefix = %q, want %q", PostgreSQLBackupPrefix, "postgres-archive/")
	}
	manifest := validManifest(t)
	archive := buildArchive(t,
		testEntry{name: DumpPath, body: []byte("PGDMPdump-data")},
		testEntry{name: ManifestPath, body: manifestJSON(t, manifest)},
	)

	got, err := ValidateArchive(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("ValidateArchive() error = %v", err)
	}
	if got != manifest {
		t.Fatalf("ValidateArchive() manifest = %#v, want %#v", got, manifest)
	}
}

func TestManifestJSONHasOnlyApprovedFields(t *testing.T) {
	manifest := validManifest(t)
	var encoded bytes.Buffer
	if err := WriteManifest(&encoded, manifest); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded.Bytes(), &fields); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	want := []string{"created_at", "database_family", "dump_format", "format_version", "postgresql_client_version", "source_database"}
	if len(fields) != len(want) {
		t.Fatalf("manifest fields = %v, want exactly %v", fields, want)
	}
	for _, name := range want {
		if _, ok := fields[name]; !ok {
			t.Fatalf("manifest missing field %q", name)
		}
	}
	for _, forbidden := range []string{"password", "uri", "connection_string"} {
		if _, ok := fields[forbidden]; ok {
			t.Fatalf("manifest unexpectedly contains credential field %q", forbidden)
		}
	}
}

func TestValidateArchiveRejectsInvalidPayloads(t *testing.T) {
	valid := validManifest(t)
	tests := []struct {
		name    string
		entries []testEntry
		want    string
	}{
		{name: "missing manifest", entries: []testEntry{{name: DumpPath, body: []byte("PGDMPdump")}}, want: `missing payload entry "manifest.json"`},
		{name: "missing dump", entries: []testEntry{{name: ManifestPath, body: manifestJSON(t, valid)}}, want: `missing payload entry "database.dump"`},
		{name: "malformed manifest", entries: validEntries([]byte(`{"format_version":`)), want: "decode manifest"},
		{name: "unsupported version", entries: validEntries(rawManifestJSON(t, withManifest(valid, func(m *Manifest) { m.FormatVersion++ }))), want: "unsupported format version"},
		{name: "wrong database family", entries: validEntries(rawManifestJSON(t, withManifest(valid, func(m *Manifest) { m.DatabaseFamily = "mongodb" }))), want: "database family"},
		{name: "wrong dump format", entries: validEntries(rawManifestJSON(t, withManifest(valid, func(m *Manifest) { m.DumpFormat = "plain" }))), want: "dump format"},
		{name: "malformed creation time", entries: validEntries(replaceJSONField(t, valid, "created_at", "not-a-time")), want: "cannot parse"},
		{name: "duplicate manifest", entries: []testEntry{{name: ManifestPath, body: manifestJSON(t, valid)}, {name: ManifestPath, body: manifestJSON(t, valid)}, {name: DumpPath, body: []byte("PGDMPdump")}}, want: "duplicate payload entry"},
		{name: "non-canonical path conflict", entries: []testEntry{{name: "./" + ManifestPath, body: manifestJSON(t, valid)}, {name: DumpPath, body: []byte("PGDMPdump")}}, want: "conflicts with canonical path"},
		{name: "child path conflict", entries: []testEntry{{name: ManifestPath + "/secret", body: []byte("secret")}, {name: DumpPath, body: []byte("PGDMPdump")}}, want: "conflicts with required path"},
		{name: "unexpected MongoDB entry", entries: []testEntry{{name: "database/collection.bson", body: []byte("mongo")}}, want: "unexpected payload entry"},
		{name: "arbitrary tarball", entries: []testEntry{{name: "readme.txt", body: []byte("not a backup")}}, want: "unexpected payload entry"},
		{name: "invalid custom dump", entries: validEntries(manifestJSON(t, valid), testEntry{name: DumpPath, body: []byte("plain SQL")}), want: "not PostgreSQL custom format"},
		{name: "unknown credential field", entries: validEntries(addJSONField(t, valid, "password", "secret")), want: "unknown field"},
		{name: "password URI as database", entries: validEntries(rawManifestJSON(t, withManifest(valid, func(m *Manifest) { m.SourceDatabase = "postgresql://user:not-a-real-password@db/app" }))), want: "not a connection string"},
		{name: "password DSN as database", entries: validEntries(rawManifestJSON(t, withManifest(valid, func(m *Manifest) { m.SourceDatabase = "dbname=app password=not-a-real-password" }))), want: "not a connection string"},
		{name: "connection string as client version", entries: validEntries(rawManifestJSON(t, withManifest(valid, func(m *Manifest) { m.PostgreSQLClientVersion = "password=not-a-real-password" }))), want: "must not contain a connection string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateArchive(bytes.NewReader(buildArchive(t, tt.entries...)))
			if !errors.Is(err, ErrInvalidPayload) {
				t.Fatalf("ValidateArchive() error = %v, want ErrInvalidPayload", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateArchive() error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateArchiveRejectsMalformedTransport(t *testing.T) {
	_, err := ValidateArchive(strings.NewReader("not gzip"))
	if !errors.Is(err, ErrInvalidPayload) || !strings.Contains(err.Error(), "open gzip stream") {
		t.Fatalf("ValidateArchive() error = %v, want invalid gzip payload", err)
	}
}

func validManifest(t *testing.T) Manifest {
	t.Helper()
	manifest, err := NewManifest(testCreatedAt, "inventory", "pg_dump (PostgreSQL) 18.1")
	if err != nil {
		t.Fatalf("NewManifest() error = %v", err)
	}
	return manifest
}

type testEntry struct {
	name string
	body []byte
}

func validEntries(manifest []byte, replacements ...testEntry) []testEntry {
	entries := []testEntry{{name: ManifestPath, body: manifest}, {name: DumpPath, body: []byte("PGDMPdump")}}
	for _, replacement := range replacements {
		for i := range entries {
			if entries[i].name == replacement.name {
				entries[i] = replacement
			}
		}
	}
	return entries
}

func buildArchive(t *testing.T, entries ...testEntry) []byte {
	t.Helper()
	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := &tar.Header{Name: entry.name, Mode: 0o600, Size: int64(len(entry.body)), Typeflag: tar.TypeReg}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader() error = %v", err)
		}
		if _, err := tarWriter.Write(entry.body); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tar Close() error = %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzip Close() error = %v", err)
	}
	return archive.Bytes()
}

func manifestJSON(t *testing.T, manifest Manifest) []byte {
	t.Helper()
	var output bytes.Buffer
	if err := WriteManifest(&output, manifest); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}
	return output.Bytes()
}

func rawManifestJSON(t *testing.T, manifest Manifest) []byte {
	t.Helper()
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return encoded
}

func withManifest(manifest Manifest, change func(*Manifest)) Manifest {
	change(&manifest)
	return manifest
}

func addJSONField(t *testing.T, manifest Manifest, name string, value any) []byte {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(manifestJSON(t, manifest), &fields); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	fields[name] = value
	encoded, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return encoded
}

func replaceJSONField(t *testing.T, manifest Manifest, name string, value any) []byte {
	return addJSONField(t, manifest, name, value)
}
