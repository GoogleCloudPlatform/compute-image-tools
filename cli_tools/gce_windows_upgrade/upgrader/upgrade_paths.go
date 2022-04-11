package upgrader

const (
	versionWindows2008r2 = "windows-2008r2"
	versionWindows2012r2 = "windows-2012r2"
	versionWindows2016   = "windows-2016"
	versionWindows2019   = "windows-2019"
	versionWindows2022   = "windows-2022"

	licenseForWindows2008r2 = "projects/windows-cloud/global/licenses/windows-server-2008-r2-dc"
	licenseForWindows2012r2 = "projects/windows-cloud/global/licenses/windows-server-2012-r2-dc"
	licenseForWindows2016   = "projects/windows-cloud/global/licenses/windows-server-2016-dc"
	licenseForWindows2019   = "projects/windows-cloud/global/licenses/windows-server-2019-dc"

	licenseForWindows2012r2Upgraded = "projects/windows-cloud/global/licenses/windows-server-2012-r2-dc-in-place-upgrade"
	licenseForWindows2016Upgraded   = "projects/windows-cloud/global/licenses/windows-server-2016-dc-in-place-upgrade"
	licenseForWindows2019Upgraded   = "projects/windows-cloud/global/licenses/windows-server-2019-dc-in-place-upgrade"
	licenseForWindows2022Upgraded   = "projects/windows-cloud/global/licenses/windows-server-2022-dc-in-place-upgrade"

	versionStringForWindows2008r2 = "Windows Server 2008 R2 Datacenter"
	versionStringForWindows2012r2 = "Windows Server 2012 R2 Datacenter"
	versionStringForWindows2016   = "Windows Server 2016 Datacenter"
	versionStringForWindows2019   = "Windows Server 2019 Datacenter"
	versionStringForWindows2022   = "Windows Server 2022 Datacenter"
)

// SupportedVersions shows all supported OS versions of the tool.
var (
	SupportedVersions = []string{
		versionWindows2008r2,
		versionWindows2012r2,
		versionWindows2016,
		versionWindows2019,
		versionWindows2022,
	}

	expectedCurrentLicenseForSourceOS = map[string][]string{
		versionWindows2008r2: {licenseForWindows2008r2},
		versionWindows2012r2: {licenseForWindows2012r2, licenseForWindows2012r2Upgraded},
		versionWindows2016:   {licenseForWindows2016, licenseForWindows2016Upgraded},
		versionWindows2019:   {licenseForWindows2019, licenseForWindows2019Upgraded},
	}

	licenseToAddForTargetOS = map[string]string{
		versionWindows2012r2: licenseForWindows2012r2Upgraded,
		versionWindows2016:   licenseForWindows2016Upgraded,
		versionWindows2019:   licenseForWindows2019Upgraded,
		versionWindows2022:   licenseForWindows2022Upgraded,
	}

	versionString = map[string]string{
		versionWindows2008r2: versionStringForWindows2008r2,
		versionWindows2012r2: versionStringForWindows2012r2,
		versionWindows2016:   versionStringForWindows2016,
		versionWindows2019:   versionStringForWindows2019,
		versionWindows2022:   versionStringForWindows2022,
	}

	installFolderForTargetOS = map[string]string{
		versionWindows2012r2: "Windows_Svr_Std_and_DataCtr_2012_R2_64Bit_English",
		versionWindows2016:   "Win_Server_STD_CORE_2016_64Bit_English",
		versionWindows2019:   "Win_Server_STD_CORE_2019_1809.1_64Bit_English",
		versionWindows2022:   "Win_Server_STD_CORE_2022_64Bit_English",
	}
)

type upgradePath struct {
	enabled                bool
	expectedCurrentLicense []string
	licenseToAdd           string
	expectedCurrentVersion string
	expectedNewVersion     string
	installFolder          string
}

func initUpgradePath(u map[string]map[string]*upgradePath) map[string]map[string]*upgradePath {
	for sourceOS, targets := range u {
		for targetOS, upgradePath := range targets {
			upgradePath.expectedCurrentLicense = expectedCurrentLicenseForSourceOS[sourceOS]
			upgradePath.licenseToAdd = licenseToAddForTargetOS[targetOS]
			upgradePath.expectedCurrentVersion = versionString[sourceOS]
			upgradePath.expectedNewVersion = versionString[targetOS]
			upgradePath.installFolder = installFolderForTargetOS[targetOS]
		}
	}
	return u
}

var upgradePaths = initUpgradePath(map[string]map[string]*upgradePath{
	versionWindows2008r2: {
		versionWindows2012r2: {enabled: true},
	},
	versionWindows2012r2: {
		versionWindows2016: {enabled: true},
		versionWindows2019: {enabled: true},
		versionWindows2022: {enabled: true},
	},
	versionWindows2016: {
		versionWindows2019: {enabled: true},
		versionWindows2022: {enabled: true},
	},
	versionWindows2019: {
		versionWindows2022: {enabled: true},
	},
})

func isSupportedOSVersion(v string) bool {
	for _, sv := range SupportedVersions {
		if sv == v {
			return true
		}
	}
	return false
}

func isSupportedUpgradePath(sourceOS, targetOS string) bool {
	targets, ok := upgradePaths[sourceOS]
	if !ok {
		return false
	}
	upgradePath, ok := targets[targetOS]
	if !ok {
		return false
	}
	return upgradePath.enabled
}
