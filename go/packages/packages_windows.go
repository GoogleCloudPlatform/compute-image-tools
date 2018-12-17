/*
Copyright 2017 Google Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package packages

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/StackExchange/wmi"
	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

var (
	googetDir = os.Getenv("GooGetRoot")
	googet    = filepath.Join(googetDir, "googet.exe")

	googetUpdateArgs         = []string{"-noconfirm", "update"}
	googetUpdateQueryArgs    = []string{"update"}
	googetInstalledQueryArgs = []string{"installed"}
	googetInstallArgs        = []string{"-noconfirm", "install"}
	googetRemoveArgs         = []string{"-noconfirm", "remove"}
)

func init() {
	GooGetExists = exists(googet)
}

// GetPackageUpdates gets available package updates GooGet as well as any
// available updates from Windows Update Agent.
func GetPackageUpdates() (Packages, []string) {
	var pkgs Packages
	var errs []string

	if GooGetExists {
		if googet, err := googetUpdates(); err != nil {
			msg := fmt.Sprintf("error listing googet updates: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.GooGet = googet
		}
	}
	if wua, err := wuaUpdates("IsInstalled=0"); err != nil {
		msg := fmt.Sprintf("error listing installed Windows updates: %v", err)
		DebugLogger.Println("Error:", msg)
		errs = append(errs, msg)
	} else {
		pkgs.WUA = wua
	}
	return pkgs, errs
}

func googetUpdates() ([]PkgInfo, error) {
	out, err := run(exec.Command(googet, googetUpdateQueryArgs...))
	if err != nil {
		return nil, err
	}

	/*
	   Searching for available updates...
	   foo.noarch, 3.5.4@1 --> 3.6.7@1 from repo
	   ...
	   Perform update? (y/N):
	*/
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	var pkgs []PkgInfo
	for _, ln := range lines[1:] {
		pkg := strings.Fields(ln)
		if len(pkg) != 6 {
			continue
		}

		p := strings.Split(pkg[0], ".")
		if len(p) != 2 {
			DebugLogger.Printf("%s does not represent a package", ln)
			continue
		}
		pkgs = append(pkgs, PkgInfo{Name: p[0], Arch: strings.Trim(p[1], ","), Version: pkg[3]})
	}
	return pkgs, nil
}

