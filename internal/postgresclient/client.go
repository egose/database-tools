package postgresclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const defaultDiagnosticLimit = 32 * 1024

type SSLMode string

const (
	SSLModeDisable    SSLMode = "disable"
	SSLModeAllow      SSLMode = "allow"
	SSLModePrefer     SSLMode = "prefer"
	SSLModeRequire    SSLMode = "require"
	SSLModeVerifyCA   SSLMode = "verify-ca"
	SSLModeVerifyFull SSLMode = "verify-full"
)

// ConnectionOptions is the shared, typed libpq connection contract for the
// PostgreSQL archive and restore commands.
type ConnectionOptions struct {
	Host     string
	Port     uint16
	User     string
	Database string
	SSLMode  SSLMode
	URI      string
	Password string
}

// Validate verifies the allowlisted libpq settings without creating files or
// discovering or starting a PostgreSQL client process.
func (connection ConnectionOptions) Validate() error {
	_, err := connection.resolve()
	return sanitizeError(err, connection.secrets())
}

// DatabaseName returns the resolved database name for archive metadata.
func (connection ConnectionOptions) DatabaseName() (string, error) {
	resolved, err := connection.resolve()
	if err != nil {
		return "", sanitizeError(err, connection.secrets())
	}
	return resolved.database, nil
}

type Client string

const (
	PGDump    Client = "pg_dump"
	PGRestore Client = "pg_restore"
)

// ProcessFactory is injectable so callers can test process behavior without
// installing PostgreSQL clients. Implementations must execute name directly.
type ProcessFactory interface {
	LookPath(file string) (string, error)
	CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd
}

type Runner struct {
	process         ProcessFactory
	tempDir         string
	diagnosticLimit int
	environ         func() []string
}

type Option func(*Runner)

func WithProcessFactory(factory ProcessFactory) Option {
	return func(r *Runner) {
		if factory != nil {
			r.process = factory
		}
	}
}

func WithTemporaryDirectory(path string) Option {
	return func(r *Runner) { r.tempDir = path }
}

func WithDiagnosticLimit(bytes int) Option {
	return func(r *Runner) {
		if bytes > 0 {
			r.diagnosticLimit = bytes
		}
	}
}

func NewRunner(options ...Option) *Runner {
	runner := &Runner{
		process:         osProcessFactory{},
		diagnosticLimit: defaultDiagnosticLimit,
		environ:         os.Environ,
	}
	for _, option := range options {
		option(runner)
	}
	return runner
}

// Dump writes a PostgreSQL custom-format dump to outputPath.
func (r *Runner) Dump(ctx context.Context, connection ConnectionOptions, outputPath string) error {
	if outputPath == "" {
		return errors.New("PostgreSQL dump output path is required")
	}

	err := r.run(ctx, PGDump, connection, []string{
		"--no-password",
		"--format=custom",
		"--file", outputPath,
	})
	if err == nil {
		return nil
	}

	if removeErr := os.Remove(outputPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		err = errors.Join(err, fmt.Errorf("remove incomplete PostgreSQL dump: %w", removeErr))
	}
	return sanitizeError(err, connection.secrets())
}

// Restore restores a custom-format dump and stops on the first reported error.
func (r *Runner) Restore(ctx context.Context, connection ConnectionOptions, inputPath string) error {
	if inputPath == "" {
		return errors.New("PostgreSQL restore input path is required")
	}
	return r.run(ctx, PGRestore, connection, []string{
		"--no-password",
		"--exit-on-error",
		inputPath,
	})
}

// Version returns the parsed PostgreSQL client version without imposing a
// client/server compatibility policy.
func (r *Runner) Version(ctx context.Context, client Client) (string, error) {
	if err := validateClient(client); err != nil {
		return "", err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("inspect %s version: %w", client, err)
	}

	path, err := r.discover(client)
	if err != nil {
		return "", err
	}
	stdout := newBoundedBuffer(r.diagnosticLimit)
	stderr := newBoundedBuffer(r.diagnosticLimit)
	cmd := r.process.CommandContext(ctx, path, "--version")
	cmd.Env = r.controlledEnvironment(nil)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", fmt.Errorf("inspect %s version: %w", client, ctxErr)
	}
	if err != nil {
		return "", processError("inspect "+string(client)+" version", err, stderr.String(), nil)
	}

	version, parseErr := parseVersion(client, stdout.String())
	if parseErr != nil {
		return "", processError("inspect "+string(client)+" version", parseErr, stdout.String(), nil)
	}
	return version, nil
}

