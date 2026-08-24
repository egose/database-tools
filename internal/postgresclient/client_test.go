package postgresclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type helperConfig struct {
	Behavior   string
	Stdout     string
	Stderr     string
	Observe    string
	CreateDump bool
}

type helperObservation struct {
	Name          string
	Args          []string
	Environment   []string
	PGPassPath    string
	PGPassMode    uint32
	PGPassContent string
}

type fakeProcessFactory struct {
	mu        sync.Mutex
	config    string
	lookupErr error
	lookups   []string
	name      string
	args      []string
}

func (factory *fakeProcessFactory) LookPath(file string) (string, error) {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	factory.lookups = append(factory.lookups, file)
	if factory.lookupErr != nil {
		return "", factory.lookupErr
	}
	return "/fake/bin/" + file, nil
}

func (factory *fakeProcessFactory) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	factory.mu.Lock()
	factory.name = name
	factory.args = append([]string(nil), args...)
	factory.mu.Unlock()
	helperArgs := []string{"-test.run=TestPostgresClientHelperProcess", "--", factory.config, name}
	helperArgs = append(helperArgs, args...)
	return exec.CommandContext(ctx, os.Args[0], helperArgs...)
}

func TestPostgresClientHelperProcess(t *testing.T) {
	separator := -1
	for index, arg := range os.Args {
		if arg == "--" {
			separator = index
			break
		}
	}
	if separator == -1 || len(os.Args) <= separator+2 {
		return
	}

	configData, err := os.ReadFile(os.Args[separator+1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(90)
	}
	var config helperConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(91)
	}

	observation := helperObservation{
		Name:        os.Args[separator+2],
		Args:        append([]string(nil), os.Args[separator+3:]...),
		Environment: os.Environ(),
		PGPassPath:  os.Getenv("PGPASSFILE"),
	}
	if observation.PGPassPath != "" {
		if info, statErr := os.Stat(observation.PGPassPath); statErr == nil {
			observation.PGPassMode = uint32(info.Mode().Perm())
		}
		if content, readErr := os.ReadFile(observation.PGPassPath); readErr == nil {
			observation.PGPassContent = string(content)
		}
	}
	if config.CreateDump {
		for index := 0; index+1 < len(observation.Args); index++ {
			if observation.Args[index] == "--file" {
				_ = os.WriteFile(observation.Args[index+1], []byte("partial dump"), 0o600)
			}
		}
	}
	if config.Observe != "" {
		data, _ := json.Marshal(observation)
		_ = os.WriteFile(config.Observe, data, 0o600)
	}
	stdout := strings.ReplaceAll(config.Stdout, "${PGPASSFILE}", observation.PGPassPath)
	stderr := strings.ReplaceAll(config.Stderr, "${PGPASSFILE}", observation.PGPassPath)
	stderr = strings.ReplaceAll(stderr, "${PGPASSCONTENT}", observation.PGPassContent)
	_, _ = os.Stdout.WriteString(stdout)
	_, _ = os.Stderr.WriteString(stderr)

	switch config.Behavior {
	case "fail":
		os.Exit(12)
	case "wait":
		time.Sleep(30 * time.Second)
	}
	os.Exit(0)
}

func newTestRunner(t *testing.T, config helperConfig, options ...Option) (*Runner, *fakeProcessFactory) {
	t.Helper()
	directory := t.TempDir()
	config.Observe = filepath.Join(directory, "observation.json")
	configPath := filepath.Join(directory, "config.json")
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	factory := &fakeProcessFactory{config: configPath}
	allOptions := []Option{WithProcessFactory(factory), WithTemporaryDirectory(directory)}
	allOptions = append(allOptions, options...)
	return NewRunner(allOptions...), factory
}

func readObservation(t *testing.T, configPath string) helperObservation {
	t.Helper()
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read helper observation: %v", err)
	}
	var observation helperObservation
	if err := json.Unmarshal(data, &observation); err != nil {
		t.Fatalf("decode helper observation: %v", err)
	}
	return observation
}

func observationPath(factory *fakeProcessFactory) string {
	data, _ := os.ReadFile(factory.config)
	var config helperConfig
	_ = json.Unmarshal(data, &config)
	return config.Observe
}

