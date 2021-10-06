// Copyright 2019 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	osconfigV1alpha "cloud.google.com/go/osconfig/apiv1alpha"
	osconfig "cloud.google.com/go/osconfig/apiv1beta"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	osconfigv1alphapb "google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha"
	osconfigpb "google.golang.org/genproto/googleapis/cloud/osconfig/v1beta"
)

const (
	keepLabel = "do-not-delete"
)

var (
	projects  = flag.String("projects", "", "comma delineated list of projects to clean")
	dryRun    = flag.Bool("dry_run", true, "only print out actions, don't delete any resources")
	oauthPath = flag.String("oauth", "", "oauth file to use to authenticate")
	duration  = flag.Duration("duration", 24*time.Hour, "cleanup all resources with a lifetime greater than this")

	instances           = flag.Bool("instances", false, "clean instances")
	disks               = flag.Bool("disks", false, "clean disks")
	images              = flag.Bool("images", false, "clean images")
	machineImages       = flag.Bool("machine_images", false, "clean machine images")
	networks            = flag.Bool("networks", false, "clean networks")
	snapshots           = flag.Bool("snapshots", false, "clean snapshots")
	guestPolicies       = flag.Bool("guest_policies", false, "clean guest policies")
	osPolicyAssignments = flag.Bool("ospolicy_assignments", false, "clean ospolicy assignments")

	now = time.Now()
)

func shouldDelete(name string, labels map[string]string, t string, s int64) bool {
	if _, ok := labels[keepLabel]; ok {
		return false
	}
	if strings.Contains(name, keepLabel) {
		return false
	}
	var c time.Time
	var err error
	switch {
	case t != "":
		c, err = time.Parse(time.RFC3339, t)
		if err != nil {
			fmt.Printf("Error parsing create time %q: %v\n", t, err)
			return false
		}
	case s != 0:
		c = time.Unix(s, 0)
	default:
		return false
	}

	return c.Add(*duration).Before(now)
}

func cleanInstances(client daisyCompute.Client, project string) {
	instances, err := client.AggregatedListInstances(project)
	if err != nil {
		fmt.Printf("Error listing instance in project %q: %v\n", project, err)
		return
	}

	fmt.Println("Cleaning instances:")
	var wg sync.WaitGroup
	for _, i := range instances {
		if i.DeletionProtection {
			continue
		}
		if !shouldDelete(i.Name, i.Labels, i.CreationTimestamp, 0) {
			continue
		}

		zone := path.Base(i.Zone)
		name := path.Base(i.SelfLink)
		fmt.Printf("- projects/%s/zones/%s/instances/%s\n", project, zone, name)
		if *dryRun {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.DeleteInstance(project, zone, name); err != nil {
				fmt.Printf("Error deleting instance: %v\n", err)
			}
		}()
	}
	wg.Wait()
}

func cleanDisks(client daisyCompute.Client, project string) {
	disks, err := client.AggregatedListDisks(project)
	if err != nil {
		fmt.Printf("Error listing disks in project %q: %v\n", project, err)
		return
	}

	fmt.Println("Cleaning disks:")
	var wg sync.WaitGroup
	for _, d := range disks {
		if !shouldDelete(d.Name, d.Labels, d.CreationTimestamp, 0) {
			continue
		}

		zone := path.Base(d.Zone)
		name := path.Base(d.SelfLink)
		fmt.Printf("- projects/%s/zones/%s/disks/%s\n", project, zone, name)
		if *dryRun {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.DeleteDisk(project, zone, name); err != nil {
				fmt.Printf("Error deleting disk: %v\n", err)
			}
		}()
	}
	wg.Wait()
}

func cleanImages(client daisyCompute.Client, project string) {
	images, err := client.ListImages(project)
	if err != nil {
		fmt.Printf("Error listing images in project %q: %v\n", project, err)
		return
	}

	fmt.Println("Cleaning images:")
	var wg sync.WaitGroup
	for _, i := range images {
		if !shouldDelete(i.Name, i.Labels, i.CreationTimestamp, 0) {
			continue
		}

		name := path.Base(i.SelfLink)
		fmt.Printf("- projects/%s/global/images/%s\n", project, name)
		if *dryRun {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.DeleteImage(project, name); err != nil {
				fmt.Printf("Error deleting image: %v\n", err)
			}
		}()
	}
	wg.Wait()
}