func (r *Runner) run(ctx context.Context, client Client, connection ConnectionOptions, args []string) (retErr error) {
	if err := validateClient(client); err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("run %s: %w", client, err)
	}

	resolved, err := connection.resolve()
	if err != nil {
		return sanitizeError(err, connection.secrets())
	}
	path, err := r.discover(client)
	if err != nil {
		return err
	}

	pgpassPath := ""
	if resolved.password != "" {
		pgpassPath, err = r.writePasswordFile(resolved)
		if err != nil {
			return sanitizeError(fmt.Errorf("create private PostgreSQL credential file: %w", err), connection.secrets())
		}
		defer func() {
			if removeErr := os.Remove(pgpassPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				retErr = errors.Join(retErr, fmt.Errorf("remove private PostgreSQL credential file: %w", removeErr))
			}
			retErr = sanitizeError(retErr, append(connection.secrets(), pgpassPath))
		}()
	}

	secrets := append(connection.secrets(), pgpassPath)
	stdout := newBoundedBuffer(r.diagnosticLimit)
	stderr := newDiagnosticBuffer(r.diagnosticLimit, secrets)
	if client == PGRestore && resolved.database != "" {
		args = append([]string{"--dbname=" + resolved.database}, args...)
	}
	cmd := r.process.CommandContext(ctx, path, args...)
	cmd.Env = r.controlledEnvironment(resolved.environment(pgpassPath))
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err = cmd.Start(); err != nil {
		return processError("start "+string(client), err, stderr.String(), secrets)
	}
	err = cmd.Wait()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return startedProcessError{cause: processError("run "+string(client), ctxErr, stderr.String(), secrets)}
	}
	if err != nil {
		return startedProcessError{cause: processError("run "+string(client), err, stderr.String(), secrets)}
	}
	return nil
}

type startedProcessError struct{ cause error }

func (err startedProcessError) Error() string            { return err.cause.Error() }
func (err startedProcessError) Unwrap() error            { return err.cause }
func (startedProcessError) PartialChangesPossible() bool { return true }

func (r *Runner) discover(client Client) (string, error) {
	path, err := r.process.LookPath(string(client))
	if err != nil {
		return "", fmt.Errorf("required PostgreSQL client %q was not found in PATH; install PostgreSQL client tools and ensure %s is executable", client, client)
	}
	return path, nil
}

func (r *Runner) writePasswordFile(connection resolvedConnection) (string, error) {
	file, err := os.CreateTemp(r.tempDir, ".database-tools-pgpass-*")
	if err != nil {
		return "", err
	}
	path := file.Name()
	ok := false
	defer func() {
		if !ok {
			_ = file.Close()
			_ = os.Remove(path)
		}
	}()

	if err := file.Chmod(0o600); err != nil {
		return "", err
	}
	line := strings.Join([]string{
		pgpassField(connection.host),
		pgpassPort(connection.port),
		pgpassField(connection.database),
		pgpassField(connection.user),
		pgpassEscape(connection.password),
	}, ":") + "\n"
	if _, err := file.WriteString(line); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	ok = true
	return path, nil
}

func (r *Runner) controlledEnvironment(additions []string) []string {
	preserved := make([]string, 0, len(r.environ())+len(additions))
	for _, entry := range r.environ() {
		key, _, found := strings.Cut(entry, "=")
		if found && preserveEnvironmentKey(strings.ToUpper(key)) {
			preserved = append(preserved, entry)
		}
	}
	return append(preserved, additions...)
}

func preserveEnvironmentKey(key string) bool {
	switch key {
	case "PATH", "HOME", "USERPROFILE", "TMPDIR", "TMP", "TEMP", "SYSTEMROOT", "WINDIR", "LANG", "TZ", "SSL_CERT_FILE", "SSL_CERT_DIR":
		return true
	default:
		return strings.HasPrefix(key, "LC_")
	}
}

type resolvedConnection struct {
	host     string
	port     uint16
	user     string
	database string
	sslMode  SSLMode
	password string
}

