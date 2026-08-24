package postgresarchive

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/egose/database-tools/internal/postgresclient"
	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/notification"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
)

const (
	envPrefix           = "POSTGRESARCHIVE__"
	fallbackEnvPrefix   = "POSTGRES__"
	defaultCronExpr     = "0 2 * * *"
	defaultWorkspaceDir = "postgresarchive"
)

type Config struct {
	Connection postgresclient.ConnectionOptions
	toolconfig.StorageOptions
	RetentionOptions
	NotificationOptions
	ScheduleOptions
	RuntimeOptions
	Keep bool
}

type RetentionOptions struct{ ExpiryDays int }

type ScheduleOptions struct {
	Cron           bool
	CronExpression string
	Location       *time.Location
}

type RuntimeOptions struct {
	WorkspaceBasePath       string
	StorageOperationTimeout time.Duration
	NotificationTimeout     time.Duration
}

type NotificationOptions struct {
	RocketChatWebhookURL, RocketChatWebhookPrefix  string
	RocketChatNotifyOnFailureOnly                  bool
	SlackWebhookURL, SlackWebhookPrefix            string
	SlackNotifyOnFailureOnly                       bool
	SMTPHost, SMTPPort, SMTPUsername, SMTPPassword string
	SMTPFrom, SMTPTo, SMTPSubjectPrefix            string
	SMTPNotifyOnFailureOnly                        bool
	SMTPAllowInsecureNoTLSInDevelopment            bool
	SESEndpoint, SESRegion                         string
	SESAccessKeyID, SESSecretAccessKey             string
	SESFrom, SESTo, SESSubjectPrefix               string
	SESNotifyOnFailureOnly                         bool
	NotificationAllowInsecureHTTPInDevelopment     bool
}