func cleanMachineImages(client daisyCompute.Client, project string) {
	machineImages, err := client.ListMachineImages(project)
	if err != nil {
		fmt.Printf("Error listing machine images in project %q: %v\n", project, err)
		return
	}

	fmt.Println("Cleaning machine images:")
	var wg sync.WaitGroup
	for _, mi := range machineImages {
		if !shouldDelete(mi.Name, nil, mi.CreationTimestamp, 0) {
			continue
		}
		name := path.Base(mi.SelfLink)
		fmt.Printf("- projects/%s/global/machineImages/%s\n", project, name)
		if *dryRun {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.DeleteMachineImage(project, name); err != nil {
				fmt.Printf("Error deleting machine image: %v\n", err)
			}
		}()
	}
	wg.Wait()
}

func cleanSnapshots(client daisyCompute.Client, project string) {
	snapshots, err := client.ListSnapshots(project)
	if err != nil {
		fmt.Printf("Error listing snapshots in project %q: %v\n", project, err)
		return
	}

	fmt.Println("Cleaning snapshots:")
	var wg sync.WaitGroup
	for _, s := range snapshots {
		if !shouldDelete(s.Name, s.Labels, s.CreationTimestamp, 0) {
			continue
		}

		name := path.Base(s.SelfLink)
		fmt.Printf("- projects/%s/global/snapshots/%s\n", project, name)
		if *dryRun {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.DeleteSnapshot(project, name); err != nil {
				fmt.Printf("Error deleting snapshot: %v\n", err)
			}
		}()
	}
	wg.Wait()
}

func cleanNetworks(client daisyCompute.Client, project string) {
	networks, err := client.ListNetworks(project)
	if err != nil {
		fmt.Printf("Error listing networks in project %q: %v\n", project, err)
		return
	}

	firewalls, err := client.ListFirewallRules(project)
	if err != nil {
		fmt.Printf("Error listing firewalls in project %q: %v\n", project, err)
		return
	}

	subnetworks, err := client.AggregatedListSubnetworks(project)
	if err != nil {
		fmt.Printf("Error listing subnetworks in project %q: %v\n", project, err)
		return
	}

	fmt.Println("Cleaning networks and associated subnetworks and firewall rules:")
	var wg sync.WaitGroup
	for _, n := range networks {
		// Don't delete the default network, or one with 'delete' in the description.
		if n.Name == "default" || strings.Contains(n.Description, "delete") {
			continue
		}

		if !shouldDelete(n.Name, nil, n.CreationTimestamp, 0) {
			continue
		}

		name := path.Base(n.SelfLink)
		fmt.Printf("- projects/%s/global/networks/%s\n", project, name)
		for _, f := range firewalls {
			if f.Network != n.SelfLink {
				continue
			}
			name := path.Base(f.SelfLink)
			fmt.Printf("  - projects/%s/global/firewalls/%s\n", project, name)
			if *dryRun {
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := client.DeleteFirewallRule(project, name); err != nil {
					fmt.Printf("Error deleting firewall: %v\n", err)
				}
			}()
		}

		for _, sn := range subnetworks {
			if sn.Network != n.SelfLink {
				continue
			}
			// If this network is setup with auto subnetworks we need to ignore any subnetworks that are in 10.128.0.0/9.
			// https://cloud.google.com/vpc/docs/vpc#ip-ranges
			if n.AutoCreateSubnetworks == true {
				i, err := strconv.Atoi(strings.Split(sn.IpCidrRange, ".")[1])
				if err != nil {
					fmt.Printf("Error parsing network range %q: %v\n", sn.IpCidrRange, err)
				}
				if i >= 128 {
					continue
				}
			}

			region := path.Base(sn.Region)
			fmt.Printf("  - projects/%s/regions/%s/subnetworks/%s\n", project, region, sn.Name)
			if *dryRun {
				continue
			}
			wg.Add(1)
			go func(snName string) {
				defer wg.Done()
				if err := client.DeleteSubnetwork(project, region, snName); err != nil {
					fmt.Printf("Error deleting subnetwork: %v\n", err)
				}
			}(sn.Name)
		}
		if *dryRun {
			continue
		}
		wg.Wait()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.DeleteNetwork(project, name); err != nil {
				fmt.Printf("Error deleting network: %v\n", err)
			}
		}()
	}
	wg.Wait()
}

func cleanGuestPolicies(ctx context.Context, client *osconfig.Client, project string) {
	fmt.Println("Cleaning GuestPolicies:")
	var wg sync.WaitGroup
	itr := client.ListGuestPolicies(ctx, &osconfigpb.ListGuestPoliciesRequest{Parent: "projects/" + project})
	for {
		gp, err := itr.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			fmt.Printf("Error calling ListGuestPolicies in project %q: %v\n", project, err)
			return
		}
		if !shouldDelete(gp.GetName(), nil, "", gp.GetCreateTime().GetSeconds()) {
			continue
		}
		fmt.Printf("- %s\n", gp.GetName())
		if *dryRun {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.DeleteGuestPolicy(ctx, &osconfigpb.DeleteGuestPolicyRequest{Name: gp.GetName()}); err != nil {
				fmt.Printf("Error deleting GuestPolicy: %v\n", err)
			}
		}()
	}
	wg.Wait()
}

