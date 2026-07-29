package mongoarchive

import (
	"flag"
	"strconv"
	"time"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/notification"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
)

const (
	envPrefix         = "MONGOARCHIVE__"
	fallbackEnvPrefix = "MONGO__"
	defaultCronExpr   = "0 2 * * *"
)

type Config struct {
	toolconfig.MongoOptions
	toolconfig.StorageOptions
	Query                         string
	QueryFile                     string
	ReadPreference                string
	ForceTableScan                bool
	ExpiryDays                    int
	RocketChatWebhookURL          string
	RocketChatWebhookPrefix       string
	RocketChatNotifyOnFailureOnly bool
	SlackWebhookURL               string
	SlackWebhookPrefix            string
	SlackNotifyOnFailureOnly      bool
	SMTPHost                      string
	SMTPPort                      string
	SMTPUsername                  string
	SMTPPassword                  string
	SMTPFrom                      string
	SMTPTo                        string
	SMTPSubjectPrefix             string
	SMTPNotifyOnFailureOnly       bool
	SESEndpoint                   string
	SESRegion                     string
	SESAccessKeyID                string
	SESSecretAccessKey            string
	SESFrom                       string
	SESTo                         string
	SESSubjectPrefix              string
	SESNotifyOnFailureOnly        bool
	Cron                          bool
	CronExpression                string
	Location                      *time.Location
	Keep                          bool
}