// InstallGooGetPackages installs GooGet packages.
func InstallGooGetPackages(pkgs []string) error {
	args := append(googetInstallArgs, pkgs...)
	out, err := run(exec.Command(googet, args...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf("  %s\n", s)
	}
	DebugLogger.Printf("GooGet install output:\n%s", msg)
	return nil
}

// RemoveGooGetPackages installs GooGet packages.
func RemoveGooGetPackages(pkgs []string) error {
	args := append(googetRemoveArgs, pkgs...)
	out, err := run(exec.Command(googet, args...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf("  %s\n", s)
	}
	DebugLogger.Printf("GooGet remove output:\n%s", msg)
	return nil
}

// InstallGooGetUpdates installs all available GooGet updates.
func InstallGooGetUpdates() error {
	out, err := run(exec.Command(googet, googetUpdateArgs...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf("  %s\n", s)
	}
	DebugLogger.Printf("GooGet update output:\n%s", msg)
	return nil
}

func installedGooGetPackages() ([]PkgInfo, error) {
	out, err := run(exec.Command(googet, googetInstalledQueryArgs...))
	if err != nil {
		return nil, err
	}

	/*
	   Installed Packages:
	   foo.x86_64 1.2.3@4
	   bar.noarch 1.2.3@4
	   ...
	*/
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	if len(lines) <= 1 {
		DebugLogger.Println("No packages GooGet installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines[1:] {
		pkg := strings.Fields(ln)
		if len(pkg) != 2 {
			DebugLogger.Printf("%s does not represent a GooGet package", ln)
			continue
		}

		p := strings.Split(pkg[0], ".")
		if len(p) != 2 {
			DebugLogger.Printf("%s does not represent a GooGet package", ln)
			continue
		}

		pkgs = append(pkgs, PkgInfo{Name: p[0], Arch: p[1], Version: pkg[1]})
	}
	return pkgs, nil
}

type (
	IUpdateSession    = *ole.IDispatch
	IUpdateCollection = *ole.VARIANT
)

func getStringSlice(dis *ole.IDispatch) ([]string, error) {
	countRaw, err := dis.GetProperty("Count")
	if err != nil {
		return nil, err
	}
	count, _ := countRaw.Value().(int32)

	if count == 0 {
		return nil, nil
	}

	var ss []string
	for i := 0; i < int(count); i++ {
		item, err := dis.GetProperty("Item", i)
		if err != nil {
			return nil, err
		}

		ss = append(ss, item.ToString())
	}
	return ss, nil
}

func getCategories(cat *ole.IDispatch) ([]string, []string, error) {
	countRaw, err := cat.GetProperty("Count")
	if err != nil {
		return nil, nil, err
	}
	count, _ := countRaw.Value().(int32)

	if count == 0 {
		return nil, nil, nil
	}

	var cns, cids []string
	for i := 0; i < int(count); i++ {
		itemRaw, err := cat.GetProperty("Item", i)
		if err != nil {
			return nil, nil, err
		}
		item := itemRaw.ToIDispatch()
		defer item.Release()

		name, err := item.GetProperty("Name")
		if err != nil {
			return nil, nil, err
		}

		categoryID, err := item.GetProperty("CategoryID")
		if err != nil {
			return nil, nil, err
		}

		cns = append(cns, name.ToString())
		cids = append(cids, categoryID.ToString())
	}
	return cns, cids, nil
}

// wuaUpdates queries the Windows Update Agent API searcher with the provided query.
func wuaUpdates(query string) ([]WUAPackage, error) {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return nil, err
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		return nil, fmt.Errorf("error creating Microsoft.Update.Session object: %v", err)
	}
	defer unknown.Release()

	session, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, err
	}
	defer session.Release()

	updtsRaw, err := GetWUAUpdateCollection(session, query)
	if err != nil {
		return nil, err
	}

	updts := updtsRaw.ToIDispatch()

	count, err := updts.GetProperty("Count")
	if err != nil {
		return nil, err
	}
	updtCnt, _ := count.Value().(int32)

	if updtCnt == 0 {
		return nil, nil
	}

	DebugLogger.Printf("%d WUA updates available", updtCnt)

	var packages []WUAPackage
	for i := 0; i < int(updtCnt); i++ {
		updtRaw, err := updts.GetProperty("Item", i)
		if err != nil {
			return nil, err
		}

		updt := updtRaw.ToIDispatch()
		defer updt.Release()

		title, err := updt.GetProperty("Title")
		if err != nil {
			return nil, err
		}

		description, err := updt.GetProperty("Description")
		if err != nil {
			return nil, err
		}

		kbArticleIDsRaw, err := updt.GetProperty("KBArticleIDs")
		if err != nil {
			return nil, err
		}
		kbArticleIDs, err := getStringSlice(kbArticleIDsRaw.ToIDispatch())
		if err != nil {
			return nil, err
		}

		categoriesRaw, err := updt.GetProperty("Categories")
		if err != nil {
			return nil, err
		}
		categories, categoryIDs, err := getCategories(categoriesRaw.ToIDispatch())
		if err != nil {
			return nil, err
		}

		supportURL, err := updt.GetProperty("SupportURL")
		if err != nil {
			return nil, err
		}

		lastDeploymentChangeTimeRaw, err := updt.GetProperty("LastDeploymentChangeTime")
		if err != nil {
			return nil, err
		}
		lastDeploymentChangeTime, err := ole.GetVariantDate(uint64(lastDeploymentChangeTimeRaw.Val))
		if err != nil {
			return nil, err
		}

		identityRaw, err := updt.GetProperty("Identity")
		if err != nil {
			return nil, err
		}
		identity := identityRaw.ToIDispatch()
		defer updt.Release()

		revisionNumber, err := identity.GetProperty("RevisionNumber")
		if err != nil {
			return nil, err
		}

		updateID, err := identity.GetProperty("UpdateID")
		if err != nil {
			return nil, err
		}

		pkg := WUAPackage{
			Title:                    title.ToString(),
			Description:              description.ToString(),
			SupportURL:               supportURL.ToString(),
			KBArticleIDs:             kbArticleIDs,
			UpdateID:                 updateID.ToString(),
			Categories:               categories,
			CategoryIDs:              categoryIDs,
			RevisionNumber:           int32(revisionNumber.Val),
			LastDeploymentChangeTime: lastDeploymentChangeTime,
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

// DownloadWUAUpdateCollection downloads all updates in a IUpdateCollection
func DownloadWUAUpdateCollection(session IUpdateSession, updates IUpdateCollection) error {
	// returns IUpdateDownloader
	// https://docs.microsoft.com/en-us/windows/desktop/api/wuapi/nn-wuapi-iupdatedownloader
	downloaderRaw, err := session.CallMethod("CreateUpdateDownloader")
	if err != nil {
		return fmt.Errorf("error calling method CreateUpdateDownloader on IUpdateSession: %v", err)
	}
	downloader := downloaderRaw.ToIDispatch()
	defer downloaderRaw.Clear()

	if _, err := downloader.PutProperty("Updates", updates); err != nil {
		return fmt.Errorf("error calling PutProperty Updates on IUpdateDownloader: %v", err)
	}

	DebugLogger.Println("Downloading WUA updates")
	if _, err := downloader.CallMethod("Download"); err != nil {
		return fmt.Errorf("error calling method Download on IUpdateDownloader: %v", err)
	}
	return nil
}

// InstallWUAUpdateCollection installs all updates in a IUpdateCollection
func InstallWUAUpdateCollection(session IUpdateSession, updates IUpdateCollection) error {
	// returns IUpdateInstaller
	// https://docs.microsoft.com/en-us/windows/desktop/api/wuapi/nf-wuapi-iupdatesession-createupdateinstaller
	installerRaw, err := session.CallMethod("CreateUpdateInstaller")
	if err != nil {
		return fmt.Errorf("error calling method CreateUpdateInstaller on IUpdateSession: %v", err)
	}
	installer := installerRaw.ToIDispatch()
	defer installerRaw.Clear()

	if _, err := installer.PutProperty("Updates", updates); err != nil {
		return fmt.Errorf("error calling PutProperty Updates on IUpdateInstaller: %v", err)
	}

	DebugLogger.Println("Installing WUA updates")
	// TODO: Look into using the async methods and attempt to track/log progress.
	if _, err := installer.CallMethod("Install"); err != nil {
		return fmt.Errorf("error calling method Install on IUpdateInstaller: %v", err)
	}
	return nil
}

// GetWUAUpdateCollection queries the Windows Update Agent API searcher with the provided query
// and returns a IUpdateCollection.
func GetWUAUpdateCollection(session IUpdateSession, query string) (IUpdateCollection, error) {
	// returns IUpdateSearcher
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa386515(v=vs.85).aspx
	searcherRaw, err := session.CallMethod("CreateUpdateSearcher")
	if err != nil {
		return nil, err
	}
	searcher := searcherRaw.ToIDispatch()
	defer searcherRaw.Clear()

	DebugLogger.Printf("Querying Windows Update Agent Search with query %q.", query)
	// returns ISearchResult
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa386077(v=vs.85).aspx
	resultRaw, err := searcher.CallMethod("Search", query)
	if err != nil {
		return nil, fmt.Errorf("error calling method Search on IUpdateSearcher: %v", err)
	}
	result := resultRaw.ToIDispatch()
	defer resultRaw.Clear()

	// returns IUpdateCollection
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa386107(v=vs.85).aspx
	return result.GetProperty("Updates")
}

// GetInstalledPackages gets all installed GooGet packages and Windows updates.
// Windows updates are read from Windows Update Agent and Win32_QuickFixEngineering.
func GetInstalledPackages() (Packages, []string) {
	var pkgs Packages
	var errs []string

	if exists(googet) {
		if googet, err := installedGooGetPackages(); err != nil {
			msg := fmt.Sprintf("error listing installed googet packages: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.GooGet = googet
		}
	}

	if wua, err := wuaUpdates("IsInstalled=1"); err != nil {
		msg := fmt.Sprintf("error listing installed Windows updates: %v", err)
		DebugLogger.Println("Error:", msg)
		errs = append(errs, msg)
	} else {
		pkgs.WUA = wua
	}

	if qfe, err := quickFixEngineering(); err != nil {
		msg := fmt.Sprintf("error listing installed QuickFixEngineering updates: %v", err)
		DebugLogger.Println("Error:", msg)
		errs = append(errs, msg)
	} else {
		pkgs.QFE = qfe
	}

	return pkgs, errs
}

type win32_QuickFixEngineering struct {
	Caption, Description, HotFixID, InstalledOn string
}

// quickFixEngineering queries the wmi object win32_QuickFixEngineering for a list of installed updates.
func quickFixEngineering() ([]QFEPackage, error) {
	var updts []win32_QuickFixEngineering
	DebugLogger.Print("Querying WMI for installed QuickFixEngineering updates.")
	if err := wmi.Query(wmi.CreateQuery(&updts, ""), &updts); err != nil {
		return nil, err
	}
	var qfe []QFEPackage
	for _, update := range updts {
		qfe = append(qfe, QFEPackage{
			Caption:     update.Caption,
			Description: update.Description,
			HotFixID:    update.HotFixID,
			InstalledOn: update.InstalledOn,
		})
	}
	return qfe, nil
}
