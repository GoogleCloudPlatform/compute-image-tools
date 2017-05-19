package workflow

import (
	"fmt"
	"regexp"
)

var (
	images        = map[*Workflow]*resourceMap{}
	imageURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/images/(?P<image>%[1]s)|family/(?P<family>%[1]s)$`, rfc1035))
)

func deleteImage(w *Workflow, r *resource) error {
	r.deleted = true
	if err := w.ComputeClient.DeleteImage(w.Project, r.real); err != nil {
		return err
	}
	return nil
}
