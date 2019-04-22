//  Copyright 2018 Google Inc. All Rights Reserved.
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

//+build !test

package ospatch

import (
	"fmt"
	"os/exec"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows/registry"
)

func systemRebootRequired() (bool, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	k.Close()

	return true, nil
}

func getIterativeProp(src *ole.IDispatch, prop string) (*ole.IDispatch, int32, error) {
	raw, err := src.GetProperty(prop)
	if err != nil {
		return nil, 0, err
	}
	dis := raw.ToIDispatch()

	countRaw, err := dis.GetProperty("Count")
	if err != nil {
		return nil, 0, err
	}
	count, _ := countRaw.Value().(int32)

	return dis, count, nil
}

func installUpdate(r *patchRun, classFilter, excludes map[string]struct{}, session, updt *ole.IDispatch) error {
	title, err := updt.GetProperty("Title")
	if err != nil {
		return fmt.Errorf(`updt.GetProperty("Title"): %v`, err)
	}

	kbArticleIDs, kbArticleIDsCount, err := getIterativeProp(updt, "KBArticleIDs")
	if err != nil {
		return fmt.Errorf(`getIterativeProp(updt, "KBArticleIDs"): %v`, err)
	}

	logger.Debugf("filtering out KBs: %q\n", excludes)
	for i := 0; i < int(kbArticleIDsCount); i++ {
		kbRaw, err := kbArticleIDs.GetProperty("Item", i)
		if err != nil {
			return err
		}
		if _, ok := excludes[kbRaw.ToString()]; ok {
			logger.Debugf("Update %s (%s) matched exclude list\n", title.ToString(), kbRaw.ToString())
			return nil
		}
	}

	logger.Debugf("filtering by classifications: %q\n", classFilter)
	if len(classFilter) != 0 {
		categories, categoriesCount, err := getIterativeProp(updt, "Categories")
		if err != nil {
			return fmt.Errorf(`getIterativeProp(updt, "Categories"): %v`, err)
		}

		var found bool
		for i := 0; i < int(categoriesCount); i++ {
			catRaw, err := categories.GetProperty("Item", i)
			if err != nil {
				return err
			}

			catIdRaw, err := catRaw.ToIDispatch().GetProperty("CategoryID")
			if err != nil {
				return fmt.Errorf(`catRaw.ToIDispatch().GetProperty("CategoryID"): %v`, err)
			}

			if _, ok := classFilter[catIdRaw.ToString()]; ok {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	updateCollObj, err := oleutil.CreateObject("Microsoft.Update.UpdateColl")
	if err != nil {
		return fmt.Errorf(`oleutil.CreateObject("updateColl"): %v`, err)
	}
	defer updateCollObj.Release()

	updateColl, err := updateCollObj.IDispatch(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer updateColl.Release()

	eula, err := updt.GetProperty("EulaAccepted")
	if err != nil {
		return fmt.Errorf(`updt.GetProperty("EulaAccepted"): %v`, err)
	}

	logger.Debugf("%s\n  - EulaAccepted: %v\n", title.Value(), eula.Value())
	if _, err := updateColl.CallMethod("Add", updt); err != nil {
		return fmt.Errorf(`updateColl.CallMethod("Add", updt): %v`, err)
	}

	if r.reportState(osconfigpb.Instance_APPLYING_PATCHES) {
		return nil
	}

	if err := packages.DownloadWUAUpdateCollection(session, updateColl); err != nil {
		return fmt.Errorf("DownloadWUAUpdateCollection error: %v", err)
	}

	if err := packages.InstallWUAUpdateCollection(session, updateColl); err != nil {
		return fmt.Errorf("InstallWUAUpdateCollection error: %v", err)
	}

	return nil
}

func installWUAUpdates(r *patchRun) error {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return err
	}
	defer ole.CoUninitialize()

	updateSessionObj, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		return fmt.Errorf(`oleutil.CreateObject("Microsoft.Update.Session"): %v`, err)
	}
	defer updateSessionObj.Release()

	session, err := updateSessionObj.IDispatch(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer session.Release()

	updts, err := packages.GetWUAUpdateCollection(session, "IsInstalled=0")
	if err != nil {
		return fmt.Errorf("GetWUAUpdateCollection error: %v", err)
	}

	countRaw, err := updts.GetProperty("Count")
	if err != nil {
		return err
	}
	count, _ := countRaw.Value().(int32)

	if count == 0 {
		logger.Infof("No Windows updates to install")
		return nil
	}

	logger.Debugf("DEBUG: %d Windows updates available\n", count)

	class := make(map[string]struct{})
	excludes := make(map[string]struct{})
	if r.Job.PatchConfig.WindowsUpdate != nil {
		for _, c := range r.Job.PatchConfig.WindowsUpdate.Classifications {
			sc, ok := classifications[c]
			if !ok {
				return fmt.Errorf("Unknown classification: %s", c)
			}
			class[sc] = struct{}{}
		}

		for _, e := range r.Job.PatchConfig.WindowsUpdate.Excludes {
			excludes[e] = struct{}{}
		}
	}

	for i := 0; i < int(count); i++ {
		updtRaw, err := updts.GetProperty("Item", i)
		if err != nil {
			return err
		}
		updt := updtRaw.ToIDispatch()
		defer updt.Release()

		if err := installUpdate(r, class, excludes, session, updt); err != nil {
			return fmt.Errorf(`installUpdate(class, excludes, updt): %v`, err)
		}
	}

	return nil
}

var classifications = map[osconfigpb.WindowsUpdateSettings_Classification]string{
	osconfigpb.WindowsUpdateSettings_CRITICAL:      "e6cf1350-c01b-414d-a61f-263d14d133b4",
	osconfigpb.WindowsUpdateSettings_SECURITY:      "0fa1201d-4330-4fa8-8ae9-b877473b6441",
	osconfigpb.WindowsUpdateSettings_DEFINITION:    "e0789628-ce08-4437-be74-2495b842f43b",
	osconfigpb.WindowsUpdateSettings_DRIVER:        "ebfc1fc5-71a4-4f7b-9aca-3b9a503104a0",
	osconfigpb.WindowsUpdateSettings_FEATURE_PACK:  "b54e7d24-7add-428f-8b75-90a396fa584f",
	osconfigpb.WindowsUpdateSettings_SERVICE_PACK:  "68c5b0a3-d1a6-4553-ae49-01d3a7827828",
	osconfigpb.WindowsUpdateSettings_TOOL:          "b4832bd8-e735-4761-8daf-37f882276dab",
	osconfigpb.WindowsUpdateSettings_UPDATE_ROLLUP: "28bc880e-0592-4cbf-8f95-c79b17911d5f",
	osconfigpb.WindowsUpdateSettings_UPDATE:        "cd5ffd1e-e932-4e3a-bf74-18bf0b1bbd83",
}

func runUpdates(r *patchRun) error {
	if err := installWUAUpdates(r); err != nil {
		return err
	}

	if packages.GooGetExists {
		if r.reportState(osconfigpb.Instance_APPLYING_PATCHES) {
			return nil
		}

		if err := packages.InstallGooGetUpdates(); err != nil {
			return err
		}
	}

	return nil
}

func rebootSystem() error {
	return exec.Command("shutdown", "/r", "/t", "00", "/f", "/d", "p:2:3").Run()
}
