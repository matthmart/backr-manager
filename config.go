package manager

// Config stores configuration used by the manager
type Config struct {
	S3            S3Config
	Bolt          BoltConfig
	API           APIConfig
	SlackNotifier SlackNotifierConfig
}

// S3Config stores S3-like API configuration
type S3Config struct {
	Bucket    string
	Endpoint  string
	AccessKey string
	SecretKey string
	Region    string
	UseTLS    bool
}

// BoltConfig stores settings required to setup BoltDB
type BoltConfig struct {
	Filepath string
}

// APIConfig stores settings to configure the API
type APIConfig struct {
	ListenIP   string
	ListenPort string
	JWTSecret  string
}

// SlackNotifierConfig stores settings to configure Slack notifier
type SlackNotifierConfig struct {
	WebhookURL string
}