var flagDefs = struct {
	host, port, user, database, sslMode, uri, password toolconfig.StringFlagDef
	expiryDays                                         toolconfig.StringFlagDef
	rocketURL, rocketPrefix                            toolconfig.StringFlagDef
	rocketFailureOnly                                  toolconfig.BoolFlagDef
	slackURL, slackPrefix                              toolconfig.StringFlagDef
	slackFailureOnly                                   toolconfig.BoolFlagDef
	smtpHost, smtpPort, smtpUsername, smtpPassword     toolconfig.StringFlagDef
	smtpFrom, smtpTo, smtpPrefix                       toolconfig.StringFlagDef
	smtpFailureOnly, smtpAllowInsecure                 toolconfig.BoolFlagDef
	sesEndpoint, sesRegion, sesAccessKey, sesSecret    toolconfig.StringFlagDef
	sesFrom, sesTo, sesPrefix                          toolconfig.StringFlagDef
	sesFailureOnly, allowInsecureHTTP                  toolconfig.BoolFlagDef
	cron                                               toolconfig.BoolFlagDef
	cronExpression, tz                                 toolconfig.StringFlagDef
	keep, version                                      toolconfig.BoolFlagDef
}{
	host:              toolconfig.StringFlagDef{Name: "host", EnvKey: "HOST", Usage: "PostgreSQL server host"},
	port:              toolconfig.StringFlagDef{Name: "port", EnvKey: "PORT", Usage: "PostgreSQL server port"},
	user:              toolconfig.StringFlagDef{Name: "user", EnvKey: "USER", Usage: "PostgreSQL user"},
	database:          toolconfig.StringFlagDef{Name: "database", EnvKey: "DATABASE", Usage: "PostgreSQL database to archive"},
	sslMode:           toolconfig.StringFlagDef{Name: "ssl-mode", EnvKey: "SSL_MODE", Usage: "libpq SSL mode (disable, allow, prefer, require, verify-ca, verify-full)"},
	uri:               toolconfig.StringFlagDef{Name: "uri", EnvKey: "URI", Usage: "PostgreSQL connection URI"},
	password:          toolconfig.StringFlagDef{Name: "password", EnvKey: "PASSWORD", Usage: "PostgreSQL password"},
	expiryDays:        toolconfig.StringFlagDef{Name: "expiry-days", EnvKey: "EXPIRY_DAYS", Usage: "maximum archive age in days"},
	rocketURL:         toolconfig.StringFlagDef{Name: "rocketchat-webhook-url", EnvKey: "ROCKETCHAT_WEBHOOK_URL", Usage: "Rocket.Chat webhook URL"},
	rocketPrefix:      toolconfig.StringFlagDef{Name: "rocketchat-webhook-prefix", EnvKey: "ROCKETCHAT_WEBHOOK_PREFIX", Usage: "Rocket.Chat message prefix"},
	rocketFailureOnly: toolconfig.BoolFlagDef{Name: "rocketchat-notify-on-failure-only", EnvKey: "ROCKETCHAT_NOTIFY_ON_FAILURE_ONLY", Usage: "send Rocket.Chat notifications only on failure"},
	slackURL:          toolconfig.StringFlagDef{Name: "slack-webhook-url", EnvKey: "SLACK_WEBHOOK_URL", Usage: "Slack webhook URL"},
	slackPrefix:       toolconfig.StringFlagDef{Name: "slack-webhook-prefix", EnvKey: "SLACK_WEBHOOK_PREFIX", Usage: "Slack message prefix"},
	slackFailureOnly:  toolconfig.BoolFlagDef{Name: "slack-notify-on-failure-only", EnvKey: "SLACK_NOTIFY_ON_FAILURE_ONLY", Usage: "send Slack notifications only on failure"},
	smtpHost:          toolconfig.StringFlagDef{Name: "smtp-host", EnvKey: "SMTP_HOST", Usage: "SMTP server host"},
	smtpPort:          toolconfig.StringFlagDef{Name: "smtp-port", EnvKey: "SMTP_PORT", Usage: "SMTP server port", Defaults: []string{"587"}},
	smtpUsername:      toolconfig.StringFlagDef{Name: "smtp-username", EnvKey: "SMTP_USERNAME", Usage: "SMTP username"},
	smtpPassword:      toolconfig.StringFlagDef{Name: "smtp-password", EnvKey: "SMTP_PASSWORD", Usage: "SMTP password"},
	smtpFrom:          toolconfig.StringFlagDef{Name: "smtp-from", EnvKey: "SMTP_FROM", Usage: "SMTP sender"},
	smtpTo:            toolconfig.StringFlagDef{Name: "smtp-to", EnvKey: "SMTP_TO", Usage: "comma-separated SMTP recipients"},
	smtpPrefix:        toolconfig.StringFlagDef{Name: "smtp-subject-prefix", EnvKey: "SMTP_SUBJECT_PREFIX", Usage: "SMTP subject prefix"},
	smtpFailureOnly:   toolconfig.BoolFlagDef{Name: "smtp-notify-on-failure-only", EnvKey: "SMTP_NOTIFY_ON_FAILURE_ONLY", Usage: "send SMTP notifications only on failure"},
	smtpAllowInsecure: toolconfig.BoolFlagDef{Name: "smtp-allow-insecure-no-tls-in-development", EnvKey: "SMTP_ALLOW_INSECURE_NO_TLS_IN_DEVELOPMENT", Usage: "allow SMTP without STARTTLS for development"},
	sesEndpoint:       toolconfig.StringFlagDef{Name: "ses-endpoint", EnvKey: "SES_ENDPOINT", Usage: "AWS SES endpoint override"},
	sesRegion:         toolconfig.StringFlagDef{Name: "ses-region", EnvKey: "SES_REGION", Usage: "AWS SES region"},
	sesAccessKey:      toolconfig.StringFlagDef{Name: "ses-access-key-id", EnvKey: "SES_ACCESS_KEY_ID", Usage: "AWS SES access key ID"},
	sesSecret:         toolconfig.StringFlagDef{Name: "ses-secret-access-key", EnvKey: "SES_SECRET_ACCESS_KEY", Usage: "AWS SES secret access key"},
	sesFrom:           toolconfig.StringFlagDef{Name: "ses-from", EnvKey: "SES_FROM", Usage: "AWS SES sender"},
	sesTo:             toolconfig.StringFlagDef{Name: "ses-to", EnvKey: "SES_TO", Usage: "comma-separated AWS SES recipients"},
	sesPrefix:         toolconfig.StringFlagDef{Name: "ses-subject-prefix", EnvKey: "SES_SUBJECT_PREFIX", Usage: "AWS SES subject prefix"},
	sesFailureOnly:    toolconfig.BoolFlagDef{Name: "ses-notify-on-failure-only", EnvKey: "SES_NOTIFY_ON_FAILURE_ONLY", Usage: "send AWS SES notifications only on failure"},
	allowInsecureHTTP: toolconfig.BoolFlagDef{Name: "notification-allow-insecure-http-in-development", EnvKey: "NOTIFICATION_ALLOW_INSECURE_HTTP_IN_DEVELOPMENT", Usage: "allow HTTP notification endpoints for development"},
	cron:              toolconfig.BoolFlagDef{Name: "cron", EnvKey: "CRON", Usage: "run as a cron scheduler"},
	cronExpression:    toolconfig.StringFlagDef{Name: "cron-expression", EnvKey: "CRON_EXPRESSION", Usage: "cron schedule expression"},
	tz:                toolconfig.StringFlagDef{Name: "tz", EnvKey: "TZ", Usage: "scheduler time zone"},
	keep:              toolconfig.BoolFlagDef{Name: "keep", EnvKey: "KEEP", Usage: "keep the private per-run workspace"},
	version:           toolconfig.BoolFlagDef{Name: "version", Usage: "show the version"},
}

