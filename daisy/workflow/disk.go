package workflow

import (
	"fmt"
	"regexp"
)

var (
	disks        = map[*Workflow]*resourceMap{}
	diskURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[1]s)/disks/(?P<disk>%[1]s)$`, rfc1035))
)

func deleteDisk(w *Workflow, r *resource) error {
	r.deleted = true
	if err := w.ComputeClient.DeleteDisk(w.Project, w.Zone, r.real); err != nil {
		return err
	}
	return nil
}
