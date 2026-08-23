package mongoarchive

import (
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/notification"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
)

const (
	envPrefix           = "MONGOARCHIVE__"
	fallbackEnvPrefix   = "MONGO__"
	defaultCronExpr     = "0 2 * * *"
	defaultWorkspaceDir = "mongoarchive"
)

type Config struct {
	toolconfig.MongoOptions
	toolconfig.StorageOptions
	ArchiveQueryOptions
	RetentionOptions
	NotificationOptions
	ScheduleOptions
	RuntimeOptions
	Keep bool
}

type ArchiveQueryOptions struct {
	Query          string
	QueryFile      string
	ReadPreference string
	ForceTableScan bool
}

type RetentionOptions struct {
	ExpiryDays int
}

type NotificationOptions struct {
	RocketChatWebhookURL                       string
	RocketChatWebhookPrefix                    string
	RocketChatNotifyOnFailureOnly              bool
	SlackWebhookURL                            string
	SlackWebhookPrefix                         string
	SlackNotifyOnFailureOnly                   bool
	SMTPHost                                   string
	SMTPPort                                   string
	SMTPUsername                               string
	SMTPPassword                               string
	SMTPFrom                                   string
	SMTPTo                                     string
	SMTPSubjectPrefix                          string
	SMTPNotifyOnFailureOnly                    bool
	SMTPAllowInsecureNoTLSInDevelopment        bool
	SESEndpoint                                string
	SESRegion                                  string
	SESAccessKeyID                             string
	SESSecretAccessKey                         string
	SESFrom                                    string
	SESTo                                      string
	SESSubjectPrefix                           string
	SESNotifyOnFailureOnly                     bool
	NotificationAllowInsecureHTTPInDevelopment bool
}

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

