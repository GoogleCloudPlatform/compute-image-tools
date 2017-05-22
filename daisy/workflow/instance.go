package workflow

import (
	"fmt"
	"regexp"
)

var (
	instances        = map[*Workflow]*resourceMap{}
	instanceURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[1]s)/instances/(?P<instance>%[1]s)$`, rfc1035))
)

func deleteInstance(w *Workflow, r *resource) error {
	r.deleted = true
	if err := w.ComputeClient.DeleteInstance(w.Project, w.Zone, r.real); err != nil {
		return err
	}
	return nil
}
