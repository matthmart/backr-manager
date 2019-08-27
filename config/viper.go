package config

import (
	"github.com/agence-webup/backr/manager"
	"github.com/spf13/viper"
)

var config manager.Config

// Get returns the config
func Get() manager.Config {
	return config
}

// SetupFromViper get config from viper
func SetupFromViper() {
	c := manager.Config{
		S3: manager.S3Config{
			Bucket:    viper.GetString("s3.bucket"),
			Endpoint:  viper.GetString("s3.endpoint"),
			AccessKey: viper.GetString("s3.access_key"),
			SecretKey: viper.GetString("s3.secret_key"),
			UseTLS:    viper.GetBool("s3.use_tls"),
		},
		Bolt: manager.BoltConfig{
			Filepath: viper.GetString("bolt.filepath"),
		},
		API: manager.APIConfig{
			ListenIP:   viper.GetString("api.listen_ip"),
			ListenPort: viper.GetString("api.listen_port"),
			JWTSecret:  viper.GetString("api.jwt_secret"),
		},
	}

	config = c
}
