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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/dustin/go-humanize"
	gzip "github.com/klauspost/pgzip"
	"google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"
)

const gcsChunkSize = 0

var (
	disk         = flag.String("disk", "", "disk to export, on linux this would be something like '/dev/sda', and on Windows '\\\\.\\PhysicalDrive1'")
	bufferPrefix = flag.String("buffer_prefix", "", "if set will use this local path as the local buffer prefix")
	gcsPath      = flag.String("gcs_path", "", "GCS path to upload the image to, gs://my-bucket/image.tar.gz")
	oauth        = flag.String("oauth", "", "path to oauth json file")
	licenses     = flag.String("licenses", "", "comma delimited list of licenses to add to the image")
	noconfirm    = flag.Bool("y", false, "skip confirmation")
	level        = flag.Int("level", 3, "level of compression from 1-9, 1 being best speed, 9 being best compression")
	bufferSize   = flag.String("buffer_size", "1GB", "")
	workers      = flag.Int("workers", 4, "")
)

// progress is a io.Writer that updates total in Write.
type progress struct {
	total int64
	lock  sync.Mutex
}

func (p *progress) Write(b []byte) (int, error) {
	p.lock.Lock()
	p.total += int64(len(b))
	p.lock.Unlock()
	return len(b), nil
}

func randString(n int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	letters := "bdghjlmnpqrstvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[gen.Int63()%int64(len(letters))]
	}
	return string(b)
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

type buffer struct {
	// These fields are read only.
	cSize    int64
	prefix   string
	ctx      context.Context
	id       string
	bkt, obj string

	upload    chan string
	tmpObjs   []string
	tmpObjsMx sync.Mutex

	sync.Mutex
	sync.WaitGroup
	bytes int64
	part  int
	file  *os.File
}

func (b *buffer) addObj(obj string) {
	b.tmpObjsMx.Lock()
	b.tmpObjs = append(b.tmpObjs, obj)
	b.tmpObjsMx.Unlock()
}

func (b *buffer) uploadWorker() {
	defer b.Done()
	for in := range b.upload {
		for i := 0; ; i++ {
			err := func() error {
				client, err := gcsClient(b.ctx)
				if err != nil {
					return err
				}
				defer client.Close()

				file, err := os.Open(in)
				if err != nil {
					return err
				}
				defer file.Close()
				defer os.Remove(in)

				tmpObj := path.Join(b.obj, strings.TrimPrefix(in, b.prefix))
				b.addObj(tmpObj)
				dst := client.Bucket(b.bkt).Object(tmpObj).NewWriter(b.ctx)
				dst.ChunkSize = gcsChunkSize
				if _, err := io.Copy(dst, file); err != nil {
					if io.EOF != err {
						return err
					}
				}

				return dst.Close()

			}()
			if err != nil {
				if i > 2 {
					log.Fatal(err)
				}
				continue
			}
			break
		}
	}
}

func (b *buffer) newChunk() error {
	fp := fmt.Sprint(b.prefix, b.id, "_part", b.part)
	f, err := os.Create(fp)
	if err != nil {
		return err
	}

	b.bytes = 0
	b.file = f
	b.part++

	return nil
}

func (b *buffer) flush() error {
	if err := b.file.Close(); err != nil {
		return err
	}

	b.upload <- b.file.Name()
	return nil
}

