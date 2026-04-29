//  Copyright 2018 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//	  http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package main

import (
	"bytes"
	"fmt"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func getItemProperties(item *ole.IDispatch) ([]string, error) {
	properties := make([]string, 0)

	propsRaw, err := oleutil.GetProperty(item, "Properties_")
	if err != nil {
		return properties, err
	}
	props := propsRaw.ToIDispatch()
	defer props.Release()

	err = oleutil.ForEach(props, func(v *ole.VARIANT) error {
		c := v.ToIDispatch()
		p, err := oleutil.GetProperty(c, "Name")
		if err != nil {
			return err
		}
		properties = append(properties, p.ToString())
		return nil
	})
	return properties, err
}

func printWmiObjects(class string, namespace string) (string, error) {
	ole.CoInitialize(0)
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return "", err
	}
	defer unknown.Release()

	wmi, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return "", err
	}
	defer wmi.Release()

	if namespace == "" {
		namespace = `root\default`
	}
	serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer", nil, namespace)
	if err != nil {
		return "", err
	}
	service := serviceRaw.ToIDispatch()
	defer service.Release()

	query := fmt.Sprintf("SELECT * FROM %s", class)
	resultRaw, err := oleutil.CallMethod(service, "ExecQuery", query)
	if err != nil {
		return "", err
	}
	items := resultRaw.ToIDispatch()
	defer items.Release()

	// Format the items into a readable list
	var bfr bytes.Buffer
	var properties []string
	err = oleutil.ForEach(items, func(itemRaw *ole.VARIANT) error {
		item := itemRaw.ToIDispatch()
		defer item.Release()

		// Read a list of the class's properties from the first item
		if properties == nil {
			properties, err = getItemProperties(item)
			if err != nil {
				return err
			}
		}

		bfr.WriteString("\r\n\r\n")
		for _, property := range properties {
			itemProp, err := oleutil.GetProperty(item, property)
			if err != nil {
				return err
			}
			p := itemProp.Value()
			if p == nil {
				p = ""
			}
			bfr.WriteString(fmt.Sprintf("%s: %v\r\n", property, p))
		}
		return nil
	})
	return bfr.String(), err
}
