package manager

// Config stores configuration used by the manager
type Config struct {
	S3 S3Config
}

// S3Config stores S3-like API configuration
type S3Config struct {
	Bucket    string
	Endpoint  string
	AccessKey string
	SecretKey string
	UseTLS    bool
}