func ParseFlags() (*Config, bool) {
	env := utils.NewEnv(envPrefix, fallbackEnvPrefix, "")
	cfg := &Config{}

	mongoBindings := toolconfig.BindMongoFlags(env)
	query := flag.String("query", env.GetValue("QUERY"), "query filter, as a v2 Extended JSON string")
	queryFile := flag.String("query-file", env.GetValue("QUERY_FILE"), "path to a file containing a query filter (v2 Extended JSON)")
	readPreference := flag.String("read-preference", env.GetValue("READ_PREFERENCE"), "specify either a preference mode (e.g. 'nearest') or a preference json object")
	forceTableScan := flag.Bool("force-table-scan", env.GetValue("FORCE_TABLE_SCAN") == "true", "force a table scan")
	storageBindings := toolconfig.BindStorageFlags(env)
	expiryDays := flag.String("expiry-days", env.GetValue("EXPIRY_DAYS"), "The maximum age, in days, for archives to be retained")
	rocketChatWebhookURL := flag.String("rocketchat-webhook-url", env.GetValue("ROCKETCHAT_WEBHOOK_URL"), "Rocket Chat Webhook URL")
	rocketChatWebhookPrefix := flag.String("rocketchat-webhook-prefix", env.GetValue("ROCKETCHAT_WEBHOOK_PREFIX"), "Rocket Chat Webhook Prefix")
	rocketChatNotifyOnFailureOnly := flag.Bool("rocketchat-notify-on-failure-only", env.GetValue("ROCKETCHAT_NOTIFY_ON_FAILURE_ONLY") == "true", "Send Rocket Chat notifications only when something goes wrong during the execution")
	slackWebhookURL := flag.String("slack-webhook-url", env.GetValue("SLACK_WEBHOOK_URL"), "Slack webhook URL")
	slackWebhookPrefix := flag.String("slack-webhook-prefix", env.GetValue("SLACK_WEBHOOK_PREFIX"), "Slack message prefix")
	slackNotifyOnFailureOnly := flag.Bool("slack-notify-on-failure-only", env.GetValue("SLACK_NOTIFY_ON_FAILURE_ONLY") == "true", "Send Slack notifications only when something goes wrong during the execution")
	smtpHost := flag.String("smtp-host", env.GetValue("SMTP_HOST"), "SMTP server host")
	smtpPort := flag.String("smtp-port", env.GetValue("SMTP_PORT", "587"), "SMTP server port")
	smtpUsername := flag.String("smtp-username", env.GetValue("SMTP_USERNAME"), "SMTP username")
	smtpPassword := flag.String("smtp-password", env.GetValue("SMTP_PASSWORD"), "SMTP password")
	smtpFrom := flag.String("smtp-from", env.GetValue("SMTP_FROM"), "SMTP from address")
	smtpTo := flag.String("smtp-to", env.GetValue("SMTP_TO"), "Comma-separated SMTP recipient addresses")
	smtpSubjectPrefix := flag.String("smtp-subject-prefix", env.GetValue("SMTP_SUBJECT_PREFIX"), "SMTP email subject prefix")
	smtpNotifyOnFailureOnly := flag.Bool("smtp-notify-on-failure-only", env.GetValue("SMTP_NOTIFY_ON_FAILURE_ONLY") == "true", "Send SMTP notifications only when something goes wrong during the execution")
	sesEndpoint := flag.String("ses-endpoint", env.GetValue("SES_ENDPOINT"), "AWS SES endpoint override")
	sesRegion := flag.String("ses-region", env.GetValue("SES_REGION", env.GetValue("AWS_REGION")), "AWS SES region")
	sesAccessKeyID := flag.String("ses-access-key-id", env.GetValue("SES_ACCESS_KEY_ID", env.GetValue("AWS_ACCESS_KEY_ID")), "AWS SES access key ID")
	sesSecretAccessKey := flag.String("ses-secret-access-key", env.GetValue("SES_SECRET_ACCESS_KEY", env.GetValue("AWS_SECRET_ACCESS_KEY")), "AWS SES secret access key")
	sesFrom := flag.String("ses-from", env.GetValue("SES_FROM"), "AWS SES sender address")
	sesTo := flag.String("ses-to", env.GetValue("SES_TO"), "Comma-separated AWS SES recipient addresses")
	sesSubjectPrefix := flag.String("ses-subject-prefix", env.GetValue("SES_SUBJECT_PREFIX"), "AWS SES email subject prefix")
	sesNotifyOnFailureOnly := flag.Bool("ses-notify-on-failure-only", env.GetValue("SES_NOTIFY_ON_FAILURE_ONLY") == "true", "Send AWS SES notifications only when something goes wrong during the execution")
	cron := flag.Bool("cron", env.GetValue("CRON") == "true", "run a cron schedular and block current execution path")
	cronExpression := flag.String("cron-expression", env.GetValue("CRON_EXPRESSION"), "a string describes individual details of the cron schedule")
	tz := flag.String("tz", env.GetValue("TZ"), "user-specified time zone")
	keep := flag.Bool("keep", env.GetValue("KEEP") == "true", "keep data dump")
	showVersion := flag.Bool("version", false, "Show the version")

	flag.Parse()

	mongoBindings.Apply(&cfg.MongoOptions)
	cfg.Query = *query
	cfg.QueryFile = *queryFile
	cfg.ReadPreference = *readPreference
	cfg.ForceTableScan = *forceTableScan
	storageBindings.Apply(&cfg.StorageOptions)
	cfg.ExpiryDays = parseExpiryDays(*expiryDays)
	cfg.RocketChatWebhookURL = *rocketChatWebhookURL
	cfg.RocketChatWebhookPrefix = *rocketChatWebhookPrefix
	cfg.RocketChatNotifyOnFailureOnly = *rocketChatNotifyOnFailureOnly
	cfg.SlackWebhookURL = *slackWebhookURL
	cfg.SlackWebhookPrefix = *slackWebhookPrefix
	cfg.SlackNotifyOnFailureOnly = *slackNotifyOnFailureOnly
	cfg.SMTPHost = *smtpHost
	cfg.SMTPPort = *smtpPort
	cfg.SMTPUsername = *smtpUsername
	cfg.SMTPPassword = *smtpPassword
	cfg.SMTPFrom = *smtpFrom
	cfg.SMTPTo = *smtpTo
	cfg.SMTPSubjectPrefix = *smtpSubjectPrefix
	cfg.SMTPNotifyOnFailureOnly = *smtpNotifyOnFailureOnly
	cfg.SESEndpoint = *sesEndpoint
	cfg.SESRegion = *sesRegion
	cfg.SESAccessKeyID = *sesAccessKeyID
	cfg.SESSecretAccessKey = *sesSecretAccessKey
	cfg.SESFrom = *sesFrom
	cfg.SESTo = *sesTo
	cfg.SESSubjectPrefix = *sesSubjectPrefix
	cfg.SESNotifyOnFailureOnly = *sesNotifyOnFailureOnly
	cfg.Cron = *cron
	cfg.CronExpression = parseCronExpression(*cronExpression)
	cfg.Location = parseLocation(*tz)
	cfg.Keep = *keep

	if showVersion != nil && *showVersion {
		return cfg, true
	}

	return cfg, false
}