func (connection ConnectionOptions) resolve() (resolvedConnection, error) {
	if err := validateConnectionValue("host", connection.Host); err != nil {
		return resolvedConnection{}, err
	}
	if err := validateConnectionValue("user", connection.User); err != nil {
		return resolvedConnection{}, err
	}
	if err := validateConnectionValue("database", connection.Database); err != nil {
		return resolvedConnection{}, err
	}
	if err := validateConnectionValue("password", connection.Password); err != nil {
		return resolvedConnection{}, err
	}
	if err := validateSSLMode(connection.SSLMode); err != nil {
		return resolvedConnection{}, err
	}

	if connection.URI == "" {
		return resolvedConnection{
			host: connection.Host, port: connection.Port, user: connection.User,
			database: connection.Database, sslMode: connection.SSLMode, password: connection.Password,
		}, nil
	}
	if connection.Host != "" || connection.Port != 0 || connection.User != "" || connection.Database != "" || connection.SSLMode != "" {
		return resolvedConnection{}, errors.New("PostgreSQL URI cannot be combined with discrete host, port, user, database, or SSL mode options")
	}

	parsed, err := url.Parse(connection.URI)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" || parsed.Fragment != "" {
		return resolvedConnection{}, errors.New("invalid PostgreSQL URI")
	}
	result := resolvedConnection{host: parsed.Hostname(), password: connection.Password}
	if parsed.User != nil {
		result.user = parsed.User.Username()
		if uriPassword, present := parsed.User.Password(); present {
			if result.password != "" && result.password != uriPassword { // pragma: allowlist secret
				return resolvedConnection{}, errors.New("PostgreSQL URI password conflicts with the discrete password option")
			}
			result.password = uriPassword // pragma: allowlist secret
		}
	}
	if port := parsed.Port(); port != "" {
		value, parseErr := strconv.ParseUint(port, 10, 16)
		if parseErr != nil || value == 0 {
			return resolvedConnection{}, errors.New("invalid PostgreSQL URI port")
		}
		result.port = uint16(value)
	}
	if parsed.Path != "" && parsed.Path != "/" {
		if strings.Count(strings.TrimPrefix(parsed.EscapedPath(), "/"), "/") != 0 {
			return resolvedConnection{}, errors.New("invalid PostgreSQL URI database path")
		}
		result.database = strings.TrimPrefix(parsed.Path, "/")
	}

	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return resolvedConnection{}, errors.New("invalid PostgreSQL URI query")
	}
	for key, values := range query {
		if len(values) != 1 || (key != "sslmode" && key != "password") {
			return resolvedConnection{}, fmt.Errorf("unsupported PostgreSQL URI option %q", key)
		}
	}
	if value := query.Get("sslmode"); value != "" {
		result.sslMode = SSLMode(value)
		if err := validateSSLMode(result.sslMode); err != nil {
			return resolvedConnection{}, err
		}
	}
	if query.Has("password") {
		uriPassword := query.Get("password")
		if result.password != "" && result.password != uriPassword { // pragma: allowlist secret
			return resolvedConnection{}, errors.New("PostgreSQL URI password conflicts with another password option")
		}
		result.password = uriPassword // pragma: allowlist secret
	}
	for name, value := range map[string]string{
		"host": result.host, "user": result.user, "database": result.database, "password": result.password,
	} {
		if err := validateConnectionValue(name, value); err != nil {
			return resolvedConnection{}, err
		}
	}
	return result, nil
}

func (connection resolvedConnection) environment(pgpassPath string) []string {
	environment := make([]string, 0, 6)
	if connection.host != "" {
		environment = append(environment, "PGHOST="+connection.host)
	}
	if connection.port != 0 {
		environment = append(environment, "PGPORT="+strconv.FormatUint(uint64(connection.port), 10))
	}
	if connection.user != "" {
		environment = append(environment, "PGUSER="+connection.user)
	}
	if connection.database != "" {
		environment = append(environment, "PGDATABASE="+connection.database)
	}
	if connection.sslMode != "" {
		environment = append(environment, "PGSSLMODE="+string(connection.sslMode))
	}
	if pgpassPath != "" {
		environment = append(environment, "PGPASSFILE="+pgpassPath)
	}
	return environment
}

func (connection ConnectionOptions) secrets() []string {
	secrets := []string{connection.Password, connection.URI}
	if parsed, err := url.Parse(connection.URI); err == nil {
		if parsed.User != nil {
			if password, ok := parsed.User.Password(); ok {
				secrets = append(secrets, password, pgpassEscape(password))
			}
		}
		queryPassword := parsed.Query().Get("password")
		secrets = append(secrets, queryPassword, pgpassEscape(queryPassword))
	}
	if connection.Password != "" {
		secrets = append(secrets, pgpassEscape(connection.Password))
	}
	return secrets
}

func validateClient(client Client) error {
	if client != PGDump && client != PGRestore {
		return fmt.Errorf("unsupported PostgreSQL client %q", client)
	}
	return nil
}