var archiveFlagDefs = struct {
	query                                      toolconfig.StringFlagDef
	queryFile                                  toolconfig.StringFlagDef
	readPreference                             toolconfig.StringFlagDef
	forceTableScan                             toolconfig.BoolFlagDef
	expiryDays                                 toolconfig.StringFlagDef
	rocketChatWebhookURL                       toolconfig.StringFlagDef
	rocketChatWebhookPrefix                    toolconfig.StringFlagDef
	rocketChatNotifyOnFailureOnly              toolconfig.BoolFlagDef
	slackWebhookURL                            toolconfig.StringFlagDef
	slackWebhookPrefix                         toolconfig.StringFlagDef
	slackNotifyOnFailureOnly                   toolconfig.BoolFlagDef
	smtpHost                                   toolconfig.StringFlagDef
	smtpPort                                   toolconfig.StringFlagDef
	smtpUsername                               toolconfig.StringFlagDef
	smtpPassword                               toolconfig.StringFlagDef
	smtpFrom                                   toolconfig.StringFlagDef
	smtpTo                                     toolconfig.StringFlagDef
	smtpSubjectPrefix                          toolconfig.StringFlagDef
	smtpNotifyOnFailureOnly                    toolconfig.BoolFlagDef
	smtpAllowInsecureNoTLSInDevelopment        toolconfig.BoolFlagDef
	sesEndpoint                                toolconfig.StringFlagDef
	sesRegion                                  toolconfig.StringFlagDef
	sesAccessKeyID                             toolconfig.StringFlagDef
	sesSecretAccessKey                         toolconfig.StringFlagDef
	sesFrom                                    toolconfig.StringFlagDef
	sesTo                                      toolconfig.StringFlagDef
	sesSubjectPrefix                           toolconfig.StringFlagDef
	sesNotifyOnFailureOnly                     toolconfig.BoolFlagDef
	notificationAllowInsecureHTTPInDevelopment toolconfig.BoolFlagDef
	cron                                       toolconfig.BoolFlagDef
	cronExpression                             toolconfig.StringFlagDef
	tz                                         toolconfig.StringFlagDef
	keep                                       toolconfig.BoolFlagDef
	version                                    toolconfig.BoolFlagDef
}{
	query:                               toolconfig.StringFlagDef{Name: "query", EnvKey: "QUERY", Usage: "query filter, as a v2 Extended JSON string"},
	queryFile:                           toolconfig.StringFlagDef{Name: "query-file", EnvKey: "QUERY_FILE", Usage: "path to a file containing a query filter (v2 Extended JSON)"},
	readPreference:                      toolconfig.StringFlagDef{Name: "read-preference", EnvKey: "READ_PREFERENCE", Usage: "specify either a preference mode (e.g. 'nearest') or a preference json object"},
	forceTableScan:                      toolconfig.BoolFlagDef{Name: "force-table-scan", EnvKey: "FORCE_TABLE_SCAN", Usage: "force a table scan"},
	expiryDays:                          toolconfig.StringFlagDef{Name: "expiry-days", EnvKey: "EXPIRY_DAYS", Usage: "The maximum age, in days, for archives to be retained"},
	rocketChatWebhookURL:                toolconfig.StringFlagDef{Name: "rocketchat-webhook-url", EnvKey: "ROCKETCHAT_WEBHOOK_URL", Usage: "Rocket Chat Webhook URL"},
	rocketChatWebhookPrefix:             toolconfig.StringFlagDef{Name: "rocketchat-webhook-prefix", EnvKey: "ROCKETCHAT_WEBHOOK_PREFIX", Usage: "Rocket Chat Webhook Prefix"},
	rocketChatNotifyOnFailureOnly:       toolconfig.BoolFlagDef{Name: "rocketchat-notify-on-failure-only", EnvKey: "ROCKETCHAT_NOTIFY_ON_FAILURE_ONLY", Usage: "Send Rocket Chat notifications only when something goes wrong during the execution"},
	slackWebhookURL:                     toolconfig.StringFlagDef{Name: "slack-webhook-url", EnvKey: "SLACK_WEBHOOK_URL", Usage: "Slack webhook URL"},
	slackWebhookPrefix:                  toolconfig.StringFlagDef{Name: "slack-webhook-prefix", EnvKey: "SLACK_WEBHOOK_PREFIX", Usage: "Slack message prefix"},
	slackNotifyOnFailureOnly:            toolconfig.BoolFlagDef{Name: "slack-notify-on-failure-only", EnvKey: "SLACK_NOTIFY_ON_FAILURE_ONLY", Usage: "Send Slack notifications only when something goes wrong during the execution"},
	smtpHost:                            toolconfig.StringFlagDef{Name: "smtp-host", EnvKey: "SMTP_HOST", Usage: "SMTP server host"},
	smtpPort:                            toolconfig.StringFlagDef{Name: "smtp-port", EnvKey: "SMTP_PORT", Usage: "SMTP server port", Defaults: []string{"587"}},
	smtpUsername:                        toolconfig.StringFlagDef{Name: "smtp-username", EnvKey: "SMTP_USERNAME", Usage: "SMTP username"},
	smtpPassword:                        toolconfig.StringFlagDef{Name: "smtp-password", EnvKey: "SMTP_PASSWORD", Usage: "SMTP password"},
	smtpFrom:                            toolconfig.StringFlagDef{Name: "smtp-from", EnvKey: "SMTP_FROM", Usage: "SMTP from address"},
	smtpTo:                              toolconfig.StringFlagDef{Name: "smtp-to", EnvKey: "SMTP_TO", Usage: "Comma-separated SMTP recipient addresses"},
	smtpSubjectPrefix:                   toolconfig.StringFlagDef{Name: "smtp-subject-prefix", EnvKey: "SMTP_SUBJECT_PREFIX", Usage: "SMTP email subject prefix"},
	smtpNotifyOnFailureOnly:             toolconfig.BoolFlagDef{Name: "smtp-notify-on-failure-only", EnvKey: "SMTP_NOTIFY_ON_FAILURE_ONLY", Usage: "Send SMTP notifications only when something goes wrong during the execution"},
	smtpAllowInsecureNoTLSInDevelopment: toolconfig.BoolFlagDef{Name: "smtp-allow-insecure-no-tls-in-development", EnvKey: "SMTP_ALLOW_INSECURE_NO_TLS_IN_DEVELOPMENT", Usage: "Allow SMTP without STARTTLS only for local development or emulator use"},
	sesEndpoint:                         toolconfig.StringFlagDef{Name: "ses-endpoint", EnvKey: "SES_ENDPOINT", Usage: "AWS SES endpoint override"},
	sesRegion:                           toolconfig.StringFlagDef{Name: "ses-region", EnvKey: "SES_REGION", Usage: "AWS SES region"},
	sesAccessKeyID:                      toolconfig.StringFlagDef{Name: "ses-access-key-id", EnvKey: "SES_ACCESS_KEY_ID", Usage: "AWS SES access key ID"},
	sesSecretAccessKey:                  toolconfig.StringFlagDef{Name: "ses-secret-access-key", EnvKey: "SES_SECRET_ACCESS_KEY", Usage: "AWS SES secret access key"},
	sesFrom:                             toolconfig.StringFlagDef{Name: "ses-from", EnvKey: "SES_FROM", Usage: "AWS SES sender address"},
	sesTo:                               toolconfig.StringFlagDef{Name: "ses-to", EnvKey: "SES_TO", Usage: "Comma-separated AWS SES recipient addresses"},
	sesSubjectPrefix:                    toolconfig.StringFlagDef{Name: "ses-subject-prefix", EnvKey: "SES_SUBJECT_PREFIX", Usage: "AWS SES email subject prefix"},
	sesNotifyOnFailureOnly:              toolconfig.BoolFlagDef{Name: "ses-notify-on-failure-only", EnvKey: "SES_NOTIFY_ON_FAILURE_ONLY", Usage: "Send AWS SES notifications only when something goes wrong during the execution"},
	notificationAllowInsecureHTTPInDevelopment: toolconfig.BoolFlagDef{Name: "notification-allow-insecure-http-in-development", EnvKey: "NOTIFICATION_ALLOW_INSECURE_HTTP_IN_DEVELOPMENT", Usage: "Allow HTTP notification webhooks or endpoint overrides only for local development or emulator use"},
	cron:           toolconfig.BoolFlagDef{Name: "cron", EnvKey: "CRON", Usage: "run a cron schedular and block current execution path"},
	cronExpression: toolconfig.StringFlagDef{Name: "cron-expression", EnvKey: "CRON_EXPRESSION", Usage: "a string describes individual details of the cron schedule"},
	tz:             toolconfig.StringFlagDef{Name: "tz", EnvKey: "TZ", Usage: "user-specified time zone"},
	keep:           toolconfig.BoolFlagDef{Name: "keep", EnvKey: "KEEP", Usage: "keep data dump"},
	version:        toolconfig.BoolFlagDef{Name: "version", Usage: "Show the version"},
}

