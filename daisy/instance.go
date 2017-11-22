//  Copyright 2017 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package daisy

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/googleapi"
)

const (
	defaultAccessConfigType = "ONE_TO_ONE_NAT"
	defaultDiskMode         = diskModeRW
	defaultDiskType         = "pd-standard"
	diskModeRO              = "READ_ONLY"
	diskModeRW              = "READ_WRITE"
)

var (
	instances      = map[*Workflow]*instanceRegistry{}
	instancesMu    sync.Mutex
	instanceURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[2]s)/instances/(?P<instance>%[2]s)$`, projectRgxStr, rfc1035))
	validDiskModes = []string{diskModeRO, diskModeRW}
)

type instanceRegistry struct {
	baseResourceRegistry
}

func initInstanceRegistry(w *Workflow) {
	ir := &instanceRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "instance", urlRgx: instanceURLRgx}}
	ir.baseResourceRegistry.deleteFn = ir.deleteFn
	ir.init()
	instancesMu.Lock()
	instances[w] = ir
	instancesMu.Unlock()
}

func (ir *instanceRegistry) deleteFn(res *resource) dErr {
	m := namedSubexp(instanceURLRgx, res.link)
	err := ir.w.ComputeClient.DeleteInstance(m["project"], m["zone"], m["instance"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, err)
	}
	return newErr(err)
}

func (ir *instanceRegistry) registerCreation(name string, res *resource, s *Step) dErr {
	// Base creation logic.
	if err := ir.baseResourceRegistry.registerCreation(name, res, s, false); err != nil {
		return err
	}

	// Find the CreateInstance responsible for this.
	var ci *CreateInstance
	for _, ci = range *s.CreateInstances {
		if ci.daisyName == name {
			break
		}
	}
	// Register attachments.
	for _, d := range ci.Disks {
		dName := d.Source
		if d.InitializeParams != nil {
			dName = d.InitializeParams.DiskName
		}
		if err := disks[ir.w].registerAttachment(dName, ci.daisyName, d.Mode, s); err != nil {
			return err
		}
	}
	return nil
}

func (ir *instanceRegistry) registerDeletion(name string, s *Step) dErr {
	if err := ir.baseResourceRegistry.registerDeletion(name, s); err != nil {
		return err
	}
	return disks[ir.w].registerAllDetachments(name, s)
}

func checkDiskMode(m string) bool {
	parts := strings.Split(m, "/")
	m = parts[len(parts)-1]
	return strIn(m, validDiskModes)
}

var instanceCache struct {
	exists map[string]map[string][]string
	mu     sync.Mutex
}

// instanceExists should only be used during validation for existing GCE instances
// and should not be relied or populated for daisy created resources.
func instanceExists(client compute.Client, project, zone, instance string) (bool, dErr) {
	instanceCache.mu.Lock()
	defer instanceCache.mu.Unlock()
	if instanceCache.exists == nil {
		instanceCache.exists = map[string]map[string][]string{}
	}
	if _, ok := instanceCache.exists[project]; !ok {
		instanceCache.exists[project] = map[string][]string{}
	}
	if _, ok := instanceCache.exists[project][zone]; !ok {
		il, err := client.ListInstances(project, zone)
		if err != nil {
			return false, errf("error listing instances for project %q: %v", project, err)
		}
		var instances []string
		for _, i := range il {
			instances = append(instances, i.Name)
		}
		instanceCache.exists[project][zone] = instances
	}
	return strIn(instance, instanceCache.exists[project][zone]), nil
}