func ParseFlags() (*Config, bool, error) {
	return parseFlags(flag.CommandLine, utils.NewEnv(envPrefix, fallbackEnvPrefix, ""), os.Args[1:])
}

func parseFlags(fs *flag.FlagSet, env toolconfig.EnvReader, args []string) (*Config, bool, error) {
	host, port, user := flagDefs.host.Bind(fs, env), flagDefs.port.Bind(fs, env), flagDefs.user.Bind(fs, env)
	database, sslMode := flagDefs.database.Bind(fs, env), flagDefs.sslMode.Bind(fs, env)
	uri, password := flagDefs.uri.Bind(fs, env), flagDefs.password.Bind(fs, env)
	storageBindings := toolconfig.BindStorageFlagsWithDefaultPrefix(fs, env, storage.PostgreSQLDefaultBackupPrefix)
	expiryDays := flagDefs.expiryDays.Bind(fs, env)
	rocketURL, rocketPrefix, rocketFailureOnly := flagDefs.rocketURL.Bind(fs, env), flagDefs.rocketPrefix.Bind(fs, env), flagDefs.rocketFailureOnly.Bind(fs, env)
	slackURL, slackPrefix, slackFailureOnly := flagDefs.slackURL.Bind(fs, env), flagDefs.slackPrefix.Bind(fs, env), flagDefs.slackFailureOnly.Bind(fs, env)
	smtpHost, smtpPort := flagDefs.smtpHost.Bind(fs, env), flagDefs.smtpPort.Bind(fs, env)
	smtpUsername, smtpPassword := flagDefs.smtpUsername.Bind(fs, env), flagDefs.smtpPassword.Bind(fs, env)
	smtpFrom, smtpTo, smtpPrefix := flagDefs.smtpFrom.Bind(fs, env), flagDefs.smtpTo.Bind(fs, env), flagDefs.smtpPrefix.Bind(fs, env)
	smtpFailureOnly, smtpAllowInsecure := flagDefs.smtpFailureOnly.Bind(fs, env), flagDefs.smtpAllowInsecure.Bind(fs, env)
	sesEndpoint := flagDefs.sesEndpoint.Bind(fs, env)
	sesRegion := fs.String(flagDefs.sesRegion.Name, env.GetValue("SES_REGION", env.GetValue("AWS_REGION")), flagDefs.sesRegion.Usage)
	sesAccessKey := fs.String(flagDefs.sesAccessKey.Name, env.GetValue("SES_ACCESS_KEY_ID", env.GetValue("AWS_ACCESS_KEY_ID")), flagDefs.sesAccessKey.Usage)
	sesSecret := fs.String(flagDefs.sesSecret.Name, env.GetValue("SES_SECRET_ACCESS_KEY", env.GetValue("AWS_SECRET_ACCESS_KEY")), flagDefs.sesSecret.Usage)
	sesFrom, sesTo, sesPrefix := flagDefs.sesFrom.Bind(fs, env), flagDefs.sesTo.Bind(fs, env), flagDefs.sesPrefix.Bind(fs, env)
	sesFailureOnly, allowInsecureHTTP := flagDefs.sesFailureOnly.Bind(fs, env), flagDefs.allowInsecureHTTP.Bind(fs, env)
	cron, cronExpression, tz := flagDefs.cron.Bind(fs, env), flagDefs.cronExpression.Bind(fs, env), flagDefs.tz.Bind(fs, env)
	keep, showVersion := flagDefs.keep.Bind(fs, env), flagDefs.version.Bind(fs, env)
	if err := fs.Parse(args); err != nil {
		return nil, false, err
	}

	parsedPort, err := parsePort(*port)
	if err != nil {
		return nil, false, err
	}
	parsedExpiry, err := parseExpiryDays(*expiryDays)
	if err != nil {
		return nil, false, err
	}
	location, err := parseLocation(*tz)
	if err != nil {
		return nil, false, fmt.Errorf("invalid time zone: %w", err)
	}
	cfg := &Config{
		Connection:       postgresclient.ConnectionOptions{Host: *host, Port: parsedPort, User: *user, Database: *database, SSLMode: postgresclient.SSLMode(*sslMode), URI: *uri, Password: *password},
		RetentionOptions: RetentionOptions{ExpiryDays: parsedExpiry},
		ScheduleOptions:  ScheduleOptions{Cron: *cron, CronExpression: cronExpressionOrDefault(*cronExpression), Location: location},
		Keep:             *keep,
		NotificationOptions: NotificationOptions{
			RocketChatWebhookURL: *rocketURL, RocketChatWebhookPrefix: *rocketPrefix, RocketChatNotifyOnFailureOnly: *rocketFailureOnly,
			SlackWebhookURL: *slackURL, SlackWebhookPrefix: *slackPrefix, SlackNotifyOnFailureOnly: *slackFailureOnly,
			SMTPHost: *smtpHost, SMTPPort: *smtpPort, SMTPUsername: *smtpUsername, SMTPPassword: *smtpPassword, SMTPFrom: *smtpFrom, SMTPTo: *smtpTo, SMTPSubjectPrefix: *smtpPrefix, SMTPNotifyOnFailureOnly: *smtpFailureOnly, SMTPAllowInsecureNoTLSInDevelopment: *smtpAllowInsecure,
			SESEndpoint: *sesEndpoint, SESRegion: *sesRegion, SESAccessKeyID: *sesAccessKey, SESSecretAccessKey: *sesSecret, SESFrom: *sesFrom, SESTo: *sesTo, SESSubjectPrefix: *sesPrefix, SESNotifyOnFailureOnly: *sesFailureOnly,
			NotificationAllowInsecureHTTPInDevelopment: *allowInsecureHTTP,
		},
	}
	storageBindings.Apply(&cfg.StorageOptions)
	if *showVersion {
		return cfg, true, nil
	}
	cfg.RuntimeOptions, err = parseRuntimeOptions(env)
	if err != nil {
		return nil, false, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, false, err
	}
	return cfg, false, nil
}

