//  Copyright 2019 Google Inc. All Rights Reserved.
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
	"math/rand"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
)

var gcsPermissionErrorRegExp = regexp.MustCompile(".*does not have storage.objects.create access to .*")

const letters = "bdghjlmnpqrstvwxyz0123456789"

func randString(n int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[gen.Int63()%int64(len(letters))]
	}
	return string(b)
}

type bufferedWriter struct {
	// These fields are read only.
	cSize    int64
	prefix   string
	ctx      context.Context
	clientMx sync.Mutex
	client   *storage.Client
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

func (b *bufferedWriter) addObj(obj string) {
	b.tmpObjsMx.Lock()
	b.tmpObjs = append(b.tmpObjs, obj)
	b.tmpObjsMx.Unlock()
}

func (b *bufferedWriter) uploadWorker() {
	defer b.Done()
	for in := range b.upload {
		for i := 1; ; i++ {
			err := func() error {
				file, err := os.Open(in)
				if err != nil {
					return err
				}
				defer file.Close()

				tmpObj := path.Join(b.obj, strings.TrimPrefix(in, b.prefix))
				b.addObj(tmpObj)

				b.clientMx.Lock()
				dst := b.client.Bucket(b.bkt).Object(tmpObj).NewWriter(b.ctx)
				b.clientMx.Unlock()

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
					fmt.Printf("GCEExport: %v", err)
					os.Exit(2)
				}

				fmt.Printf("Failed %v time(s) to upload '%v', error: %v\n", i, in, err)
				if i > 16 {
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

func (b *bufferedWriter) newChunk() error {
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

func (b *bufferedWriter) flush() error {
	if err := b.file.Close(); err != nil {
		return err
	}

	b.upload <- b.file.Name()
	return nil
}

func (b *bufferedWriter) Close() error {
	if err := b.flush(); err != nil {
		return err
	}
	close(b.upload)
	b.Wait()

	b.clientMx.Lock()
	defer b.clientMx.Unlock()
	defer b.client.Close()

	// Compose the object.
	for i := 0; ; i++ {
		var objs []*storage.ObjectHandle
		// Max 32 components in a single compose.
		l := math.Min(float64(32), float64(len(b.tmpObjs)))
		for _, obj := range b.tmpObjs[:int(l)] {
			objs = append(objs, b.client.Bucket(b.bkt).Object(obj))
		}
		if len(objs) == 1 {
			if _, err := b.client.Bucket(b.bkt).Object(b.obj).CopierFrom(objs[0]).Run(b.ctx); err != nil {
				return err
			}
			objs[0].Delete(b.ctx)
			break
		}
		newObj := b.client.Bucket(b.bkt).Object(path.Join(b.obj, b.id+"_compose_"+strconv.Itoa(i)))
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

func (b *bufferedWriter) Write(d []byte) (int, error) {
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

func NewBuffer(ctx context.Context, client *storage.Client, size, workers int64, prefix, bkt, obj string) *bufferedWriter {
	b := &bufferedWriter{
		cSize:  size / workers,
		prefix: prefix,
		id:     randString(5),

		upload: make(chan string),
		bkt:    bkt,
		obj:    obj,
		ctx:    ctx,
		client: client,
	}
	for i := int64(0); i < workers; i++ {
		b.Add(1)
		go b.uploadWorker()
	}
	return b
}