func cleanOSPolicyAssignments(ctx context.Context, computeClient daisyCompute.Client, client *osconfigV1alpha.OsConfigZonalClient, project string) {
	fmt.Println("Cleaning OSPolicyAssignments:")
	var wg sync.WaitGroup
	zones, err := computeClient.ListZones(project)
	if err != nil {
		fmt.Printf("Error calling ListZones in project %q: %v\n", project, err)
		return
	}
	for _, zone := range zones {
		if zone.Name == "us-east2-a" {
			continue
		}
		fmt.Println("Cleaning zone:", zone.Name)
		now := time.Now()
		itr := client.ListOSPolicyAssignments(ctx, &osconfigv1alphapb.ListOSPolicyAssignmentsRequest{Parent: fmt.Sprintf("projects/%s/locations/%s", project, zone.Name)})
		var count int
		for {
			ospa, err := itr.Next()
			if err != nil {
				if err == iterator.Done {
					fmt.Println(time.Since(now), zone.Name)
					break
				}
				fmt.Printf("Error calling ListOSPolicyAssignments in project %q: %v\n", project, err)
				break
			}
			if !shouldDelete(ospa.GetName(), nil, "", ospa.GetRevisionCreateTime().GetSeconds()) {
				continue
			}
			if *dryRun {
				continue
			}
			count++
			wg.Add(1)
			go func() {
				defer wg.Done()
				op, err := client.DeleteOSPolicyAssignment(ctx, &osconfigv1alphapb.DeleteOSPolicyAssignmentRequest{Name: ospa.GetName()})
				if err != nil {
					fmt.Printf("Error deleting OSPolicyAssignment: %v\n", err)
					return
				}
				op.Wait(ctx)
			}()
		}
		fmt.Printf("Cleaning %d OSPolicyAssignments\n", count)
		wg.Wait()
	}
}

func main() {
	flag.Parse()
	ctx := context.Background()

	ps := strings.Split(*projects, ",")
	if len(ps) == 0 {
		log.Fatal("Need to provide at least 1 project")
	}

	if *dryRun {
		fmt.Println("-dry_run flag used, no actual action will be taken")
	}

	computeClient, err := daisyCompute.NewClient(ctx, option.WithCredentialsFile(*oauthPath))
	if err != nil {
		log.Fatal(err)
	}
	osconfigClientV1beta, err := osconfig.NewClient(ctx, option.WithCredentialsFile(*oauthPath))
	if err != nil {
		log.Fatal(err)
	}
	osconfigClientV1betaStaging, err := osconfig.NewClient(ctx, option.WithCredentialsFile(*oauthPath), option.WithEndpoint("staging-osconfig.sandbox.googleapis.com:443"))
	if err != nil {
		log.Fatal(err)
	}
	osconfigZonalClientV1alpha, err := osconfigV1alpha.NewOsConfigZonalClient(ctx, option.WithCredentialsFile(*oauthPath))
	if err != nil {
		log.Fatal(err)
	}
	osconfigZonalClientV1alphaStaging, err := osconfigV1alpha.NewOsConfigZonalClient(ctx, option.WithCredentialsFile(*oauthPath), option.WithEndpoint("staging-osconfig.sandbox.googleapis.com:443"))
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range ps {
		fmt.Println("Cleaning project", p)
		// We do all of this sequentially so as not to DOS the API.
		if *instances {
			cleanInstances(computeClient, p)
		}
		if *disks {
			cleanDisks(computeClient, p)
		}
		if *networks {
			cleanNetworks(computeClient, p)
		}
		if *images {
			cleanImages(computeClient, p)
		}
		if *machineImages {
			cleanMachineImages(computeClient, p)
		}
		if *snapshots {
			cleanSnapshots(computeClient, p)
		}
		if *guestPolicies {
			cleanGuestPolicies(ctx, osconfigClientV1beta, p)
			cleanGuestPolicies(ctx, osconfigClientV1betaStaging, p)
		}
		if *osPolicyAssignments {
			cleanOSPolicyAssignments(ctx, computeClient, osconfigZonalClientV1alpha, p)
			cleanOSPolicyAssignments(ctx, computeClient, osconfigZonalClientV1alphaStaging, p)
		}
		fmt.Println()
	}
}
