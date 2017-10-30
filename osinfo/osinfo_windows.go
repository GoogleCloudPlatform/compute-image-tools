package osinfo

import (
	"github.com/StackExchange/wmi"
)

func GetDistributionInfo() (*DistributionInfo, error) {
	oi, err := osInfo()
	if err != nil {
		return nil, err
	}

	// TODO(ajackura): Get kernel version from ntoskrnl.exe.
	return &DistributionInfo{ShortName: "windows", LongName: oi.Caption, Version: oi.Version, kernel: oi.Version}, nil
}

type win32_OperatingSystem struct {
	Caption, Version string
}

func osInfo() (*win32_OperatingSystem, error) {
	var ops []win32_OperatingSystem
	if err := wmi.Query(wmi.CreateQuery(&ops, ""), &ops); err != nil {
		return nil, err
	}
	return &ops[0], nil
}
