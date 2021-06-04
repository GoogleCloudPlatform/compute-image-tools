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
	"fmt"
	"net/http"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

// DeleteResources deletes GCE/GCS resources.
type DeleteResources struct {
	Disks         []string `json:",omitempty"`
	Images        []string `json:",omitempty"`
	MachineImages []string `json:",omitempty"`
	Instances     []string `json:",omitempty"`
	Networks      []string `json:",omitempty"`
	Subnetworks   []string `json:",omitempty"`
	GCSPaths      []string `json:",omitempty"`
	Firewalls     []string `json:",omitempty"`
}

func (d *DeleteResources) populate(ctx context.Context, s *Step) DError {
	for i, disk := range d.Disks {
		if diskURLRgx.MatchString(disk) {
			d.Disks[i] = extendPartialURL(disk, s.w.Project)
		}
	}
	for i, image := range d.Images {
		if imageURLRgx.MatchString(image) {
			d.Images[i] = extendPartialURL(image, s.w.Project)
		}
	}
	for i, machineImage := range d.MachineImages {
		if machineImageURLRgx.MatchString(machineImage) {
			d.MachineImages[i] = extendPartialURL(machineImage, s.w.Project)
		}
	}
	for i, instance := range d.Instances {
		if instanceURLRgx.MatchString(instance) {
			d.Instances[i] = extendPartialURL(instance, s.w.Project)
		}
	}
	for i, network := range d.Networks {
		if networkURLRegex.MatchString(network) {
			d.Networks[i] = extendPartialURL(network, s.w.Project)
		}
	}
	for i, subnetwork := range d.Subnetworks {
		if subnetworkURLRegex.MatchString(subnetwork) {
			d.Subnetworks[i] = extendPartialURL(subnetwork, s.w.Project)
		}
	}
	for i, firewall := range d.Firewalls {
		if firewallRuleURLRegex.MatchString(firewall) {
			d.Firewalls[i] = extendPartialURL(firewall, s.w.Project)
		}
	}
	return nil
}

func (d *DeleteResources) validateInstance(i string, s *Step) DError {
	if err := s.w.instances.regDelete(i, s); err != nil {
		return err
	}
	ir, _ := s.w.instances.get(i)

	// Get the Instance that created this instance, if any.

	if ir.creator != nil {
		//Try GA
		for _, createI := range (*ir.creator.CreateInstances).Instances {
			if createI.daisyName != i {
				continue
			}
			attachedDisks := createI.Disks
			for _, ad := range attachedDisks {
				if !ad.AutoDelete {
					continue
				}
				dName := ad.Source
				if ad.InitializeParams != nil {
					dName = ad.InitializeParams.DiskName
				}
				if err := s.w.disks.regDelete(dName, s); err != nil {
					return err
				}
				return nil
			}
		}

		//Try Beta
		for _, createI := range (*ir.creator.CreateInstances).InstancesBeta {
			if createI.daisyName != i {
				continue
			}
			attachedDisks := createI.Disks
			for _, ad := range attachedDisks {
				if !ad.AutoDelete {
					continue
				}
				dName := ad.Source
				if ad.InitializeParams != nil {
					dName = ad.InitializeParams.DiskName
				}
				if err := s.w.disks.regDelete(dName, s); err != nil {
					return err
				}
				return nil
			}
		}
	}

	return nil
}

func (d *DeleteResources) checkError(err DError, s *Step) DError {
	if err != nil && strings.HasSuffix(err.etype(), resourceDNEError) {
		s.w.LogStepInfo(s.name, "DeleteResources", "WARNING: Error validating deletion: %v", err)
		return nil
	} else if err != nil && err.etype() == imageObsoleteDeletedError {
		return nil
	}
	return err
}

func (d *DeleteResources) validate(ctx context.Context, s *Step) DError {
	// Instance checking.
	for _, i := range d.Instances {
		if err := d.validateInstance(i, s); d.checkError(err, s) != nil {
			return err
		}
	}

	// Disk checking.
	for _, disk := range d.Disks {
		if err := s.w.disks.regDelete(disk, s); d.checkError(err, s) != nil {
			return err
		}
	}

	// Image checking.
	for _, i := range d.Images {
		if err := s.w.images.regDelete(i, s); d.checkError(err, s) != nil {
			return err
		}
	}

	// Machine image checking.
	for _, i := range d.MachineImages {
		if err := s.w.machineImages.regDelete(i, s); d.checkError(err, s) != nil {
			return err
		}
	}

	// Network checking.
	for _, n := range d.Networks {
		if err := s.w.networks.regDelete(n, s); d.checkError(err, s) != nil {
			return err
		}
	}

	// Subnetwork checking.
	for _, sn := range d.Subnetworks {
		if err := s.w.subnetworks.regDelete(sn, s); d.checkError(err, s) != nil {
			return err
		}
	}

	// GCS path checking
	for _, p := range d.GCSPaths {
		bkt, _, err := splitGCSPath(p)
		if err != nil {
			return err
		}

		// Check if bucket exists and is writeable.
		writableBkts.mx.Lock()
		if !strIn(bkt, writableBkts.bkts) {
			if _, err := s.w.StorageClient.Bucket(bkt).Attrs(ctx); err != nil {
				return Errf("error reading bucket %q: %v", bkt, err)
			}

			tObj := s.w.StorageClient.Bucket(bkt).Object(fmt.Sprintf("daisy-validate-%s-%s", s.name, s.w.id))
			w := tObj.NewWriter(ctx)
			if _, err := w.Write(nil); err != nil {
				return newErr("failed to write to GCS object when deleting resources", err)
			}
			if err := w.Close(); err != nil {
				return Errf("error writing to bucket %q: %v", bkt, err)
			}
			if err := tObj.Delete(ctx); err != nil {
				return Errf("error deleting file %+v after write validation: %v", tObj, err)
			}
			writableBkts.bkts = append(writableBkts.bkts, bkt)
		}
		writableBkts.mx.Unlock()
	}

	return nil
}

