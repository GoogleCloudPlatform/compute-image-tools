package config

import (
	"fmt"
	"time"
)

var (
	// TODO: allow this to be configurable through flag to test against staging
	prodEndpoint           = "osconfig.googleapis.com:443"
	oauthDefault           = ""
	bucketDefault          = "osconfig-agent-end2end-tests"
	logsPathDefault        = "logs"
	logPushIntervalDefault = 3 * time.Second
)

// SvcEndpoint returns the svcEndpoint
func SvcEndpoint() string {
	return prodEndpoint
}

// OauthPath returns the oauthPath file path
func OauthPath() string {
	return oauthDefault
}

// LogBucket returns the oauthPath file path
func LogBucket() string {
	return bucketDefault
}

// LogsPath returns the oauthPath file path
func LogsPath() string {
	return fmt.Sprintf("%s-%s", logsPathDefault, time.Now().Format("2006-01-02-15:04:05"))
}

// LogPushInterval returns the interval at which the serial console logs are written to GCS
func LogPushInterval() time.Duration {
	return logPushIntervalDefault
}
