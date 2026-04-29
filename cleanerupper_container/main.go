// Copyright 2022 Google Inc. All Rights Reserved.
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
	"strings"
	"time"

	osconfigV1alpha "cloud.google.com/go/osconfig/apiv1alpha"
	osconfig "cloud.google.com/go/osconfig/apiv1beta"
	"github.com/GoogleCloudPlatform/cloud-image-tests/cleanerupper"
	daisyCompute "github.com/GoogleCloudPlatform/compute-daisy/compute"
	"google.golang.org/api/option"
)

const (
	timeFormat = "[%v]"
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
	loadBalancers       = flag.Bool("load_balancers", false, "clean load balancer resources")
	networks            = flag.Bool("networks", false, "clean networks")
	snapshots           = flag.Bool("snapshots", false, "clean snapshots")
	guestPolicies       = flag.Bool("guest_policies", false, "clean guest policies")
	osPolicyAssignments = flag.Bool("ospolicy_assignments", false, "clean ospolicy assignments")

	now = time.Now()
)

func shouldDelete(name string, labels map[string]string, t string, s int64) bool {
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

func main() {
	flag.Parse()
	ctx := context.Background()

	cutoff := now.Add(time.Duration(-1) * *duration)
	policy := cleanerupper.AgePolicy(cutoff)
	ps := strings.Split(*projects, ",")
	if len(ps) == 0 {
		log.Fatal("Need to provide at least 1 project")
	}

	if *dryRun {
		fmt.Println("-dry_run flag used, no actual action will be taken")
	}

	var clients, stagingClients cleanerupper.Clients
	var err error
	clients.Daisy, err = daisyCompute.NewClient(ctx, option.WithCredentialsFile(*oauthPath))
	if err != nil {
		log.Fatal(err)
	}
	stagingClients.Daisy = clients.Daisy
	clients.OSConfig, err = osconfig.NewClient(ctx, option.WithCredentialsFile(*oauthPath))
	if err != nil {
		log.Fatal(err)
	}
	stagingClients.OSConfig, err = osconfig.NewClient(ctx, option.WithCredentialsFile(*oauthPath), option.WithEndpoint("staging-osconfig.sandbox.googleapis.com:443"))
	if err != nil {
		log.Fatal(err)
	}
	clients.OSConfigZonal, err = osconfigV1alpha.NewOsConfigZonalClient(ctx, option.WithCredentialsFile(*oauthPath))
	if err != nil {
		log.Fatal(err)
	}
	stagingClients.OSConfigZonal, err = osconfigV1alpha.NewOsConfigZonalClient(ctx, option.WithCredentialsFile(*oauthPath), option.WithEndpoint("staging-osconfig.sandbox.googleapis.com:443"))
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range ps {
		startTime := time.Now()
		fmt.Println(fmt.Sprintf(timeFormat, startTime.Format(time.RFC3339)), "Cleaning project", p)
		// We do all of this sequentially so as not to DOS the API.
		if *instances {
			currTime := time.Now().Format(time.RFC3339)
			fmt.Println(fmt.Sprintf(timeFormat, currTime), "Cleaning instances")
			insts, errs := cleanerupper.CleanInstances(clients, p, policy, *dryRun)
			for _, i := range insts {
				fmt.Printf(" - %s\n", i)
			}
			for _, e := range errs {
				fmt.Println(e)
			}
		}
		if *disks {
			currTime := time.Now().Format(time.RFC3339)
			fmt.Println(fmt.Sprintf(timeFormat, currTime), "Cleaning disks")
			cleaned, errs := cleanerupper.CleanDisks(clients, p, policy, *dryRun)
			for _, c := range cleaned {
				fmt.Printf(" - %s\n", c)
			}
			for _, e := range errs {
				fmt.Println(e)
			}
		}
		if *loadBalancers {
			currTime := time.Now().Format(time.RFC3339)
			fmt.Println(fmt.Sprintf(timeFormat, currTime), "Cleaning Load Balancer Resources")
			cleaned, errs := cleanerupper.CleanLoadBalancerResources(clients, p, policy, *dryRun)
			for _, c := range cleaned {
				fmt.Printf(" - %s\n", c)
			}
			for _, e := range errs {
				fmt.Println(e)
			}
		}
		if *networks {
			currTime := time.Now().Format(time.RFC3339)
			fmt.Println(fmt.Sprintf(timeFormat, currTime), "Cleaning networks")
			cleaned, errs := cleanerupper.CleanNetworks(clients, p, policy, *dryRun)
			for _, c := range cleaned {
				fmt.Printf(" - %s\n", c)
			}
			for _, e := range errs {
				fmt.Println(e)
			}
		}
		if *images {
			currTime := time.Now().Format(time.RFC3339)
			fmt.Println(fmt.Sprintf(timeFormat, currTime), "Cleaning images")
			cleaned, errs := cleanerupper.CleanImages(clients, p, policy, *dryRun)
			for _, c := range cleaned {
				fmt.Printf(" - %s\n", c)
			}
			for _, e := range errs {
				fmt.Println(e)
			}
		}
		if *machineImages {
			currTime := time.Now().Format(time.RFC3339)
			fmt.Println(fmt.Sprintf(timeFormat, currTime), "Cleaning machine images")
			cleaned, errs := cleanerupper.CleanMachineImages(clients, p, policy, *dryRun)
			for _, c := range cleaned {
				fmt.Printf(" - %s\n", c)
			}
			for _, e := range errs {
				fmt.Println(e)
			}
		}
		if *snapshots {
			currTime := time.Now().Format(time.RFC3339)
			fmt.Println(fmt.Sprintf(timeFormat, currTime), "Cleaning snapshots")
			cleaned, errs := cleanerupper.CleanSnapshots(clients, p, policy, *dryRun)
			for _, c := range cleaned {
				fmt.Printf(" - %s\n", c)
			}
			for _, e := range errs {
				fmt.Println(e)
			}
		}
		if *guestPolicies {
			currTime := time.Now().Format(time.RFC3339)
			fmt.Println(fmt.Sprintf(timeFormat, currTime), "Cleaning guest policies")
			cleaned, errs := cleanerupper.CleanGuestPolicies(ctx, clients, p, policy, *dryRun)
			for _, c := range cleaned {
				fmt.Printf(" - %s\n", c)
			}
			for _, e := range errs {
				fmt.Println(e)
			}
			currTime = time.Now().Format(time.RFC3339)
			fmt.Println(fmt.Sprintf(timeFormat, currTime), "Cleaning staging guest policies")
			cleaned, errs = cleanerupper.CleanGuestPolicies(ctx, stagingClients, p, policy, *dryRun)
			for _, c := range cleaned {
				fmt.Printf(" - %s\n", c)
			}
			for _, e := range errs {
				fmt.Println(e)
			}
		}
		if *osPolicyAssignments {
			currTime := time.Now().Format(time.RFC3339)
			fmt.Println(fmt.Sprintf(timeFormat, currTime), "Cleaning os policy assignments")
			cleaned, errs := cleanerupper.CleanOSPolicyAssignments(ctx, clients, p, policy, *dryRun)
			for _, c := range cleaned {
				fmt.Printf(" - %s\n", c)
			}
			for _, e := range errs {
				fmt.Println(e)
			}
			currTime = time.Now().Format(time.RFC3339)
			fmt.Println(fmt.Sprintf(timeFormat, currTime), "Cleaning staing os policy assignments")
			cleaned, errs = cleanerupper.CleanOSPolicyAssignments(ctx, stagingClients, p, policy, *dryRun)
			for _, c := range cleaned {
				fmt.Printf(" - %s\n", c)
			}
			for _, e := range errs {
				fmt.Println(e)
			}
		}
		endTime := time.Now().Format(time.RFC3339)
		duration := time.Since(startTime)
		fmt.Println(fmt.Sprintf(timeFormat, endTime), "Finished cleaning up project", p, "after", duration.Truncate(time.Second).String())
		fmt.Println()
	}
}