func ParseFlags() (*Config, bool, error) {
	env := utils.NewEnv(envPrefix, fallbackEnvPrefix, "")
	return parseFlags(flag.CommandLine, env, os.Args[1:])
}

func parseFlags(flagSet *flag.FlagSet, env toolconfig.EnvReader, args []string) (*Config, bool, error) {
	cfg := &Config{}

	mongoBindings := toolconfig.BindMongoFlags(flagSet, env)
	query := archiveFlagDefs.query.Bind(flagSet, env)
	queryFile := archiveFlagDefs.queryFile.Bind(flagSet, env)
	readPreference := archiveFlagDefs.readPreference.Bind(flagSet, env)
	forceTableScan := archiveFlagDefs.forceTableScan.Bind(flagSet, env)
	storageBindings := toolconfig.BindStorageFlags(flagSet, env)
	expiryDays := archiveFlagDefs.expiryDays.Bind(flagSet, env)
	rocketChatWebhookURL := archiveFlagDefs.rocketChatWebhookURL.Bind(flagSet, env)
	rocketChatWebhookPrefix := archiveFlagDefs.rocketChatWebhookPrefix.Bind(flagSet, env)
	rocketChatNotifyOnFailureOnly := archiveFlagDefs.rocketChatNotifyOnFailureOnly.Bind(flagSet, env)
	slackWebhookURL := archiveFlagDefs.slackWebhookURL.Bind(flagSet, env)
	slackWebhookPrefix := archiveFlagDefs.slackWebhookPrefix.Bind(flagSet, env)
	slackNotifyOnFailureOnly := archiveFlagDefs.slackNotifyOnFailureOnly.Bind(flagSet, env)
	smtpHost := archiveFlagDefs.smtpHost.Bind(flagSet, env)
	smtpPort := archiveFlagDefs.smtpPort.Bind(flagSet, env)
	smtpUsername := archiveFlagDefs.smtpUsername.Bind(flagSet, env)
	smtpPassword := archiveFlagDefs.smtpPassword.Bind(flagSet, env)
	smtpFrom := archiveFlagDefs.smtpFrom.Bind(flagSet, env)
	smtpTo := archiveFlagDefs.smtpTo.Bind(flagSet, env)
	smtpSubjectPrefix := archiveFlagDefs.smtpSubjectPrefix.Bind(flagSet, env)
	smtpNotifyOnFailureOnly := archiveFlagDefs.smtpNotifyOnFailureOnly.Bind(flagSet, env)
	smtpAllowInsecureNoTLSInDevelopment := archiveFlagDefs.smtpAllowInsecureNoTLSInDevelopment.Bind(flagSet, env)
	sesEndpoint := archiveFlagDefs.sesEndpoint.Bind(flagSet, env)
	sesRegion := flagSet.String(archiveFlagDefs.sesRegion.Name, env.GetValue("SES_REGION", env.GetValue("AWS_REGION")), archiveFlagDefs.sesRegion.Usage)
	sesAccessKeyID := flagSet.String(archiveFlagDefs.sesAccessKeyID.Name, env.GetValue("SES_ACCESS_KEY_ID", env.GetValue("AWS_ACCESS_KEY_ID")), archiveFlagDefs.sesAccessKeyID.Usage)
	sesSecretAccessKey := flagSet.String(archiveFlagDefs.sesSecretAccessKey.Name, env.GetValue("SES_SECRET_ACCESS_KEY", env.GetValue("AWS_SECRET_ACCESS_KEY")), archiveFlagDefs.sesSecretAccessKey.Usage)
	sesFrom := archiveFlagDefs.sesFrom.Bind(flagSet, env)
	sesTo := archiveFlagDefs.sesTo.Bind(flagSet, env)
	sesSubjectPrefix := archiveFlagDefs.sesSubjectPrefix.Bind(flagSet, env)
	sesNotifyOnFailureOnly := archiveFlagDefs.sesNotifyOnFailureOnly.Bind(flagSet, env)
	notificationAllowInsecureHTTPInDevelopment := archiveFlagDefs.notificationAllowInsecureHTTPInDevelopment.Bind(flagSet, env)
	cron := archiveFlagDefs.cron.Bind(flagSet, env)
	cronExpression := archiveFlagDefs.cronExpression.Bind(flagSet, env)
	tz := archiveFlagDefs.tz.Bind(flagSet, env)
	keep := archiveFlagDefs.keep.Bind(flagSet, env)
	showVersion := archiveFlagDefs.version.Bind(flagSet, env)

	if err := flagSet.Parse(args); err != nil {
		return nil, false, err
	}

	mongoBindings.Apply(&cfg.MongoOptions)
	cfg.ArchiveQueryOptions = ArchiveQueryOptions{
		Query:          *query,
		QueryFile:      *queryFile,
		ReadPreference: *readPreference,
		ForceTableScan: *forceTableScan,
	}
	storageBindings.Apply(&cfg.StorageOptions)
	parsedExpiryDays, err := parseExpiryDays(*expiryDays)
	if err != nil {
		return nil, false, err
	}
	parsedLocation, err := parseLocation(*tz)
	if err != nil {
		return nil, false, err
	}
	cfg.RetentionOptions = RetentionOptions{ExpiryDays: parsedExpiryDays}
	cfg.NotificationOptions = NotificationOptions{
		RocketChatWebhookURL:                *rocketChatWebhookURL,
		RocketChatWebhookPrefix:             *rocketChatWebhookPrefix,
		RocketChatNotifyOnFailureOnly:       *rocketChatNotifyOnFailureOnly,
		SlackWebhookURL:                     *slackWebhookURL,
		SlackWebhookPrefix:                  *slackWebhookPrefix,
		SlackNotifyOnFailureOnly:            *slackNotifyOnFailureOnly,
		SMTPHost:                            *smtpHost,
		SMTPPort:                            *smtpPort,
		SMTPUsername:                        *smtpUsername,
		SMTPPassword:                        *smtpPassword,
		SMTPFrom:                            *smtpFrom,
		SMTPTo:                              *smtpTo,
		SMTPSubjectPrefix:                   *smtpSubjectPrefix,
		SMTPNotifyOnFailureOnly:             *smtpNotifyOnFailureOnly,
		SMTPAllowInsecureNoTLSInDevelopment: *smtpAllowInsecureNoTLSInDevelopment,
		SESEndpoint:                         *sesEndpoint,
		SESRegion:                           *sesRegion,
		SESAccessKeyID:                      *sesAccessKeyID,
		SESSecretAccessKey:                  *sesSecretAccessKey,
		SESFrom:                             *sesFrom,
		SESTo:                               *sesTo,
		SESSubjectPrefix:                    *sesSubjectPrefix,
		SESNotifyOnFailureOnly:              *sesNotifyOnFailureOnly,
		NotificationAllowInsecureHTTPInDevelopment: *notificationAllowInsecureHTTPInDevelopment,
	}
	cfg.ScheduleOptions = ScheduleOptions{
		Cron:           *cron,
		CronExpression: parseCronExpression(*cronExpression),
		Location:       parsedLocation,
	}
	cfg.Keep = *keep

	if showVersion != nil && *showVersion {
		return cfg, true, nil
	}

	runtimeOptions, err := parseRuntimeOptions(env)
	if err != nil {
		return nil, false, err
	}
	cfg.RuntimeOptions = runtimeOptions

	if err := cfg.Validate(); err != nil {
		return nil, false, err
	}

	return cfg, false, nil
}

