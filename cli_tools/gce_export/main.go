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
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/dustin/go-humanize"
	gzip "github.com/klauspost/pgzip"
	"google.golang.org/api/option"
)

var (
	disk      = flag.String("disk", "", "disk to export, on linux this would be something like '/dev/sda', and on Windows '\\\\.\\PhysicalDrive0'")
	localPath = flag.String("local_path", "", "local path to store the image file on disk, on linux this would be something like '/dev/sdb/buffer', and on Windows '\\\\.\\PhysicalDrive1\buffer'. ")
	gcsPath   = flag.String("gcs_path", "", "GCS path to upload the image to, gs://my-bucket/image.tar.gz")
	oauth     = flag.String("oauth", "", "path to oauth json file")
	licenses  = flag.String("licenses", "", "comma delimited list of licenses to add to the image")
	noconfirm = flag.Bool("y", false, "skip confirmation")
	level     = flag.Int("level", 3, "level of compression from 1-9, 1 being best speed, 9 being best compression")

	gsRegex = regexp.MustCompile(`^gs://([a-z0-9][-_.a-z0-9]*)/(.+)$`)
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

	if *localPath != "" && *gcsPath != "" {
		log.Fatal("-local_path and -gcs_path can't be both specified")
	}

	if *localPath == "" && *gcsPath == "" {
		log.Fatal("-local_path or -gcs_path must be provided")
	}

	if *disk == "" {
		log.Fatal("The flag -disk must be provided")
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

	writer, targetPath := createWriter()
	createGzipFile(file, size, writer, targetPath)
}

func createWriter() (io.WriteCloser, string) {
	var w io.WriteCloser
	var p string
	if *gcsPath != "" {
		bkt, obj, err := storageutils.SplitGCSPath(*gcsPath)
		if err != nil {
			log.Fatal(err)
		}

		ctx := context.Background()
		client, err := storage.NewClient(ctx, option.WithServiceAccountFile(*oauth))
		if err != nil {
			log.Fatal(err)
		}
		w = client.Bucket(bkt).Object(obj).NewWriter(ctx)
		p = fmt.Sprintf("gs://%s/%s", bkt, obj)
	} else {
		bufferFile, err := os.Create(*localPath)
		if err != nil {
			log.Fatal(err)
		}
		bw := bufio.NewWriter(bufferFile)
		w = &FileWriter{bw}
		p = *localPath
	}

	return w, p
}

func createGzipFile(file *os.File, size int64, writer io.WriteCloser, targetPath string) {
	up := progress{}
	gw, err := gzip.NewWriterLevel(io.MultiWriter(&up, writer), *level)
	if err != nil {
		log.Fatal(err)
	}
	rp := progress{}
	tw := tar.NewWriter(io.MultiWriter(&rp, gw))

	ls := splitLicenses(*licenses)
	fmt.Printf("GCEExport: Disk %s is %s, compressed size will most likely be much smaller.\n", *disk, humanize.IBytes(uint64(size)))
	if ls != nil {
		fmt.Printf("GCEExport: Exporting disk with licenses %q to %s.\n", ls, targetPath)
	} else {
		fmt.Printf("GCEExport: Exporting disk to %s.\n", targetPath)
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

	fmt.Println("GCEExport: Beginning export...")
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
			Name:   "manifest.json",
			Mode:   0600,
			Size:   int64(len(body)),
			Format: tar.FormatGNU,
		}); err != nil {
			log.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			log.Fatal(err)
		}
	}
	if err := tw.WriteHeader(&tar.Header{
		Name:   "disk.raw",
		Mode:   0600,
		Size:   size,
		Format: tar.FormatGNU,
	}); err != nil {
		log.Fatal(err)
	}

	// This function only serves to update progress for the user.
	go func() {
		time.Sleep(5 * time.Second)
		var oldUpload int64
		var oldRead int64
		var oldSince int64
		totalSize := humanize.IBytes(uint64(size))
		for {
			since := int64(time.Since(start).Seconds())
			diskSpd := humanize.IBytes(uint64((rp.total - oldRead) / (since - oldSince)))
			upldSpd := humanize.IBytes(uint64((up.total - oldUpload) / (since - oldSince)))
			uploadTotal := humanize.IBytes(uint64(up.total))
			readTotal := humanize.IBytes(uint64(rp.total))
			fmt.Printf("GCEExport: Read %s of %s (%s/sec),", readTotal, totalSize, diskSpd)
			fmt.Printf(" total written size: %s (%s/sec)\n", uploadTotal, upldSpd)
			oldUpload = up.total
			oldRead = rp.total
			oldSince = since
			time.Sleep(45 * time.Second)
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

	if err := writer.Close(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("GCEExport: Finished export in ", time.Since(start))
}