func parseLocation(tz string) *time.Location {
	if tz == "" {
		return time.Local
	}

	loc, _ := time.LoadLocation(tz)
	return loc
}

func parseExpiryDays(raw string) int {
	if raw == "" {
		mlog.Logvf(mlog.Always, "Backup does not expire")
		return 0
	}

	if expiryDays, err := strconv.Atoi(raw); err == nil {
		mlog.Logvf(mlog.Always, "Backup expiration: %v days", expiryDays)
		return expiryDays
	}

	return 0
}

func parseCronExpression(raw string) string {
	if raw != "" {
		return raw
	}

	return defaultCronExpr
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

func (c *Config) GetStorages() []storage.Storage {
	return c.StorageOptions.GetStorages(c.ExpiryDays)
}

func (c *Config) getRocketChat() (*notification.RocketChat, error) {
	rc := new(notification.RocketChat)
	err := rc.Init(c.RocketChatWebhookURL, c.RocketChatWebhookPrefix, c.RocketChatNotifyOnFailureOnly)
	return rc, err
}

func (c *Config) getSlack() (*notification.Slack, error) {
	slack := new(notification.Slack)
	err := slack.Init(c.SlackWebhookURL, c.SlackWebhookPrefix, c.SlackNotifyOnFailureOnly)
	return slack, err
}

func (c *Config) getSMTP() (*notification.SMTP, error) {
	smtpNotification := new(notification.SMTP)
	err := smtpNotification.Init(c.SMTPHost, c.SMTPPort, c.SMTPUsername, c.SMTPPassword, c.SMTPFrom, c.SMTPTo, c.SMTPSubjectPrefix, c.SMTPNotifyOnFailureOnly)
	return smtpNotification, err
}

func (c *Config) getSES() (*notification.SES, error) {
	sesNotification := new(notification.SES)
	err := sesNotification.Init(c.SESEndpoint, c.SESRegion, c.SESAccessKeyID, c.SESSecretAccessKey, c.SESFrom, c.SESTo, c.SESSubjectPrefix, c.SESNotifyOnFailureOnly)
	return sesNotification, err
}

func (c *Config) GetNotifications() []notification.Notification {
	notifications := make([]notification.Notification, 0)

	if c.useRocketChat() {
		rc, err := c.getRocketChat()
		if err != nil {
			mlog.Logvf(mlog.Always, "Failed to initialize RocketChat notification: %v", err)
		} else if rc != nil {
			mlog.Logvf(mlog.Always, "Found Notification Option: %v", "RocketChat")
			notifications = append(notifications, rc)
		}
	}

	if c.useSlack() {
		slack, err := c.getSlack()
		if err != nil {
			mlog.Logvf(mlog.Always, "Failed to initialize Slack notification: %v", err)
		} else if slack != nil {
			mlog.Logvf(mlog.Always, "Found Notification Option: %v", "Slack")
			notifications = append(notifications, slack)
		}
	}

	if c.useSMTP() {
		smtpNotification, err := c.getSMTP()
		if err != nil {
			mlog.Logvf(mlog.Always, "Failed to initialize SMTP notification: %v", err)
		} else if smtpNotification != nil {
			mlog.Logvf(mlog.Always, "Found Notification Option: %v", "SMTP")
			notifications = append(notifications, smtpNotification)
		}
	}

	if c.useSES() {
		sesNotification, err := c.getSES()
		if err != nil {
			mlog.Logvf(mlog.Always, "Failed to initialize SES notification: %v", err)
		} else if sesNotification != nil {
			mlog.Logvf(mlog.Always, "Found Notification Option: %v", "SES")
			notifications = append(notifications, sesNotification)
		}
	}

	return notifications
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

func (c *Config) useRocketChat() bool {
	return c.RocketChatWebhookURL != ""
}

func (c *Config) useSlack() bool {
	return c.SlackWebhookURL != ""
}

func (c *Config) useSMTP() bool {
	return c.SMTPHost != "" || c.SMTPFrom != "" || c.SMTPTo != ""
}

func (c *Config) useSES() bool {
	return c.SESFrom != "" || c.SESTo != ""
}
