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
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

// DisksCheck performs disk configuration checking:
// - finding the root filesystem partition
// - checking if the device is MBR
// - check for GRUB
// - warning for any mount points from partitions from other devices
type DisksCheck struct {
	getMBROverride func(devName string) ([]byte, error)
	lsblkOverride  func() (string, error)
}

func (c *DisksCheck) getMBR(devName string) ([]byte, error) {
	devPath := filepath.Join("/dev", devName)
	f, err := os.Open(devPath)
	if err != nil {
		return nil, err
	}
	data := make([]byte, MBRSIZE)
	_, err = f.Read(data)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %v", devPath, err)
	}
	return data, nil
}

func (c *DisksCheck) lsblk() (string, error) {
	cmd := exec.Command("lsblk", "-i")
	out, err := cmd.Output()
	if err != nil {
		exitErr := err.(*exec.ExitError)
		return "", fmt.Errorf("lsblk: %v, stderr: %s", err, exitErr.Stderr)
	}
	return string(out), nil
}

func (c *DisksCheck) GetName() string {
	return "Disks Check"
}

func (c *DisksCheck) Run() (*Report, error) {
	r := &Report{Name: c.GetName()}
	var out string
	var err error
	if c.lsblkOverride != nil {
		out, err = c.lsblkOverride()
	} else {
		out, err = c.lsblk()
	}
	if err != nil {
		return nil, err
	}

	l := lsblkParse(out)

	r.Info(fmt.Sprintf("`lsblk -i` results:\n%s", l.raw))
	if len(l.devRows) != 1 {
		r.Warn("translation only supports single block device. Will attempt to determine if translation is possible...")
	}

	if l.rootDev == "" {
		r.Fatal("root filesystem partition not found on any block devices.")
		return r, nil
	} else {
		r.Info(fmt.Sprintf("root filesystem on device %q partition %q", l.rootDev, l.rootPart))
	}

	for dev, rows := range l.devRows {
		for _, row := range rows {
			if row["MOUNTPOINT"] != "" && dev != l.rootDev {
				format := "partition %s (%s) is not on the root device %s and will OMITTED from translation."
				r.Warn(fmt.Sprintf(format, row["NAME"], row["MOUNTPOINT"], l.rootDev))
			}
		}
	}

	// MBR checking.
	var mbrData []byte
	if c.getMBROverride != nil {
		mbrData, err = c.getMBROverride(l.rootDev)
	} else {
		mbrData, err = c.getMBR(l.rootDev)
	}
	if err != nil {
		return nil, err
	}
	if mbrData[510] != 0x55 || mbrData[511] != 0xAA {
		r.Fatal("root filesystem device is NOT MBR")
	} else {
		r.Info("root filesystem device is MBR.")
	}
	if !bytes.Contains(mbrData, []byte("GRUB")) {
		r.Fatal("GRUB not detected on MBR")
	} else {
		r.Info("GRUB found in root filesystem device MBR")
	}

	return r, nil
}

type lsblk struct {
	schema   []lsblkSchemaCol
	devRows  map[string][]lsblkRow
	rootDev  string
	rootPart string
	rows     []lsblkRow
	raw      string
}

type lsblkRow map[string]string

type lsblkSchemaCol struct {
	name       string
	start, end int
}

func (l *lsblk) parse(content string) {
	l.raw = content
	lines := strings.Split(content, "\n")

	// Block device and partition mount checking.
	// devs is a list of devices/partitions. Map of device name -> device/partitions data
	// Slice schema: NAME, MAJ:MIN, SIZE, RO, TYPE, MOUNTPOINT.
	l.schema = lsblkSchema(lines[0])
	var curDev string
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		row := lsblkRow{}
		for i, col := range l.schema {
			var val string
			if col.start >= len(line) {
				val = ""
			} else if col.end >= len(line) {
				val = line[col.start:]
			} else {
				val = line[col.start:col.end]
			}
			val = strings.TrimSpace(val)
			if i == 0 {
				val = strings.TrimLeft(val, "|`- ")
			}
			row[string(col.name)] = val
		}
		if unicode.IsLetter(rune(line[0])) {
			curDev = row["NAME"]
		}
		if row["MOUNTPOINT"] == "/" {
			l.rootDev = curDev
			l.rootPart = row["NAME"]
		}

		if l.devRows == nil {
			l.devRows = map[string][]lsblkRow{curDev: {row}}
		} else if _, ok := l.devRows[curDev]; !ok {
			l.devRows[curDev] = []lsblkRow{row}
		} else {
			l.devRows[curDev] = append(l.devRows[curDev], row)
		}
		l.rows = append(l.rows, row)
	}
}

func lsblkParse(output string) lsblk {
	l := lsblk{}
	l.parse(output)
	return l
}

func lsblkSchema(header string) []lsblkSchemaCol {
	var cols []lsblkSchemaCol
	start := 0
	for i, c := range header {
		isSpace := unicode.IsSpace(c)
		next := i + 1
		if next == len(header) || isSpace && !unicode.IsSpace(rune(header[next])) {
			// Start of next word (or end of header). Finish this current column. Set start to next word.
			name := strings.TrimSpace(header[start:next])
			col := lsblkSchemaCol{name, start, next}
			cols = append(cols, col)
			start = next
		}
	}
	return cols
}
