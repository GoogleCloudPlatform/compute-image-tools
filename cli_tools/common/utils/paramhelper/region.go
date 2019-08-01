package paramhelper

import (
	"fmt"
	"strings"
)

// GetRegion extracts region from a zones
func GetRegion(zone string) (string, error) {
	if zone == "" {
		return "", fmt.Errorf("zone is empty. Can't determine region")
	}
	zoneStrs := strings.Split(zone, "-")
	if len(zoneStrs) < 2 {
		return "", fmt.Errorf("%v is not a valid zone", zone)
	}
	return strings.Join(zoneStrs[:len(zoneStrs)-1], "-"), nil
}