func validateSSLMode(mode SSLMode) error {
	switch mode {
	case "", SSLModeDisable, SSLModeAllow, SSLModePrefer, SSLModeRequire, SSLModeVerifyCA, SSLModeVerifyFull:
		return nil
	default:
		return fmt.Errorf("unsupported PostgreSQL SSL mode %q", mode)
	}
}

func validateConnectionValue(name, value string) error {
	if strings.ContainsAny(value, "\x00\r\n") {
		return fmt.Errorf("PostgreSQL %s contains an invalid control character", name)
	}
	return nil
}

func pgpassField(value string) string {
	if value == "" {
		return "*"
	}
	return pgpassEscape(value)
}

func pgpassPort(port uint16) string {
	if port == 0 {
		return "*"
	}
	return strconv.FormatUint(uint64(port), 10)
}

func pgpassEscape(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, ":", `\:`)
}

var versionPattern = regexp.MustCompile(`^([^\s]+) \(PostgreSQL\) ([^\s]+)(?:\s.*)?$`)

func parseVersion(client Client, output string) (string, error) {
	line := strings.TrimSpace(output)
	matches := versionPattern.FindStringSubmatch(line)
	if len(matches) != 3 || matches[1] != string(client) {
		return "", errors.New("malformed PostgreSQL client version output")
	}
	return matches[2], nil
}

type boundedBuffer struct {
	buffer       bytes.Buffer
	captureLimit int
	outputLimit  int
	secrets      []string
	truncated    bool
}

func newBoundedBuffer(limit int) *boundedBuffer {
	return &boundedBuffer{captureLimit: limit, outputLimit: limit}
}

func newDiagnosticBuffer(limit int, secrets []string) *boundedBuffer {
	longestSecret := 0
	for _, secret := range secrets { // pragma: allowlist secret
		if len(secret) > longestSecret {
			longestSecret = len(secret)
		}
	}
	return &boundedBuffer{
		captureLimit: limit + longestSecret,
		outputLimit:  limit,
		secrets:      secrets,
	}
}

func (buffer *boundedBuffer) Write(value []byte) (int, error) {
	written := len(value)
	remaining := buffer.captureLimit - buffer.buffer.Len()
	if remaining > 0 {
		if len(value) > remaining {
			_, _ = buffer.buffer.Write(value[:remaining])
			buffer.truncated = true
		} else {
			_, _ = buffer.buffer.Write(value)
		}
	} else if len(value) > 0 {
		buffer.truncated = true
	}
	return written, nil
}

func (buffer *boundedBuffer) String() string {
	value := sanitize(buffer.buffer.String(), buffer.secrets)
	if len(value) > buffer.outputLimit {
		value = value[:buffer.outputLimit]
		buffer.truncated = true
	}
	if buffer.truncated {
		return value + " [diagnostic truncated]"
	}
	return value
}

var (
	connectionURIPattern = regexp.MustCompile(`(?i)postgres(?:ql)?://[^\s]+`)
	secretOptionPattern  = regexp.MustCompile(`(?i)((?:password|sslpassword|passfile)\s*=\s*)[^\s]+`)
)

func processError(operation string, cause error, diagnostic string, secrets []string) error {
	diagnostic = sanitize(diagnostic, secrets)
	if strings.TrimSpace(diagnostic) == "" {
		return fmt.Errorf("%s failed: %w", operation, cause)
	}
	return fmt.Errorf("%s failed: %s: %w", operation, strings.TrimSpace(diagnostic), cause)
}

func sanitizeError(err error, secrets []string) error {
	if err == nil {
		return nil
	}
	message := sanitize(err.Error(), secrets)
	return wrappedSanitizedError{message: message, cause: err}
}

type wrappedSanitizedError struct {
	message string
	cause   error
}

func (err wrappedSanitizedError) Error() string { return err.message }
func (err wrappedSanitizedError) Unwrap() error { return err.cause }
func (err wrappedSanitizedError) Is(target error) bool {
	return errors.Is(err.cause, target)
}

func sanitize(value string, secrets []string) string {
	for _, secret := range secrets { // pragma: allowlist secret
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[REDACTED]")
		}
	}
	value = connectionURIPattern.ReplaceAllString(value, `[REDACTED]`)
	return secretOptionPattern.ReplaceAllString(value, `${1}[REDACTED]`)
}

type osProcessFactory struct{}

func (osProcessFactory) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (osProcessFactory) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