func recursiveGCSDelete(ctx context.Context, w *Workflow, bkt, prefix string) DError {
	it := w.StorageClient.Bucket(bkt).Objects(ctx, &storage.Query{Prefix: prefix})
	for objAttr, err := it.Next(); err != iterator.Done; objAttr, err = it.Next() {
		if err != nil {
			return typedErr(apiError, "failed to iterate GCS object for deletion", err)
		}
		if objAttr.Size == 0 {
			continue
		}
		if err := w.StorageClient.Bucket(bkt).Object(objAttr.Name).Delete(ctx); err != nil {
			return typedErr(apiError, "failed to delete GCS object", err)
		}
	}
	return nil
}

// Waits for the whole group to run. Monitors for error and cancels.
// Returns true if error should be raised, false otherwise.
func waitGroup(wg *sync.WaitGroup, e chan DError, w *Workflow) (bool, DError) {
	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		if err != nil {
			return true, err
		}
	case <-w.Cancel:
		return true, nil
	}
	return false, nil
}

func (d *DeleteResources) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)

	for _, i := range d.Instances {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			w.LogStepInfo(s.name, "DeleteResources", "Deleting instance %q.", i)
			if err := w.instances.delete(i); err != nil {
				if err.etype() == resourceDNEError {
					w.LogStepInfo(s.name, "DeleteResources", "WARNING: Error deleting instance %q: %v", i, err)
					return
				}
				e <- err
			}
		}(i)
	}

	for _, i := range d.Images {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			w.LogStepInfo(s.name, "DeleteResources", "Deleting image %q.", i)
			if err := w.images.delete(i); err != nil {
				if err.etype() == resourceDNEError {
					w.LogStepInfo(s.name, "DeleteResources", "WARNING: Error deleting image %q: %v", i, err)
					return
				}
				e <- err
			}
		}(i)
	}

	for _, i := range d.MachineImages {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			w.LogStepInfo(s.name, "DeleteResources", "Deleting machine image %q.", i)
			if err := w.machineImages.delete(i); err != nil {
				if err.etype() == resourceDNEError {
					w.LogStepInfo(s.name, "DeleteResources", "WARNING: Error deleting machine image %q: %v", i, err)
					return
				}
				e <- err
			}
		}(i)
	}

	for _, p := range d.GCSPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			bkt, obj, err := splitGCSPath(p)
			if err != nil {
				e <- err
				return
			}

			if obj == "" || strings.HasSuffix(obj, "/") {
				if err := recursiveGCSDelete(ctx, s.w, bkt, obj); err != nil {
					e <- err
				}
				return
			}

			if err := w.StorageClient.Bucket(bkt).Object(obj).Delete(ctx); err != nil {
				if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
					w.LogStepInfo(s.name, "DeleteResources", "WARNING: Error deleting GCS Path %q: %v", p, err)
					return
				}
				e <- Errf("error deleting GCS path %q: %v", p, err)
			}
		}(p)
	}

	if abort, ret := waitGroup(&wg, e, w); abort {
		return ret
	}

	// Delete disks only after instances have been deleted.
	e = make(chan DError)
	for _, d := range d.Disks {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			w.LogStepInfo(s.name, "DeleteResources", "Deleting disk %q.", d)
			if err := w.disks.delete(d); err != nil {
				if err.etype() == resourceDNEError {
					w.LogStepInfo(s.name, "DeleteResources", "WARNING: Error deleting disk %q: %v", d, err)
					return
				}
				e <- err
			}
		}(d)
	}

	// Delete firewalls after instance have been deleted
	for _, n := range d.Firewalls {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			w.LogStepInfo(s.name, "DeleteResources", "Deleting firewall %q.", n)
			if err := w.firewallRules.delete(n); err != nil {
				if err.etype() == resourceDNEError {
					w.LogStepInfo(s.name, "DeleteResources", "WARNING: Error deleting firewall %q: %v", n, err)
				}
				e <- err
			}
		}(n)
	}

	// Delete subnetworks after instances.
	for _, sn := range d.Subnetworks {
		wg.Add(1)
		go func(sn string) {
			defer wg.Done()
			w.LogStepInfo(s.name, "DeleteResources", "Deleting subnetwork %q.", sn)
			if err := w.subnetworks.delete(sn); err != nil {
				if err.etype() == resourceDNEError {
					w.LogStepInfo(s.name, "DeleteResources", "WARNING: Error deleting subnetwork %q: %v", sn, err)
				}
				e <- err
			}
		}(sn)
	}

	if abort, ret := waitGroup(&wg, e, w); abort {
		return ret
	}

	// Delete networks after firewalls have been deleted
	for _, n := range d.Networks {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			w.LogStepInfo(s.name, "DeleteResources", "Deleting network %q.", n)
			if err := w.networks.delete(n); err != nil {
				if err.etype() == resourceDNEError {
					w.LogStepInfo(s.name, "DeleteResources", "WARNING: Error deleting network %q: %v", n, err)
				}
				e <- err
			}
		}(n)
	}

	_, ret := waitGroup(&wg, e, w)
	return ret
}
