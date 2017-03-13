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

// export streams a local disk to a Google Compute Engine image file in a Google Cloud Storage bucket.
package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	humanize "github.com/dustin/go-humanize"
)

var (
	disk      = flag.String("disk", "", "disk to copy, on linux this would be something like '/dev/sda', and on Windows '\\\\.\\PhysicalDrive0'")
	bucket    = flag.String("bucket", "", "bucket to copy the image to")
	out       = flag.String("out", "image", "what to call the resultant image (.tar.gz will be appened)")
	licenses  = flag.String("licenses", "", "comma deliminated list of licenses to add to the image")
	noconfirm = flag.Bool("y", false, "skip confirmation")
)

// progress is a io.Writer that updates total in Write.
type progress struct {
	total int64
}

func (p *progress) Write(b []byte) (int, error) {
	p.total += int64(len(b))
	return len(b), nil
}

func splitLicenses(input string) []string {
	if input == "" {
		return nil
	}
	var ls []string
	for _, l := range strings.Split(input, ",") {
		ls = append(ls, l)
	}
	return ls
}

func main() {
	flag.Parse()

	if *bucket == "" {
		log.Fatalf("The flag -bucket must be provided")
	}

	if *disk == "" {
		log.Fatalf("The flag -disk must be provided")
	}

	file, err := os.Open(*disk)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	size, err := diskLength(file)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	name := *out + ".tar.gz"
	w := client.Bucket(*bucket).Object(name).NewWriter(ctx)
	up := progress{}
	gw := gzip.NewWriter(io.MultiWriter(&up, w))
	rp := progress{}
	tw := tar.NewWriter(io.MultiWriter(&rp, gw))

	ls := splitLicenses(*licenses)
	fmt.Printf("Disk %s is %s, compressed size will most likely be much smaller.\n", *disk, humanize.IBytes(uint64(size)))
	if ls != nil {
		fmt.Printf("Exporting disk with licenses %q to gs://%s/%s.\n", ls, *bucket, name)
	} else {
		fmt.Printf("Exporting disk to gs://%s/%s.\n", *bucket, name)
	}

	if !*noconfirm {
		fmt.Print("Continue? (y/N): ")
		var c string
		fmt.Scanln(&c)
		c = strings.ToLower(c)
		if c != "y" && c != "yes" {
			fmt.Println("Aborting")
			os.Exit(0)
		}
	}

	fmt.Println("Beginning copy...")
	start := time.Now()

	if ls != nil {
		type lsJSON struct {
			Licenses []string `json:"licenses"`
		}
		body, err := json.Marshal(lsJSON{Licenses: ls})
		if err != nil {
			log.Fatal(err)
		}

		if err := tw.WriteHeader(&tar.Header{
			Name: "manifest.json",
			Size: int64(len(body)),
		}); err != nil {
			log.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			log.Fatal(err)
		}
	}

	if err := tw.WriteHeader(&tar.Header{
		Name: "disk.raw",
		Size: size,
	}); err != nil {
		log.Fatal(err)
	}

	// This function only serves to update progress for the user.
	go func() {
		time.Sleep(5 * time.Second)
		var ou int64
		var or int64
		for {
			diskSpd := humanize.IBytes(uint64((rp.total - or) / 10))
			upldSpd := humanize.IBytes(uint64((up.total - ou) / 10))
			left := time.Since(start).Seconds() * (100 / (100 * (float64(rp.total) / float64(size))))

			fmt.Printf("Read %s of %s (%s/sec, ~%s left), ", humanize.IBytes(uint64(rp.total)), humanize.IBytes(uint64(size)), diskSpd, time.Duration(int64(left))*time.Second)
			fmt.Printf("total uploaded size: %s (%s/sec)\n", humanize.IBytes(uint64(up.total)), upldSpd)
			ou = up.total
			or = rp.total
			time.Sleep(30 * time.Second)
		}
	}()

	if _, err := io.CopyN(tw, file, size); err != nil {
		log.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		log.Fatal(err)
	}

	if err := gw.Close(); err != nil {
		log.Fatal(err)
	}

	if err := w.Close(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Finished export in", time.Since(start))
}
