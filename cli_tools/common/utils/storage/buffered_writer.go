//  Copyright 2020 Google Inc. All Rights Reserved.
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

package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	pathutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"google.golang.org/api/googleapi"
)

var gcsPermissionErrorRegExp = regexp.MustCompile(".*does not have storage.objects.* access to .*")

type gcsClient func(ctx context.Context, oauth string) (domain.StorageClientInterface, error)

var exit = func(code int) {
	os.Exit(code)
}

// BufferedWriter is responsible for multipart component upload while using a local buffer.
type BufferedWriter struct {
	// These fields are read only.
	cSize    int64
	prefix   string
	ctx      context.Context
	oauth    string
	client   gcsClient
	id       string
	bkt, obj string

	errLogPrefix string

	upload    chan string
	tmpObjs   []string
	tmpObjsMx sync.Mutex

	sync.Mutex
	sync.WaitGroup
	bytes int64
	part  int
	file  *os.File
}

// NewBufferedWriter creates a BufferedWriter
func NewBufferedWriter(ctx context.Context, size, workers int64, client gcsClient, oauth, prefix, bkt, obj, errLogPrefix string) *BufferedWriter {
	b := &BufferedWriter{
		cSize:  size / workers,
		prefix: prefix,
		id:     pathutils.RandString(5),

		errLogPrefix: errLogPrefix,

		upload: make(chan string),
		bkt:    bkt,
		obj:    obj,
		ctx:    ctx,
		oauth:  oauth,
		client: client,
	}
	for i := int64(0); i < workers; i++ {
		b.Add(1)
		go b.uploadWorker()
	}
	return b
}

func (b *BufferedWriter) addObj(obj string) {
	b.tmpObjsMx.Lock()
	b.tmpObjs = append(b.tmpObjs, obj)
	b.tmpObjsMx.Unlock()
}

func (b *BufferedWriter) uploadWorker() {
	defer b.Done()
	for in := range b.upload {
		for i := 1; ; i++ {
			err := func() error {
				client, err := b.client(b.ctx, b.oauth)
				if err != nil {
					return err
				}
				defer client.Close()

				file, err := os.Open(in)
				if err != nil {
					return err
				}
				defer file.Close()

				tmpObj := path.Join(b.obj, strings.TrimPrefix(in, b.prefix))
				b.addObj(tmpObj)
				dst := client.GetObject(b.bkt, tmpObj).NewWriter()
				if _, err := io.Copy(dst, file); err != nil {
					if io.EOF != err {
						return err
					}
				}
				return dst.Close()
			}()
			if err != nil {
				// Don't retry if permission error as it's not recoverable.
				gAPIErr, isGAPIErr := err.(*googleapi.Error)
				if isGAPIErr && gAPIErr.Code == 403 && gcsPermissionErrorRegExp.MatchString(gAPIErr.Message) {
					fmt.Printf("%v: %v\n", b.errLogPrefix, err)
					exit(2)
					break
				}

				fmt.Printf("Failed %v time(s) to upload '%v', error: %v\n", i, in, err)
				if i > 16 {
					fmt.Printf("%v: %v\n", b.errLogPrefix, err)
					log.Fatal(err)
				}

				fmt.Printf("Retrying upload '%v' after %v second(s)...\n", in, i)
				time.Sleep(time.Duration(1*i) * time.Second)
				continue
			}
			os.Remove(in)
			break
		}
	}
}

func (b *BufferedWriter) newChunk() error {
	fp := path.Join(b.prefix, fmt.Sprint(b.id, "_part", b.part))
	f, err := os.Create(fp)
	if err != nil {
		return err
	}

	b.bytes = 0
	b.file = f
	b.part++

	return nil
}

func (b *BufferedWriter) flush() error {
	if err := b.file.Close(); err != nil {
		return err
	}

	b.upload <- b.file.Name()
	return nil
}

// Close composes the objects and close buffered writer.
func (b *BufferedWriter) Close() error {
	if err := b.flush(); err != nil {
		return err
	}
	close(b.upload)
	b.Wait()

	client, err := b.client(b.ctx, b.oauth)
	if err != nil {
		return err
	}
	defer client.Close()

	// Compose the object.
	for i := 0; ; i++ {
		var objs []domain.StorageObject
		// Max 32 components in a single compose.
		l := math.Min(float64(32), float64(len(b.tmpObjs)))
		for _, obj := range b.tmpObjs[:int(l)] {
			objs = append(objs, client.GetObject(b.bkt, obj))
		}
		if len(objs) == 1 {
			if _, err := client.GetObject(b.bkt, b.obj).CopyFrom(objs[0]); err != nil {
				return err
			}
			err = objs[0].Delete()
			if err != nil {
				return err
			}
			break
		}
		newObj := client.GetObject(b.bkt, path.Join(b.obj, b.id+"_compose_"+strconv.Itoa(i)))
		b.tmpObjs = append([]string{newObj.ObjectName()}, b.tmpObjs[int(l):]...)
		if _, err := newObj.Compose(objs...); err != nil {
			return err
		}
		for _, o := range objs {
			err = o.Delete()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Write writes the passed in bytes to buffer.
func (b *BufferedWriter) Write(d []byte) (int, error) {
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
