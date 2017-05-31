package workflow

import (
	"fmt"
	"regexp"
)

var networkURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/networks/(?P<network>%[1]s)$`, rfc1035))
