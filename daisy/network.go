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
	networkCache struct {
		exists map[string][]string
		mu     sync.Mutex
	}
	networkURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/networks/(?P<network>%[2]s)$`, projectRgxStr, rfc1035))
)

func networkExists(client daisyCompute.Client, project, name string) (bool, dErr) {
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

// Network is used to create a GCE network.
type Network struct {
	compute.Network
	Resource
}

// MarshalJSON is a hacky workaround to compute.Network's implementation.
func (n *Network) MarshalJSON() ([]byte, error) {
	return json.Marshal(*n)
}

func (n *Network) populate(ctx context.Context, s *Step) dErr {
	var errs dErr
	n.Name, _, errs = n.Resource.populate(ctx, s, n.Name, "")

	n.Description = strOr(n.Description, defaultDescription("Network", s.w.Name, s.w.username))
	n.link = fmt.Sprintf("projects/%s/global/networks/%s", n.Project, n.Name)
	return errs
}

func (n *Network) validate(ctx context.Context, s *Step) dErr {
	pre := fmt.Sprintf("cannot create network %q", n.daisyName)
	errs := n.Resource.validate(ctx, s, pre)

	if n.IPv4Range != "" {
		if _, _, err := net.ParseCIDR(n.IPv4Range); err != nil {
			errs = addErrs(errs, errf("%s: bad IPv4Range: %q, error: %v", pre, n.IPv4Range, err))
		}
	}

	modes := []string{"REGIONAL", "GLOBAL"}
	if n.RoutingConfig != nil && !strIn(n.RoutingConfig.RoutingMode, modes) {
		errs = addErrs(errs, errf("%s: RoutingConfig %q not one of %v", pre, n.RoutingConfig.RoutingMode, modes))
	}

	// Register creation.
	errs = addErrs(errs, s.w.networks.regCreate(n.daisyName, &n.Resource, s, false))
	return errs
}

type networkConnection struct {
	connector, disconnector *Step
}

type networkRegistry struct {
	baseResourceRegistry
	connections          map[string]map[string]*networkConnection
	testDisconnectHelper func(nName, iName string, s *Step) dErr
}

func newNetworkRegistry(w *Workflow) *networkRegistry {
	nr := &networkRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "network", urlRgx: networkURLRegex}}
	nr.baseResourceRegistry.deleteFn = nr.deleteFn
	nr.connections = map[string]map[string]*networkConnection{}
	nr.init()
	return nr
}

func (nr *networkRegistry) deleteFn(res *Resource) dErr {
	m := namedSubexp(networkURLRegex, res.link)
	err := nr.w.ComputeClient.DeleteNetwork(m["project"], m["network"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, err)
	}
	return newErr(err)
}

func (nr *networkRegistry) disconnectHelper(nName, iName string, s *Step) dErr {
	if nr.testDisconnectHelper != nil {
		return nr.testDisconnectHelper(nName, iName, s)
	}
	pre := fmt.Sprintf("step %q cannot disconnect instance %q from network %q", s.name, iName, nName)
	var conn *networkConnection

	if im, _ := nr.connections[nName]; im == nil {
		return errf("%s: not connected", pre)
	} else if conn, _ = im[iName]; conn == nil {
		return errf("%s: not attached", pre)
	} else if conn.disconnector != nil {
		return errf("%s: already disconnected or concurrently disconnected by step %q", pre, conn.disconnector.name)
	} else if !s.nestedDepends(conn.connector) {
		return errf("%s: step %q does not depend on connecting step %q", pre, s.name, conn.connector)
	}
	conn.disconnector = s
	return nil
}

// regConnect marks a network and instance as connected by a Step s.
func (nr *networkRegistry) regConnect(nName, iName string, s *Step) dErr {
	nr.mx.Lock()
	defer nr.mx.Unlock()

	pre := fmt.Sprintf("step %q cannot connect instance %q to network %q", s.name, iName, nName)
	if im, _ := nr.connections[nName]; im == nil {
		nr.connections[nName] = map[string]*networkConnection{iName: {connector: s}}
	} else if nc, _ := im[iName]; nc != nil && !s.nestedDepends(nc.disconnector) {
		return errf("%s: concurrently connected by step %q", pre, nc.connector.name)
	} else {
		nr.connections[nName][iName] = &networkConnection{connector: s}
	}
	return nil
}

func (nr *networkRegistry) regDisconnect(nName, iName string, s *Step) dErr {
	nr.mx.Lock()
	defer nr.mx.Unlock()

	return nr.disconnectHelper(nName, iName, s)
}

// regDisconnect all is called by Instance.regDelete and registers Step s as the disconnector for all networks that iName is currently connected to.
func (nr *networkRegistry) regDisconnectAll(iName string, s *Step) dErr {
	nr.mx.Lock()
	defer nr.mx.Unlock()

	var errs dErr
	// For every network, if connected, disconnect.
	for nName, im := range nr.connections {
		if conn, _ := im[iName]; conn != nil && conn.disconnector == nil {
			errs = addErrs(nr.disconnectHelper(nName, iName, s))
		}
	}

	return errs
}