func parsePort(raw string) (uint16, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(raw, 10, 16)
	if err != nil || value == 0 {
		return 0, errors.New("port must be an integer from 1 through 65535")
	}
	return uint16(value), nil
}

func parseExpiryDays(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, errors.New("expiry-days must be a non-negative integer")
	}
	return value, nil
}

func parseLocation(raw string) (*time.Location, error) {
	if raw == "" {
		return time.Local, nil
	}
	return time.LoadLocation(raw)
}

func cronExpressionOrDefault(raw string) string {
	if raw == "" {
		return defaultCronExpr
	}
	return raw
}

func parseRuntimeOptions(env toolconfig.EnvReader) (RuntimeOptions, error) {
	storageTimeout, err := toolconfig.ReadOptionalDuration(env, "STORAGE_OPERATION_TIMEOUT")
	if err != nil {
		return RuntimeOptions{}, err
	}
	notificationTimeout, err := toolconfig.ReadOptionalDuration(env, "NOTIFICATION_TIMEOUT")
	if err != nil {
		return RuntimeOptions{}, err
	}
	return RuntimeOptions{
		WorkspaceBasePath:       toolconfig.ReadWorkspaceBase(env, "DUMP_PATH", filepath.Join(os.TempDir(), defaultWorkspaceDir)),
		StorageOperationTimeout: storageTimeout,
		NotificationTimeout:     notificationTimeout,
	}, nil
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("PostgreSQL archive configuration is required")
	}
	if err := c.Connection.Validate(); err != nil {
		return err
	}
	database, err := c.Connection.DatabaseName()
	if err != nil {
		return err
	}
	if database == "" {
		return errors.New("PostgreSQL database is required through --database or the URI path")
	}
	if c.StorageOperationTimeout < 0 {
		return errors.New("STORAGE_OPERATION_TIMEOUT must be greater than zero")
	}
	if c.NotificationTimeout < 0 {
		return errors.New("NOTIFICATION_TIMEOUT must be greater than zero")
	}
	if err := c.StorageOptions.Validate(); err != nil {
		return err
	}
	return c.NotificationOptions.Validate()
}