func TestDumpUsesAllowlistedArgumentsControlledEnvironmentAndPrivatePGPass(t *testing.T) {
	t.Setenv("PGPASSWORD", "inherited-secret")
	t.Setenv("PGOPTIONS", "-c unsafe=value")
	runner, factory := newTestRunner(t, helperConfig{})
	output := filepath.Join(t.TempDir(), "database.dump")
	connection := ConnectionOptions{
		Host: "db.internal", Port: 5433, User: "backup:user", Database: "app\\db",
		SSLMode: SSLModeVerifyFull, Password: "pa:ss\\word",
	}

	if err := runner.Dump(context.Background(), connection, output); err != nil {
		t.Fatalf("Dump() error = %v", err)
	}
	wantArgs := []string{"--no-password", "--format=custom", "--file", output}
	if !reflect.DeepEqual(factory.args, wantArgs) {
		t.Fatalf("arguments = %#v, want %#v", factory.args, wantArgs)
	}
	for _, forbidden := range []string{connection.Password, "inherited-secret", "postgres://"} {
		if strings.Contains(strings.Join(factory.args, " "), forbidden) {
			t.Fatalf("arguments contain secret %q: %#v", forbidden, factory.args)
		}
	}

	observation := readObservation(t, observationPath(factory))
	if observation.PGPassMode != 0o600 {
		t.Fatalf("PGPASSFILE mode = %#o, want 0600", observation.PGPassMode)
	}
	if observation.PGPassContent != `db.internal:5433:app\\db:backup\:user:pa\:ss\\word`+"\n" {
		t.Fatalf("PGPASSFILE content = %q", observation.PGPassContent)
	}
	if _, err := os.Stat(observation.PGPassPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("PGPASSFILE remains after success: %v", err)
	}
	environment := strings.Join(observation.Environment, "\n")
	for _, want := range []string{"PGHOST=db.internal", "PGPORT=5433", "PGUSER=backup:user", "PGDATABASE=app\\db", "PGSSLMODE=verify-full", "PGPASSFILE="} {
		if !strings.Contains(environment, want) {
			t.Errorf("environment missing %q: %s", want, environment)
		}
	}
	for _, forbidden := range []string{"PGPASSWORD=", "PGOPTIONS=", "inherited-secret", "pa:ss\\word"} {
		if strings.Contains(environment, forbidden) {
			t.Errorf("controlled environment contains %q: %s", forbidden, environment)
		}
	}
}

func TestDumpWithoutPasswordCreatesNoCredentialFile(t *testing.T) {
	runner, factory := newTestRunner(t, helperConfig{})
	if err := runner.Dump(context.Background(), ConnectionOptions{Database: "app"}, filepath.Join(t.TempDir(), "dump")); err != nil {
		t.Fatal(err)
	}
	observation := readObservation(t, observationPath(factory))
	if observation.PGPassPath != "" {
		t.Fatalf("PGPASSFILE = %q in no-password mode", observation.PGPassPath)
	}
	if len(factory.args) == 0 || factory.args[0] != "--no-password" {
		t.Fatalf("arguments = %#v, want --no-password", factory.args)
	}
}

func TestPasswordBearingURIIsResolvedOutsideArguments(t *testing.T) {
	runner, factory := newTestRunner(t, helperConfig{})
	uri := "postgresql://backup:uri-secret@db.example:5544/app%2Ddb?sslmode=require" // pragma: allowlist secret
	if err := runner.Dump(context.Background(), ConnectionOptions{URI: uri}, filepath.Join(t.TempDir(), "dump")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(factory.args, " "), uri) || strings.Contains(strings.Join(factory.args, " "), "uri-secret") {
		t.Fatalf("arguments expose URI credentials: %#v", factory.args)
	}
	observation := readObservation(t, observationPath(factory))
	environment := strings.Join(observation.Environment, "\n")
	for _, want := range []string{"PGHOST=db.example", "PGPORT=5544", "PGUSER=backup", "PGDATABASE=app-db", "PGSSLMODE=require"} {
		if !strings.Contains(environment, want) {
			t.Errorf("environment missing %q", want)
		}
	}
	if strings.Contains(environment, uri) || strings.Contains(environment, "uri-secret") {
		t.Fatalf("environment exposes URI credentials: %s", environment)
	}
}

