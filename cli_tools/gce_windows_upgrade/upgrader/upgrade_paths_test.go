package upgrader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitUpgradePaths(t *testing.T) {
	p := initUpgradePath(map[string]map[string]*upgradePath{
		versionWindows2008r2: {
			versionWindows2012r2: {enabled: true},
		},
		versionWindows2016: {
			versionWindows2019: {enabled: true},
			versionWindows2022: {enabled: false},
		},
	})

	for _, targets := range p {
		for _, up := range targets {
			assert.NotEmpty(t, up.installFolder)
			assert.NotEmpty(t, up.expectedNewVersion)
			assert.NotEmpty(t, up.expectedCurrentVersion)
			assert.NotEmpty(t, up.licenseToAdd)
			assert.NotNil(t, up.expectedCurrentLicense)
		}
	}
}

func TestIsSupportedOSVersion(t *testing.T) {
	for _, v := range SupportedVersions {
		assert.True(t, isSupportedOSVersion(v))
	}
	assert.False(t, isSupportedOSVersion(""))
	assert.False(t, isSupportedOSVersion("android"))
}

func TestIsSupportedUpgradePath(t *testing.T) {
	assert.True(t, isSupportedUpgradePath(versionWindows2008r2, versionWindows2012r2))
	assert.False(t, isSupportedUpgradePath(versionWindows2012r2, versionWindows2008r2))
	assert.False(t, isSupportedUpgradePath("unknown", versionWindows2012r2))
	assert.False(t, isSupportedUpgradePath(versionWindows2008r2, "unknown"))
}
