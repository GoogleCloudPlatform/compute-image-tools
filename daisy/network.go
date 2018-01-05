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
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/googleapi"
)

var (
	networkURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/networks/(?P<network>%[2]s)$`, projectRgxStr, rfc1035))
)

type networkRegistry struct {
	baseResourceRegistry
}

func newNetworkRegistry(w *Workflow) *networkRegistry {
	nr := &networkRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "network", urlRgx: networkURLRegex}}
	nr.baseResourceRegistry.deleteFn = nr.deleteFn
	nr.init()
	return nr
}

func (ir *networkRegistry) deleteFn(res *Resource) dErr {
	m := namedSubexp(networkURLRegex, res.link)
	err := ir.w.ComputeClient.DeleteImage(m["project"], m["network"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, err)
	}
	return newErr(err)
}

var networkCache struct {
	exists map[string][]string
	mu     sync.Mutex
}

func networkExists(client compute.Client, project, name string) (bool, dErr) {
	networkCache.mu.Lock()
	defer networkCache.mu.Unlock()
	if networkCache.exists == nil {
		networkCache.exists = map[string][]string{}
	}
	if _, ok := networkCache.exists[project]; !ok {
		nl, err := client.ListNetworks(project)
		if err != nil {
			return false, errf("error listing networks for project %q: %v", project, err)
		}
		var networks []string
		for _, n := range nl {
			networks = append(networks, n.Name)
		}
		networkCache.exists[project] = networks
	}
	return strIn(name, networkCache.exists[project]), nil
}
