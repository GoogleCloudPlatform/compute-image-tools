package daisy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

var (
	targetPoolURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?regions/(?P<zone>%[2]s)/targetPools/(?P<targetPool>%[2]s)$`, projectRgxStr, rfc1035))
)

func (w *Workflow) targetPoolExists(project, zone, targetInstance string) (bool, DError) {
	return w.targetInstanceCache.resourceExists(func(project, zone string, opts ...daisyCompute.ListCallOption) (interface{}, error) {
		return w.ComputeClient.ListTargetInstances(project, zone)
	}, project, zone, targetInstance)
}

// TargetPool is used to create a GCE targetPool.
type TargetPool struct {
	compute.TargetPool
	Resource
}

// MarshalJSON is a hacky workaround to compute.TargetPool's implementation.
func (tp *TargetPool) MarshalJSON() ([]byte, error) {
	return json.Marshal(*tp)
}

func (tp *TargetPool) populate(ctx context.Context, s *Step) DError {
	var errs DError
	tp.Name, errs = tp.Resource.populateWithGlobal(ctx, s, tp.Name)

	tp.Description = strOr(tp.Description, defaultDescription("TargetPool", s.w.Name, s.w.username))
	tp.link = fmt.Sprintf("projects/%s/regions/%s/targetPools/%s", tp.Project, getRegionFromZone(s.w.Zone), tp.Name)
	return errs
}

func (tp *TargetPool) validate(ctx context.Context, s *Step) DError {
	pre := fmt.Sprintf("cannot create target pool %q", tp.daisyName)
	errs := tp.Resource.validate(ctx, s, pre)

	if tp.Name == "" {
		errs = addErrs(errs, Errf("%s: TargetPool not set", pre))
	}

	// Register creation.
	errs = addErrs(errs, s.w.targetPools.regCreate(tp.daisyName, &tp.Resource, s, false))
	return errs
}

type targetPoolConnection struct {
	connector, disconnector *Step
}

type targetPoolRegistry struct {
	baseResourceRegistry
	connections          map[string]map[string]*targetPoolConnection
	testDisconnectHelper func(nName, iName string, s *Step) DError
}

func newTargetPoolRegistry(w *Workflow) *targetPoolRegistry {
	tpr := &targetPoolRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "targetPool", urlRgx: targetPoolURLRegex}}
	tpr.baseResourceRegistry.deleteFn = tpr.deleteFn
	tpr.connections = map[string]map[string]*targetPoolConnection{}
	tpr.init()
	return tpr
}

func (tpr *targetPoolRegistry) deleteFn(res *Resource) DError {
	m := NamedSubexp(targetPoolURLRegex, res.link)
	err := tpr.w.ComputeClient.DeleteTargetPool(m["project"], m["region"], m["targetPool"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, "failed to delete target pool", err)
	}
	return newErr("failed to delete target pool", err)
}
