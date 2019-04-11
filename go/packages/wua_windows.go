/*
Copyright 2019 Google Inc. All Rights Reserved.
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

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

type (
	IUpdateSession    = *ole.IDispatch
	IUpdateCollection = *ole.IDispatch
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

// WUAUpdates queries the Windows Update Agent API searcher with the provided query.
func WUAUpdates(query string) ([]WUAPackage, error) {
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

	updts, err := GetWUAUpdateCollection(session, query)
	if err != nil {
		return nil, err
	}

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
	updtsRaw, err := result.GetProperty("Updates")
	if err != nil {
		return nil, fmt.Errorf("error calling GetProperty Updates on ISearchResult: %v", err)
	}

	return updtsRaw.ToIDispatch(), nil
}