func TestFailureRedactsDiagnosticsRemovesPGPassAndIncompleteDump(t *testing.T) {
	password := "super-secret-password" // pragma: allowlist secret
	uri := "postgresql://backup:" + password + "@db.example/app"
	runner, factory := newTestRunner(t, helperConfig{
		Behavior: "fail", CreateDump: true, Stderr: "password=" + password + " uri=" + uri + " passfile=${PGPASSFILE} content=${PGPASSCONTENT}",
	})
	output := filepath.Join(t.TempDir(), "partial.dump")

	err := runner.Dump(context.Background(), ConnectionOptions{URI: uri}, output)
	if err == nil {
		t.Fatal("Dump() error = nil")
	}
	observation := readObservation(t, observationPath(factory))
	for _, secret := range []string{password, uri, observation.PGPassPath, observation.PGPassContent} { // pragma: allowlist secret
		if secret != "" && strings.Contains(err.Error(), secret) {
			t.Fatalf("error exposes secret %q: %v", secret, err)
		}
	}
	if _, statErr := os.Stat(observation.PGPassPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("PGPASSFILE remains after failure: %v", statErr)
	}
	if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed dump output remains: %v", statErr)
	}
}

func TestDiagnosticsAreBounded(t *testing.T) {
	runner, _ := newTestRunner(t, helperConfig{Behavior: "fail", Stderr: strings.Repeat("x", 4096)}, WithDiagnosticLimit(64))
	err := runner.Restore(context.Background(), ConnectionOptions{}, "input.dump")
	if err == nil || !strings.Contains(err.Error(), "[diagnostic truncated]") {
		t.Fatalf("Restore() error = %v, want truncation marker", err)
	}
	if len(err.Error()) > 200 {
		t.Fatalf("bounded error length = %d, want <= 200", len(err.Error()))
	}
	var partial interface{ PartialChangesPossible() bool }
	if !errors.As(err, &partial) || !partial.PartialChangesPossible() {
		t.Fatalf("started restore error does not report partial-change risk: %v", err)
	}
}

func TestSecretCrossingDiagnosticBoundaryIsFullyRedacted(t *testing.T) {
	password := strings.Repeat("secret", 20)
	runner, _ := newTestRunner(t, helperConfig{
		Behavior: "fail",
		Stderr:   strings.Repeat("x", 60) + password,
	}, WithDiagnosticLimit(64))
	err := runner.Restore(context.Background(), ConnectionOptions{Password: password}, "input.dump")
	if err == nil {
		t.Fatal("Restore() error = nil")
	}
	for length := len("secretsecret"); length <= len(password); length++ {
		if strings.Contains(err.Error(), password[:length]) {
			t.Fatalf("bounded diagnostic exposes a password prefix of length %d: %v", length, err)
		}
	}
}

func TestDumpCancellationAndDeadlinePreserveContextCauseAndCleanup(t *testing.T) {
	t.Run("cancellation", func(t *testing.T) {
		runner, factory := newTestRunner(t, helperConfig{Behavior: "wait", CreateDump: true})
		output := filepath.Join(t.TempDir(), "partial.dump")
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() {
			result <- runner.Dump(ctx, ConnectionOptions{Password: "cancel-secret"}, output)
		}()
		waitForFile(t, observationPath(factory))
		cancel()
		assertCanceledDumpCleanup(t, <-result, context.Canceled, factory, output)
	})

	t.Run("deadline", func(t *testing.T) {
		runner, factory := newTestRunner(t, helperConfig{Behavior: "wait", CreateDump: true})
		output := filepath.Join(t.TempDir(), "partial.dump")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err := runner.Dump(ctx, ConnectionOptions{Password: "deadline-secret"}, output)
		assertCanceledDumpCleanup(t, err, context.DeadlineExceeded, factory, output)
	})
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for helper observation %q", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertCanceledDumpCleanup(t *testing.T, err, want error, factory *fakeProcessFactory, output string) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("Dump() error = %v, want errors.Is(%v)", err, want)
	}
	observation := readObservation(t, observationPath(factory))
	if _, statErr := os.Stat(observation.PGPassPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("PGPASSFILE remains after cancellation: %v", statErr)
	}
	if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("dump output remains after cancellation: %v", statErr)
	}
}

