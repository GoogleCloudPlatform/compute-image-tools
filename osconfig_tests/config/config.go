package config

import (
	"fmt"
	"time"
)

var (
	// TODO: allow this to be configurable through flag to test against staging
	prodEndpoint    = "osconfig.googleapis.com:443"
	oauthDefault    = ""
	bucketDefault   = "osconfig-agent-end2end-tests"
	logsPathDefault = "logs"

	TestConfig *config
)

type config struct {
	svcEndpoint, oauthPath, logsBucket, logsPath string
}

// SetConfig sets the test configs
func SetConfig() error {
	TestConfig = &config{
		svcEndpoint: prodEndpoint,
		oauthPath:   oauthDefault,
		logsBucket:  bucketDefault,
		logsPath:    logsPathDefault,
	}
	TestConfig.logsPath = fmt.Sprintf("%s-%s", TestConfig.logsPath, time.Now().Format("2006-01-02-15:04:05"))
	return nil
}

// SvcEndpoint returns the svcEndpoint
func SvcEndpoint() string {
	return TestConfig.svcEndpoint
}

// OauthPath returns the oauthPath file path
func OauthPath() string {
	return TestConfig.oauthPath
}

// LogBucket returns the oauthPath file path
func LogBucket() string {
	return TestConfig.logsBucket
}

// LogsPath returns the oauthPath file path
func LogsPath() string {
	return TestConfig.logsPath
}
