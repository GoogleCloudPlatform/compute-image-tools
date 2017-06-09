package workflow

import (
	"fmt"
	"regexp"
)

var machineTypeURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[1]s)/machineTypes/(?P<machinetype>%[1]s)$`, rfc1035))