func (c *Config) GetStorages(ctx context.Context) ([]storage.ArchiveBackend, error) {
	options := c.StorageOptions
	options.BackupPrefix = storage.NormalizeBackupPrefixWithDefault(options.BackupPrefix, storage.PostgreSQLDefaultBackupPrefix)
	return options.GetArchiveStorages(ctx, c.ExpiryDays)
}

func (c *Config) GetCronExpression() string   { return cronExpressionOrDefault(c.CronExpression) }
func (c *Config) GetLocation() *time.Location { return c.Location }
func (c *Config) HasCron() bool               { return c.Cron }
func (c *Config) HasKeep() bool               { return c.Keep }

func (o NotificationOptions) Validate() error {
	if o.RocketChatWebhookURL != "" {
		if err := notification.ValidateWebhookURL(o.RocketChatWebhookURL, o.NotificationAllowInsecureHTTPInDevelopment, "Rocket.Chat webhook URL"); err != nil {
			return err
		}
	}
	if o.SlackWebhookURL != "" {
		if err := notification.ValidateWebhookURL(o.SlackWebhookURL, o.NotificationAllowInsecureHTTPInDevelopment, "Slack webhook URL"); err != nil {
			return err
		}
	}
	if o.SMTPHost != "" || o.SMTPFrom != "" || o.SMTPTo != "" {
		if _, err := notification.ValidateSMTPOptions(o.SMTPHost, o.SMTPPort, o.SMTPUsername, o.SMTPPassword, o.SMTPFrom, o.SMTPTo, o.SMTPSubjectPrefix); err != nil {
			return err
		}
	}
	if o.SESFrom != "" || o.SESTo != "" {
		if _, err := notification.ValidateSESOptions(o.SESEndpoint, o.SESRegion, o.SESAccessKeyID, o.SESSecretAccessKey, o.SESFrom, o.SESTo, o.NotificationAllowInsecureHTTPInDevelopment); err != nil {
			return err
		}
	}
	return nil
}