func parseLocation(tz string) (*time.Location, error) {
	if tz == "" {
		return time.Local, nil
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, err
	}
	return loc, nil
}

func parseExpiryDays(raw string) (int, error) {
	if raw == "" {
		mlog.Logvf(mlog.Always, "Backup does not expire")
		return 0, nil
	}

	expiryDays, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New("expiry-days must be a non-negative integer")
	}
	if expiryDays < 0 {
		return 0, errors.New("expiry-days must be a non-negative integer")
	}
	mlog.Logvf(mlog.Always, "Backup expiration: %v days", expiryDays)

	return expiryDays, nil
}

func parseCronExpression(raw string) string {
	if raw != "" {
		return raw
	}

	return defaultCronExpr
}

func parseRuntimeOptions(env toolconfig.EnvReader) (RuntimeOptions, error) {
	storageOperationTimeout, err := toolconfig.ReadOptionalDuration(env, "STORAGE_OPERATION_TIMEOUT")
	if err != nil {
		return RuntimeOptions{}, err
	}
	notificationTimeout, err := toolconfig.ReadOptionalDuration(env, "NOTIFICATION_TIMEOUT")
	if err != nil {
		return RuntimeOptions{}, err
	}

	return RuntimeOptions{
		WorkspaceBasePath:       toolconfig.ReadWorkspaceBase(env, "DUMP_PATH", filepath.Join(os.TempDir(), defaultWorkspaceDir)),
		StorageOperationTimeout: storageOperationTimeout,
		NotificationTimeout:     notificationTimeout,
	}, nil
}

