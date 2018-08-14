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
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sync"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

var (
	subnetworkCache struct {
		exists map[string]map[string][]string
		mu     sync.Mutex
	}
	subnetworkURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?regions/(?P<region>%[2]s)/subnetworks/(?P<subnetwork>%[2]s)$`, projectRgxStr, rfc1035))
)

func subnetworkExists(client daisyCompute.Client, project, region, name string) (bool, dErr) {
	subnetworkCache.mu.Lock()
	defer subnetworkCache.mu.Unlock()
	if subnetworkCache.exists == nil {
		subnetworkCache.exists = map[string]map[string][]string{}
	}
	if _, ok := subnetworkCache.exists[project]; !ok {
		subnetworkCache.exists[project] = map[string][]string{}
	}
	if _, ok := subnetworkCache.exists[project][region]; !ok {
		nl, err := client.ListSubnetworks(project, region)
		if err != nil {
			return false, errf("error listing subnetworks for project %q: %v", project, err)
		}
		var subnetworks []string
		for _, sn := range nl {
			subnetworks = append(subnetworks, sn.Name)
		}
		subnetworkCache.exists[project][region] = subnetworks
	}
	return strIn(name, subnetworkCache.exists[project][region]), nil
}

// Subnetwork is used to create a GCE subnetwork.
type Subnetwork struct {
	compute.Subnetwork
	Resource
}

// MarshalJSON is a hacky workaround to compute.Subnetwork's implementation.
func (sn *Subnetwork) MarshalJSON() ([]byte, error) {
	return json.Marshal(*sn)
}

func (sn *Subnetwork) populate(ctx context.Context, s *Step) dErr {
	var errs dErr
	sn.Name, errs = sn.Resource.populateWithGlobal(ctx, s, sn.Name)

	sn.Description = strOr(sn.Description, defaultDescription("Subnetwork", s.w.Name, s.w.username))
	sn.link = fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", sn.Project, getRegionFromZone(s.w.Zone), sn.Name)
	return errs
}

func (sn *Subnetwork) validate(ctx context.Context, s *Step) dErr {
	pre := fmt.Sprintf("cannot create subnetwork %q", sn.daisyName)
	errs := sn.Resource.validate(ctx, s, pre)

	if sn.Name == "" {
		errs = addErrs(errs, errf("%s: name is mandatory", pre))
	}
	if sn.Network == "" {
		errs = addErrs(errs, errf("%s: network is mandatory", pre))
	}
	sn.Region = strOr(sn.Region, getRegionFromZone(s.w.Zone))
	if _, _, err := net.ParseCIDR(sn.IpCidrRange); err != nil {
		errs = addErrs(errs, errf("%s: bad IpCidrRange: %q, error: %v", pre, sn.IpCidrRange, err))
	}

	// Register creation.
	errs = addErrs(errs, s.w.subnetworks.regCreate(sn.daisyName, &sn.Resource, s, false))
	return errs
}

type subnetworkConnection struct {
	connector, disconnector *Step
}

type subnetworkRegistry struct {
	baseResourceRegistry
	connections          map[string]map[string]*subnetworkConnection
	testDisconnectHelper func(nName, iName string, s *Step) dErr
}

func newSubnetworkRegistry(w *Workflow) *subnetworkRegistry {
	nr := &subnetworkRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "subnetwork", urlRgx: subnetworkURLRegex}}
	nr.baseResourceRegistry.deleteFn = nr.deleteFn
	nr.connections = map[string]map[string]*subnetworkConnection{}
	nr.init()
	return nr
}

func (nr *subnetworkRegistry) deleteFn(res *Resource) dErr {
	m := namedSubexp(subnetworkURLRegex, res.link)
	err := nr.w.ComputeClient.DeleteSubnetwork(m["project"], m["region"], m["subnetwork"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, err)
	}
	return newErr(err)
}

func (nr *subnetworkRegistry) disconnectHelper(nName, iName string, s *Step) dErr {
	if nr.testDisconnectHelper != nil {
		return nr.testDisconnectHelper(nName, iName, s)
	}
	pre := fmt.Sprintf("step %q cannot disconnect instance %q from subnetwork %q", s.name, iName, nName)
	var conn *subnetworkConnection

	if im, _ := nr.connections[nName]; im == nil {
		return errf("%s: not connected", pre)
	} else if conn, _ = im[iName]; conn == nil {
		return errf("%s: not attached", pre)
	} else if conn.disconnector != nil {
		return errf("%s: already disconnected or concurrently disconnected by step %q", pre, conn.disconnector.name)
	} else if !s.nestedDepends(conn.connector) {
		return errf("%s: step %q does not depend on connecting step %q", pre, s.name, conn.connector.name)
	}
	conn.disconnector = s
	return nil
}

// regConnect marks a subnetwork and instance as connected by a Step s.
func (nr *subnetworkRegistry) regConnect(nName, iName string, s *Step) dErr {
	nr.mx.Lock()
	defer nr.mx.Unlock()

	pre := fmt.Sprintf("step %q cannot connect instance %q to subnetwork %q", s.name, iName, nName)
	if im, _ := nr.connections[nName]; im == nil {
		nr.connections[nName] = map[string]*subnetworkConnection{iName: {connector: s}}
	} else if nc, _ := im[iName]; nc != nil && !s.nestedDepends(nc.disconnector) {
		return errf("%s: concurrently connected by step %q", pre, nc.connector.name)
	} else {
		nr.connections[nName][iName] = &subnetworkConnection{connector: s}
	}
	return nil
}

func (nr *subnetworkRegistry) regDisconnect(nName, iName string, s *Step) dErr {
	nr.mx.Lock()
	defer nr.mx.Unlock()

	return nr.disconnectHelper(nName, iName, s)
}

// regDisconnect all is called by Instance.regDelete and registers Step s as the disconnector for all subnetworks that iName is currently connected to.
func (nr *subnetworkRegistry) regDisconnectAll(iName string, s *Step) dErr {
	nr.mx.Lock()
	defer nr.mx.Unlock()

	var errs dErr
	// For every subnetwork, if connected, disconnect.
	for nName, im := range nr.connections {
		if conn, _ := im[iName]; conn != nil && conn.disconnector == nil {
			errs = addErrs(nr.disconnectHelper(nName, iName, s))
		}
	}

	return errs
}
