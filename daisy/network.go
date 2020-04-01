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

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

var (
	networkURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/networks/(?P<network>%[2]s)$`, projectRgxStr, rfc1035))
)

func (w *Workflow) networkExists(project, network string) (bool, DError) {
	return w.networkCache.resourceExists(func(project string, opts ...daisyCompute.ListCallOption) (interface{}, error) {
		return w.ComputeClient.ListNetworks(project)
	}, project, network)
}

// Network is used to create a GCE network.
type Network struct {
	compute.Network
	AutoCreateSubnetworks *bool `json:"autoCreateSubnetworks,omitempty"`
	Resource
}

// MarshalJSON is a hacky workaround to compute.Network's implementation.
func (n *Network) MarshalJSON() ([]byte, error) {
	return json.Marshal(*n)
}

func (n *Network) populate(ctx context.Context, s *Step) DError {
	var errs DError
	n.Name, errs = n.Resource.populateWithGlobal(ctx, s, n.Name)

	n.Description = strOr(n.Description, defaultDescription("Network", s.w.Name, s.w.username))
	n.link = fmt.Sprintf("projects/%s/global/networks/%s", n.Project, n.Name)

	if n.AutoCreateSubnetworks != nil {
		n.Network.AutoCreateSubnetworks = *n.AutoCreateSubnetworks
		n.Network.ForceSendFields = []string{"AutoCreateSubnetworks"}
	}
	return errs
}

func (n *Network) validate(ctx context.Context, s *Step) DError {
	pre := fmt.Sprintf("cannot create network %q", n.daisyName)
	errs := n.Resource.validate(ctx, s, pre)

	if n.IPv4Range != "" {
		if _, _, err := net.ParseCIDR(n.IPv4Range); err != nil {
			errs = addErrs(errs, Errf("%s: bad IPv4Range: %q, error: %v", pre, n.IPv4Range, err))
		}
	}

	modes := []string{"REGIONAL", "GLOBAL"}
	if n.RoutingConfig != nil && !strIn(n.RoutingConfig.RoutingMode, modes) {
		errs = addErrs(errs, Errf("%s: RoutingConfig %q not one of %v", pre, n.RoutingConfig.RoutingMode, modes))
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
	testDisconnectHelper func(nName, iName string, s *Step) DError
}

func newNetworkRegistry(w *Workflow) *networkRegistry {
	nr := &networkRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "network", urlRgx: networkURLRegex}}
	nr.baseResourceRegistry.deleteFn = nr.deleteFn
	nr.connections = map[string]map[string]*networkConnection{}
	nr.init()
	return nr
}

func (nr *networkRegistry) deleteFn(res *Resource) DError {
	m := NamedSubexp(networkURLRegex, res.link)
	err := nr.w.ComputeClient.DeleteNetwork(m["project"], m["network"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, "failed to delete network", err)
	}
	return newErr("failed to delete network", err)
}

func (nr *networkRegistry) disconnectHelper(nName, iName string, s *Step) DError {
	if nr.testDisconnectHelper != nil {
		return nr.testDisconnectHelper(nName, iName, s)
	}
	pre := fmt.Sprintf("step %q cannot disconnect instance %q from network %q", s.name, iName, nName)
	var conn *networkConnection

	if im, _ := nr.connections[nName]; im == nil {
		return Errf("%s: not connected", pre)
	} else if conn, _ = im[iName]; conn == nil {
		return Errf("%s: not attached", pre)
	} else if conn.disconnector != nil {
		return Errf("%s: already disconnected or concurrently disconnected by step %q", pre, conn.disconnector.name)
	} else if !s.nestedDepends(conn.connector) {
		return Errf("%s: step %q does not depend on connecting step %q", pre, s.name, conn.connector.name)
	}
	conn.disconnector = s
	return nil
}

// regConnect marks a network and instance as connected by a Step s.
func (nr *networkRegistry) regConnect(nName, iName string, s *Step) DError {
	nr.mx.Lock()
	defer nr.mx.Unlock()

	pre := fmt.Sprintf("step %q cannot connect instance %q to network %q", s.name, iName, nName)
	if im, _ := nr.connections[nName]; im == nil {
		nr.connections[nName] = map[string]*networkConnection{iName: {connector: s}}
	} else if nc, _ := im[iName]; nc != nil && !s.nestedDepends(nc.disconnector) {
		return Errf("%s: concurrently connected by step %q", pre, nc.connector.name)
	} else {
		nr.connections[nName][iName] = &networkConnection{connector: s}
	}
	return nil
}

func (nr *networkRegistry) regDisconnect(nName, iName string, s *Step) DError {
	nr.mx.Lock()
	defer nr.mx.Unlock()

	return nr.disconnectHelper(nName, iName, s)
}

// regDisconnect all is called by Instance.regDelete and registers Step s as the disconnector for all networks that iName is currently connected to.
func (nr *networkRegistry) regDisconnectAll(iName string, s *Step) DError {
	nr.mx.Lock()
	defer nr.mx.Unlock()

	var errs DError
	// For every network, if connected, disconnect.
	for nName, im := range nr.connections {
		if conn, _ := im[iName]; conn != nil && conn.disconnector == nil {
			errs = addErrs(nr.disconnectHelper(nName, iName, s))
		}
	}

	return errs
}