func TestMissingExecutableIsActionable(t *testing.T) {
	for _, test := range []struct {
		client Client
		run    func(*Runner) error
	}{
		{client: PGDump, run: func(runner *Runner) error {
			return runner.Dump(context.Background(), ConnectionOptions{}, filepath.Join(t.TempDir(), "dump"))
		}},
		{client: PGRestore, run: func(runner *Runner) error {
			return runner.Restore(context.Background(), ConnectionOptions{}, "dump")
		}},
	} {
		t.Run(string(test.client), func(t *testing.T) {
			runner, factory := newTestRunner(t, helperConfig{})
			factory.lookupErr = exec.ErrNotFound
			err := test.run(runner)
			if err == nil || !strings.Contains(err.Error(), "install PostgreSQL client tools") || !strings.Contains(err.Error(), string(test.client)) {
				t.Fatalf("run %s error = %v, want actionable dependency error", test.client, err)
			}
			if factory.name != "" {
				t.Fatalf("process started despite discovery failure: %q", factory.name)
			}
			var partial interface{ PartialChangesPossible() bool }
			if errors.As(err, &partial) {
				t.Fatalf("pre-start dependency error reports partial-change risk: %v", err)
			}
		})
	}
}

func TestVersionParsingAndFailures(t *testing.T) {
	for _, test := range []struct {
		name    string
		client  Client
		config  helperConfig
		want    string
		wantErr string
	}{
		{name: "dump", client: PGDump, config: helperConfig{Stdout: "pg_dump (PostgreSQL) 17.4 (Ubuntu 17.4-1)\n"}, want: "17.4"},
		{name: "restore", client: PGRestore, config: helperConfig{Stdout: "pg_restore (PostgreSQL) 16.9\n"}, want: "16.9"},
		{name: "malformed", client: PGDump, config: helperConfig{Stdout: "pg_dump unknown\n"}, wantErr: "malformed"},
		{name: "nonzero", client: PGDump, config: helperConfig{Behavior: "fail", Stderr: "version unavailable"}, wantErr: "version unavailable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner, factory := newTestRunner(t, test.config)
			got, err := runner.Version(context.Background(), test.client)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("Version() = %q, %v, want error containing %q", got, err, test.wantErr)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("Version() = %q, %v, want %q", got, err, test.want)
			}
			if !reflect.DeepEqual(factory.args, []string{"--version"}) {
				t.Fatalf("version arguments = %#v", factory.args)
			}
		})
	}
}

func TestRestoreUsesOnlyFixedArguments(t *testing.T) {
	runner, factory := newTestRunner(t, helperConfig{})
	if err := runner.Restore(context.Background(), ConnectionOptions{Database: "inventory"}, "database.dump"); err != nil {
		t.Fatal(err)
	}
	want := []string{"--dbname=inventory", "--no-password", "--exit-on-error", "database.dump"}
	if !reflect.DeepEqual(factory.args, want) {
		t.Fatalf("Restore() arguments = %#v, want %#v", factory.args, want)
	}
}

func TestConnectionOptionsRejectUntypedOrUnsafeValues(t *testing.T) {
	tests := []ConnectionOptions{
		{SSLMode: SSLMode("arbitrary")},
		{URI: "postgresql://db/app?options=-c%20unsafe=true"},
		{URI: "postgresql://db/app?sslmode=require;password=secret"}, // pragma: allowlist secret
		{URI: "postgresql://db/app", Host: "other"},
		{Password: "line1\nline2"},
		{URI: "mysql://db/app"},
	}
	for _, connection := range tests {
		if _, err := connection.resolve(); err == nil {
			t.Errorf("resolve(%#v) error = nil", connection)
		}
	}
	if err := validateClient(Client("sh -c")); err == nil {
		t.Fatal("validateClient accepted arbitrary executable")
	}
}

func TestAlreadyCanceledContextDoesNotCreateCredentialsOrStartProcess(t *testing.T) {
	runner, factory := newTestRunner(t, helperConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := runner.Dump(ctx, ConnectionOptions{Password: "unused-secret"}, filepath.Join(t.TempDir(), "dump"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Dump() error = %v, want context canceled", err)
	}
	if len(factory.lookups) != 0 || factory.name != "" {
		t.Fatalf("canceled operation performed process work: lookups=%v name=%q", factory.lookups, factory.name)
	}
}