func (o NotificationOptions) GetNotifications() ([]notification.Notification, error) {
	result := make([]notification.Notification, 0, 4)
	if o.RocketChatWebhookURL != "" {
		item := new(notification.RocketChat)
		if err := item.Init(o.RocketChatWebhookURL, o.RocketChatWebhookPrefix, o.RocketChatNotifyOnFailureOnly, o.NotificationAllowInsecureHTTPInDevelopment); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if o.SlackWebhookURL != "" {
		item := new(notification.Slack)
		if err := item.Init(o.SlackWebhookURL, o.SlackWebhookPrefix, o.SlackNotifyOnFailureOnly, o.NotificationAllowInsecureHTTPInDevelopment); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if o.SMTPHost != "" || o.SMTPFrom != "" || o.SMTPTo != "" {
		item := new(notification.SMTP)
		if err := item.Init(o.SMTPHost, o.SMTPPort, o.SMTPUsername, o.SMTPPassword, o.SMTPFrom, o.SMTPTo, o.SMTPSubjectPrefix, o.SMTPNotifyOnFailureOnly, o.SMTPAllowInsecureNoTLSInDevelopment); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if o.SESFrom != "" || o.SESTo != "" {
		item := new(notification.SES)
		if err := item.Init(o.SESEndpoint, o.SESRegion, o.SESAccessKeyID, o.SESSecretAccessKey, o.SESFrom, o.SESTo, o.SESSubjectPrefix, o.SESNotifyOnFailureOnly, o.NotificationAllowInsecureHTTPInDevelopment); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func FlagDocumentation() toolconfig.CommandDoc {
	flags := []toolconfig.FlagDoc{
		flagDefs.host.Doc(envPrefix), flagDefs.port.Doc(envPrefix), flagDefs.user.Doc(envPrefix),
		flagDefs.database.Doc(envPrefix), flagDefs.sslMode.Doc(envPrefix), flagDefs.uri.Doc(envPrefix), flagDefs.password.Doc(envPrefix),
	}
	flags = append(flags, toolconfig.StorageFlagDocs(envPrefix)...)
	flags = append(flags,
		flagDefs.expiryDays.Doc(envPrefix),
		flagDefs.rocketURL.Doc(envPrefix),
		flagDefs.rocketPrefix.Doc(envPrefix),
		flagDefs.rocketFailureOnly.Doc(envPrefix),
		flagDefs.slackURL.Doc(envPrefix),
		flagDefs.slackPrefix.Doc(envPrefix),
		flagDefs.slackFailureOnly.Doc(envPrefix),
		flagDefs.smtpHost.Doc(envPrefix),
		flagDefs.smtpPort.Doc(envPrefix),
		flagDefs.smtpUsername.Doc(envPrefix),
		flagDefs.smtpPassword.Doc(envPrefix),
		flagDefs.smtpFrom.Doc(envPrefix),
		flagDefs.smtpTo.Doc(envPrefix),
		flagDefs.smtpPrefix.Doc(envPrefix),
		flagDefs.smtpFailureOnly.Doc(envPrefix),
		flagDefs.smtpAllowInsecure.Doc(envPrefix),
		flagDefs.sesEndpoint.Doc(envPrefix),
		flagDefs.sesRegion.Doc(envPrefix),
		flagDefs.sesAccessKey.Doc(envPrefix),
		flagDefs.sesSecret.Doc(envPrefix),
		flagDefs.sesFrom.Doc(envPrefix),
		flagDefs.sesTo.Doc(envPrefix),
		flagDefs.sesPrefix.Doc(envPrefix),
		flagDefs.sesFailureOnly.Doc(envPrefix),
		flagDefs.allowInsecureHTTP.Doc(envPrefix),
		flagDefs.cron.Doc(envPrefix),
		flagDefs.cronExpression.Doc(envPrefix),
		flagDefs.tz.Doc(envPrefix),
		flagDefs.keep.Doc(envPrefix),
		flagDefs.version.Doc(envPrefix),
	)

	return toolconfig.CommandDoc{
		Name:  "postgres-archive",
		Flags: flags,
		EnvVars: []toolconfig.EnvDoc{
			{EnvVar: envPrefix + "DUMP_PATH", Description: "Base directory for private per-run PostgreSQL archive workspaces"},
			{EnvVar: envPrefix + "STORAGE_OPERATION_TIMEOUT", Description: "Optional timeout for storage upload and retention operations"},
			{EnvVar: envPrefix + "NOTIFICATION_TIMEOUT", Description: "Optional timeout applied to outbound notification sends"},
		},
	}
}
