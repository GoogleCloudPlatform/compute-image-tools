// Package osinfo provides basic system info functions for Windows and
// Linux.
package osinfo


type DistributionInfo struct {
	LongName, ShortName, Version, Kernel string
}