func (b *buffer) Close() error {
	if err := b.flush(); err != nil {
		return err
	}
	close(b.upload)
	b.Wait()

	client, err := gcsClient(b.ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// Compose the object.
	for i := 0; ; i++ {
		var objs []*storage.ObjectHandle
		l := math.Min(float64(30), float64(len(b.tmpObjs)))
		for _, obj := range b.tmpObjs[:int(l)] {
			objs = append(objs, client.Bucket(b.bkt).Object(obj))
		}
		if len(objs) == 1 {
			if _, err := client.Bucket(b.bkt).Object(b.obj).CopierFrom(objs[0]).Run(b.ctx); err != nil {
				return err
			}
			objs[0].Delete(b.ctx)
			break
		}
		newObj := client.Bucket(b.bkt).Object(path.Join(b.obj, "compose_"+strconv.Itoa(i)))
		b.tmpObjs = append([]string{newObj.ObjectName()}, b.tmpObjs[int(l):]...)
		if _, err := newObj.ComposerFrom(objs...).Run(b.ctx); err != nil {
			return err
		}
		for _, o := range objs {
			o.Delete(b.ctx)
		}
	}
	return nil
}

func (b *buffer) Write(d []byte) (int, error) {
	b.Lock()
	defer b.Unlock()

	if b.file == nil {
		if err := b.newChunk(); err != nil {
			return 0, err
		}
	}

	b.bytes += int64(len(d))
	if b.bytes >= b.cSize {
		if err := b.flush(); err != nil {
			return 0, err
		}
		if err := b.newChunk(); err != nil {
			return 0, err
		}
		b.bytes = int64(len(d))
	}
	n, err := b.file.Write(d)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func gcsClient(ctx context.Context) (*storage.Client, error) {
	//return storage.NewClient(ctx)
	baseTransport := &http.Transport{
		DisableKeepAlives:     false,
		MaxIdleConns:          0,
		MaxIdleConnsPerHost:   1000,
		MaxConnsPerHost:       0,
		IdleConnTimeout:       60 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	transport, err := htransport.NewTransport(ctx, baseTransport)
	if err != nil {
		return nil, err
	}
	return storage.NewClient(ctx, option.WithHTTPClient(&http.Client{Transport: transport}))
}

func newBuffer(ctx context.Context, size, workers int64, prefix, bkt, obj string) *buffer {
	b := &buffer{
		cSize:  size / workers,
		prefix: prefix,
		id:     randString(5),

		upload: make(chan string),
		bkt:    bkt,
		obj:    obj,
		ctx:    ctx,
	}
	for i := int64(0); i < workers; i++ {
		b.Add(1)
		go b.uploadWorker()
	}
	return b
}

func writeGzipProgress(start time.Time, size int64, rp, wp *progress) {
	time.Sleep(5 * time.Second)
	var oldUpload int64
	var oldRead int64
	var oldSince int64
	totalSize := humanize.IBytes(uint64(size))
	for {
		rp.lock.Lock()
		rpTotal := rp.total
		rp.lock.Unlock()
		wp.lock.Lock()
		wpTotal := wp.total
		wp.lock.Unlock()

		since := int64(time.Since(start).Seconds())
		diskSpd := humanize.IBytes(uint64((rpTotal - oldRead) / (since - oldSince)))
		upldSpd := humanize.IBytes(uint64((wpTotal - oldUpload) / (since - oldSince)))
		uploadTotal := humanize.IBytes(uint64(wpTotal))
		readTotal := humanize.IBytes(uint64(rpTotal))
		if readTotal == totalSize {
			return
		}
		fmt.Printf("GCEExport: Read %s of %s (%s/sec),", readTotal, totalSize, diskSpd)
		fmt.Printf(" total written size: %s (%s/sec)\n", uploadTotal, upldSpd)
		oldUpload = wpTotal
		oldRead = rpTotal
		oldSince = since
		time.Sleep(45 * time.Second)
	}
}

func gzipDisk(file *os.File, size int64, writer io.WriteCloser) error {
	wp := &progress{}
	gw, err := gzip.NewWriterLevel(io.MultiWriter(wp, writer), *level)
	if err != nil {
		return err
	}
	rp := &progress{}
	tw := tar.NewWriter(io.MultiWriter(rp, gw))

	ls := splitLicenses(*licenses)
	if ls != nil {
		fmt.Printf("GCEExport: Creating gzipped image of %q with licenses %q.\n", file.Name(), ls)
	} else {
		fmt.Printf("GCEExport: Creating gzipped image of %q.\n", file.Name())
	}
	start := time.Now()

	if ls != nil {
		type lsJSON struct {
			Licenses []string `json:"licenses"`
		}
		body, err := json.Marshal(lsJSON{Licenses: ls})
		if err != nil {
			return err
		}

		if err := tw.WriteHeader(&tar.Header{
			Name:   "manifest.json",
			Mode:   0600,
			Size:   int64(len(body)),
			Format: tar.FormatGNU,
		}); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			return err
		}
	}
	if err := tw.WriteHeader(&tar.Header{
		Name:   "disk.raw",
		Mode:   0600,
		Size:   size,
		Format: tar.FormatGNU,
	}); err != nil {
		return err
	}

	go writeGzipProgress(start, size, rp, wp)

	if _, err := io.CopyN(tw, file, size); err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}

	if err := gw.Close(); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	since := time.Since(start)
	spd := humanize.IBytes(uint64(size / int64(since.Seconds())))
	ratio := size / wp.total
	log.Printf("GCEExport: Finished creating gzipped image of %q in %s [%s/s] with a compression ratio of %d.", file.Name(), since, spd, ratio)

	return nil
}

func stream(ctx context.Context, src *os.File, size int64, prefix, bkt, obj string) error {
	fmt.Printf("GCEExport: Copying %q to gs://%s/%s.\n", src.Name(), bkt, obj)

	if prefix != "" {
		bs, err := humanize.ParseBytes(*bufferSize)
		if err != nil {
			return err
		}

		prefix, err := filepath.Abs(prefix)
		if err != nil {
			return err
		}
		buf := newBuffer(ctx, int64(bs), int64(*workers), prefix, bkt, obj)

		fmt.Printf("GCEExport: Using %q as the local buffer prefix, %s as the max buffer size, and %d as the number of workers.\n", prefix, humanize.IBytes(bs), *workers)
		return gzipDisk(src, size, buf)
	}

	client, err := gcsClient(ctx)
	if err != nil {
		return err
	}

	w := client.Bucket(bkt).Object(obj).NewWriter(ctx)
	w.ChunkSize = gcsChunkSize
	fmt.Println("GCEExport: No local cache set, streaming directly to GCS.")
	return gzipDisk(src, size, w)
}

func main() {
	flag.Parse()
	ctx := context.Background()

	if *gcsPath == "" {
		log.Fatal("The flag -gcs_path must be provided")
	}

	if *disk == "" {
		log.Fatal("The flag -disk must be provided")
	}

	bkt, obj, err := storageutils.SplitGCSPath(*gcsPath)
	if err != nil {
		log.Fatal(err)
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

	fmt.Printf("GCEExport: Disk %s is %s, compressed size will most likely be much smaller.\n", *disk, humanize.IBytes(uint64(size)))
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

	fmt.Println("GCEExport: Beginning export process...")
	start := time.Now()
	if err := stream(ctx, file, size, *bufferPrefix, bkt, obj); err != nil {
		log.Fatal(err)
	}

	fmt.Println("GCEExport: Finished export in ", time.Since(start))
}