func (c *Config) GetTZ() *time.Location {
	return c.Location
}

func (c *Config) GetMongodumpOptions() []string {
	options := c.MongoOptions.AppendToolOptions([]string{"--gzip"})
	if c.Query != "" {
		options = append(options, "--query="+c.Query)
	}
	if c.QueryFile != "" {
		options = append(options, "--queryFile="+c.QueryFile)
	}
	if c.ReadPreference != "" {
		options = append(options, "--readPreference="+c.ReadPreference)
	}
	if c.ForceTableScan {
		options = append(options, "--forceTableScan")
	}

	return options
}

func (c *Config) GetStorages(ctx context.Context) ([]storage.ArchiveBackend, error) {
	return c.StorageOptions.GetArchiveStorages(ctx, c.ExpiryDays)
}

func (c *Config) Validate() error {
	if err := c.RuntimeOptions.Validate(); err != nil {
		return err
	}
	if err := c.StorageOptions.Validate(); err != nil {
		return err
	}
	return c.NotificationOptions.Validate()
}

func (r RuntimeOptions) Validate() error {
	if r.StorageOperationTimeout < 0 {
		return errors.New("STORAGE_OPERATION_TIMEOUT must be greater than zero")
	}
	if r.NotificationTimeout < 0 {
		return errors.New("NOTIFICATION_TIMEOUT must be greater than zero")
	}
	return nil
}

func (o NotificationOptions) Validate() error {
	if o.useRocketChat() {
		if err := notification.ValidateWebhookURL(o.RocketChatWebhookURL, o.NotificationAllowInsecureHTTPInDevelopment, "Rocket.Chat webhook URL"); err != nil {
			return err
		}
	}
	if o.useSlack() {
		if err := notification.ValidateWebhookURL(o.SlackWebhookURL, o.NotificationAllowInsecureHTTPInDevelopment, "Slack webhook URL"); err != nil {
			return err
		}
	}
	if o.useSMTP() {
		if _, err := notification.ValidateSMTPOptions(o.SMTPHost, o.SMTPPort, o.SMTPUsername, o.SMTPPassword, o.SMTPFrom, o.SMTPTo, o.SMTPSubjectPrefix); err != nil {
			return err
		}
	}
	if o.useSES() {
		if _, err := notification.ValidateSESOptions(o.SESEndpoint, o.SESRegion, o.SESAccessKeyID, o.SESSecretAccessKey, o.SESFrom, o.SESTo, o.NotificationAllowInsecureHTTPInDevelopment); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) getRocketChat() (*notification.RocketChat, error) {
	rc := new(notification.RocketChat)
	err := rc.Init(c.RocketChatWebhookURL, c.RocketChatWebhookPrefix, c.RocketChatNotifyOnFailureOnly, c.NotificationAllowInsecureHTTPInDevelopment)
	return rc, err
}

func (c *Config) getSlack() (*notification.Slack, error) {
	slack := new(notification.Slack)
	err := slack.Init(c.SlackWebhookURL, c.SlackWebhookPrefix, c.SlackNotifyOnFailureOnly, c.NotificationAllowInsecureHTTPInDevelopment)
	return slack, err
}

func (c *Config) getSMTP() (*notification.SMTP, error) {
	smtpNotification := new(notification.SMTP)
	err := smtpNotification.Init(c.SMTPHost, c.SMTPPort, c.SMTPUsername, c.SMTPPassword, c.SMTPFrom, c.SMTPTo, c.SMTPSubjectPrefix, c.SMTPNotifyOnFailureOnly, c.SMTPAllowInsecureNoTLSInDevelopment)
	return smtpNotification, err
}

func (c *Config) getSES() (*notification.SES, error) {
	sesNotification := new(notification.SES)
	err := sesNotification.Init(c.SESEndpoint, c.SESRegion, c.SESAccessKeyID, c.SESSecretAccessKey, c.SESFrom, c.SESTo, c.SESSubjectPrefix, c.SESNotifyOnFailureOnly, c.NotificationAllowInsecureHTTPInDevelopment)
	return sesNotification, err
}

// GetNotifications constructs concrete notifier transports and SDK clients.
// Startup validation must use NotificationOptions.Validate instead.
func (c *Config) GetNotifications() ([]notification.Notification, error) {
	notifications := make([]notification.Notification, 0)

	if c.NotificationOptions.useRocketChat() {
		rc, err := c.getRocketChat()
		if err != nil {
			return nil, err
		} else if rc != nil {
			mlog.Logvf(mlog.Always, "Found Notification Option: %v", "RocketChat")
			notifications = append(notifications, rc)
		}
	}

	if c.NotificationOptions.useSlack() {
		slack, err := c.getSlack()
		if err != nil {
			return nil, err
		} else if slack != nil {
			mlog.Logvf(mlog.Always, "Found Notification Option: %v", "Slack")
			notifications = append(notifications, slack)
		}
	}

	if c.NotificationOptions.useSMTP() {
		smtpNotification, err := c.getSMTP()
		if err != nil {
			return nil, err
		} else if smtpNotification != nil {
			mlog.Logvf(mlog.Always, "Found Notification Option: %v", "SMTP")
			notifications = append(notifications, smtpNotification)
		}
	}

	if c.NotificationOptions.useSES() {
		sesNotification, err := c.getSES()
		if err != nil {
			return nil, err
		} else if sesNotification != nil {
			mlog.Logvf(mlog.Always, "Found Notification Option: %v", "SES")
			notifications = append(notifications, sesNotification)
		}
	}

	return notifications, nil
}

func (c *Config) GetLocation() *time.Location {
	return c.Location
}

func (c *Config) GetCronExpression() string {
	if c.CronExpression == "" {
		return defaultCronExpr
	}

	return c.CronExpression
}

func (c *Config) HasCron() bool {
	return c.Cron
}

func (c *Config) HasKeep() bool {
	return c.Keep
}

func FlagDocumentation() toolconfig.CommandDoc {
	flags := append([]toolconfig.FlagDoc{}, toolconfig.MongoFlagDocs(envPrefix)...)
	flags = append(flags,
		archiveFlagDefs.query.Doc(envPrefix),
		archiveFlagDefs.queryFile.Doc(envPrefix),
		archiveFlagDefs.readPreference.Doc(envPrefix),
		archiveFlagDefs.forceTableScan.Doc(envPrefix),
	)
	flags = append(flags, toolconfig.StorageFlagDocs(envPrefix)...)
	flags = append(flags,
		archiveFlagDefs.expiryDays.Doc(envPrefix),
		archiveFlagDefs.rocketChatWebhookURL.Doc(envPrefix),
		archiveFlagDefs.rocketChatWebhookPrefix.Doc(envPrefix),
		archiveFlagDefs.rocketChatNotifyOnFailureOnly.Doc(envPrefix),
		archiveFlagDefs.slackWebhookURL.Doc(envPrefix),
		archiveFlagDefs.slackWebhookPrefix.Doc(envPrefix),
		archiveFlagDefs.slackNotifyOnFailureOnly.Doc(envPrefix),
		archiveFlagDefs.smtpHost.Doc(envPrefix),
		archiveFlagDefs.smtpPort.Doc(envPrefix),
		archiveFlagDefs.smtpUsername.Doc(envPrefix),
		archiveFlagDefs.smtpPassword.Doc(envPrefix),
		archiveFlagDefs.smtpFrom.Doc(envPrefix),
		archiveFlagDefs.smtpTo.Doc(envPrefix),
		archiveFlagDefs.smtpSubjectPrefix.Doc(envPrefix),
		archiveFlagDefs.smtpNotifyOnFailureOnly.Doc(envPrefix),
		archiveFlagDefs.smtpAllowInsecureNoTLSInDevelopment.Doc(envPrefix),
		archiveFlagDefs.sesEndpoint.Doc(envPrefix),
		archiveFlagDefs.sesRegion.Doc(envPrefix),
		archiveFlagDefs.sesAccessKeyID.Doc(envPrefix),
		archiveFlagDefs.sesSecretAccessKey.Doc(envPrefix),
		archiveFlagDefs.sesFrom.Doc(envPrefix),
		archiveFlagDefs.sesTo.Doc(envPrefix),
		archiveFlagDefs.sesSubjectPrefix.Doc(envPrefix),
		archiveFlagDefs.sesNotifyOnFailureOnly.Doc(envPrefix),
		archiveFlagDefs.notificationAllowInsecureHTTPInDevelopment.Doc(envPrefix),
		archiveFlagDefs.cron.Doc(envPrefix),
		archiveFlagDefs.cronExpression.Doc(envPrefix),
		archiveFlagDefs.tz.Doc(envPrefix),
		archiveFlagDefs.keep.Doc(envPrefix),
		archiveFlagDefs.version.Doc(envPrefix),
	)

	return toolconfig.CommandDoc{
		Name:  "mongo-archive",
		Flags: flags,
		EnvVars: []toolconfig.EnvDoc{
			{EnvVar: envPrefix + "DUMP_PATH", Description: "Base directory for per-run dump workspaces before uploads"},
			{EnvVar: envPrefix + "STORAGE_OPERATION_TIMEOUT", Description: "Optional timeout applied to storage lookup, upload, and retention operations"},
			{EnvVar: envPrefix + "NOTIFICATION_TIMEOUT", Description: "Optional timeout applied to outbound notification sends"},
		},
	}
}

func (o NotificationOptions) useRocketChat() bool {
	return o.RocketChatWebhookURL != ""
}

func (o NotificationOptions) useSlack() bool {
	return o.SlackWebhookURL != ""
}

func (o NotificationOptions) useSMTP() bool {
	return o.SMTPHost != "" || o.SMTPFrom != "" || o.SMTPTo != ""
}

func (o NotificationOptions) useSES() bool {
	return o.SESFrom != "" || o.SESTo != ""
}
