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

package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dustin/go-humanize"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"
	imageImporter "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/importer"
)

// Make file paths mutable
var (
	WorkflowDir                = "daisy_workflows/image_import/"
	ImportWorkflow             = "import_image.wf.json"
	ImportFromImageWorkflow    = "import_from_image.wf.json"
	ImportAndTranslateWorkflow = "import_and_translate.wf.json"

	workers                  = runtime.NumCPU()
	gcsPermissionErrorRegExp = regexp.MustCompile(".*does not have storage.objects.create access to .*")
)

// Parameter key shared with other packages
const (
	ImageNameFlagKey = "image_name"
	ClientIDFlagKey  = "client_id"
)

const (
	downloadBufSize = "100MB"
	downloadBufNum  = 3
	uploadBufSize   = "500MB"
	logPrefix       = "[import-image]"
	letters         = "bdghjlmnpqrstvwxyz0123456789"
)

func randString(n int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[gen.Int63()%int64(len(letters))]
	}
	return string(b)
}

func gcsClient(ctx context.Context, oauth string) (*storage.Client, error) {
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
	return storage.NewClient(ctx, option.WithHTTPClient(&http.Client{Transport: transport}),
		option.WithCredentialsFile(oauth))
}

// Upload buffer writer
type bufferedWriter struct {
	// These fields are read only.
	cSize    int64
	prefix   string
	ctx      context.Context
	oauth    string
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
				client, err := gcsClient(b.ctx, b.oauth)
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
				dst := client.Bucket(b.bkt).Object(tmpObj).NewWriter(b.ctx)
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

	client, err := gcsClient(b.ctx, b.oauth)
	if err != nil {
		return err
	}
	defer client.Close()

	// Compose the object.
	for i := 0; ; i++ {
		var objs []*storage.ObjectHandle
		// Max 32 components in a single compose.
		l := math.Min(float64(32), float64(len(b.tmpObjs)))
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
		newObj := client.Bucket(b.bkt).Object(path.Join(b.obj, b.id+"_compose_"+strconv.Itoa(i)))
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

func newBuffer(ctx context.Context, oauth string, size, workers int64, prefix, bkt, obj string) *bufferedWriter {
	b := &bufferedWriter{
		cSize:  size / workers,
		prefix: prefix,
		id:     randString(5),

		upload: make(chan string),
		bkt:    bkt,
		obj:    obj,
		ctx:    ctx,
		oauth:  oauth,
	}
	for i := int64(0); i < workers; i++ {
		b.Add(1)
		go b.uploadWorker()
	}
	return b
}

func validateAndParseFlags(clientID string, imageName string, sourceFile string, sourceImage string, dataDisk bool, osID string, customTranWorkflow string, labels string) (
		string, string, map[string]string, error) {

	// call original validate, then do own validation
	return "", "", nil, nil
}

// Run runs import workflow.
func Run(clientID string, imageName string, dataDisk bool, osID string, customTranWorkflow string,
		noGuestEnvironment bool, family string, description string,
		network string, subnet string, zone string, timeout string, project *string,
		scratchBucketGcsPath string, oauth string, ce string, gcsLogsDisabled bool, cloudLogsDisabled bool,
		stdoutLogsDisabled bool, kmsKey string, kmsKeyring string, kmsLocation string, kmsProject string,
		noExternalIP bool, labels string, currentExecutablePath string, storageLocation string,
		uefiCompatible bool, awsImageId string, awsExportBucket string, awsExportFolder string,
		awsAccessKeyId string, awsSecrectAccessKey string, awsRegion string, awsRand string, awsExportTid string) (*daisy.Workflow, error) {

	log.SetPrefix(logPrefix + " ")

	skipAwsExport := awsRand != ""
	if !skipAwsExport {
		awsRand = string(rand.Int() % 1000000)
	}
	tmpFilePath := fmt.Sprintf("gs://fionaliu-daisy-bkt-us-east1/onestep-import/tmp-%v/tmp-%v.vmdk", awsRand, awsRand)

	// 0. aws2 configure
	err := configure(awsAccessKeyId, awsSecrectAccessKey, awsRegion)
	if err != nil {
		return nil, err
	}

	var exportedFilePath string
	if !skipAwsExport {
		// 1. export: aws ec2 export-image --image-id ami-0bdc89ef2ef39dd0a --disk-image-format VMDK --s3-export-location S3Bucket=dntczdx,S3Prefix=exports/
		//runCliTool("./gce_onestep_image_import", []string{""})
		exportedFilePath, err = exportAwsImage(awsImageId, awsExportBucket, awsExportFolder)
		if err != nil {
			return nil, err
		}
	} else {
		exportedFilePath = fmt.Sprintf("s3://%v/%v/%v.vmdk", awsExportBucket, awsExportFolder, awsExportTid)
	}

	awsExportKey := strings.TrimPrefix(exportedFilePath, "s3://"+awsExportBucket+"/")
	fileSize, err := getAwsFileSize(awsExportBucket, awsExportKey)
	if err != nil {
		return nil, err
	}
	// 2. copy: gsutil cp s3://dntczdx/exports/export-ami-0b768c1d619f93184.vmdk gs://tzz-noogler-3-daisy-bkt/amazon1.vmdk
	if err := copyToGcs(awsExportBucket, awsExportKey, fileSize, tmpFilePath, oauth); err != nil {
		return nil, err
	}

	// 3. call image importer
	log.Println("Starting image import to GCE...")
	return imageImporter.Run(clientID, imageName, dataDisk, osID, customTranWorkflow, tmpFilePath,
		"", noGuestEnvironment, family, description, network, subnet, zone, timeout,
		project, scratchBucketGcsPath, oauth, ce, gcsLogsDisabled, cloudLogsDisabled,
		stdoutLogsDisabled, kmsKey, kmsKeyring, kmsLocation, kmsProject, noExternalIP,
		labels, currentExecutablePath, storageLocation, uefiCompatible)
}

func getAwsFileSize(awsExportBucket string, awsExportKey string) (int64, error) {
	output, err := runCmdAndGetOutput("aws", []string{"s3api", "head-object", "--bucket", fmt.Sprintf("%v", awsExportBucket), "--key", fmt.Sprintf("%v", awsExportKey)})
	if err != nil {
		return 0, err
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(output, &resp); err != nil {
		return 0, err
	}
	fileSize := int64(resp["ContentLength"].(float64))
	if fileSize == 0 {
		return 0, fmt.Errorf("File is empty")
	}
	return fileSize, nil
}

func configure(awsAccessKeyId string, awsSecrectAccessKey string, awsRegion string) error {
	if err := runCmd("aws", []string{"configure", "set", "aws_access_key_id", awsAccessKeyId}); err != nil {
		return err
	}
	if err := runCmd("aws", []string{"configure", "set", "aws_secret_access_key", awsSecrectAccessKey}); err != nil {
		return err
	}
	if err := runCmd("aws", []string{"configure", "set", "region", awsRegion}); err != nil {
		return err
	}
	if err := runCmd("aws", []string{"configure", "set", "output", "json"}); err != nil {
		return err
	}
	return nil
}

type AwsTaskResponse struct {
	ExportImageTasks []AwsExportImageTasks `json:omitempty`
}

type AwsExportImageTasks struct {
	Status        string `json:omitempty`
	StatusMessage string `json:omitempty`
	Progress      string `json:omitempty`
}

func exportAwsImage(awsImageId string, awsExportBucket string, awsExportFolder string) (string, error) {
	output, err := runCmdAndGetOutput("aws", []string{"ec2", "export-image", "--image-id", awsImageId, "--disk-image-format", "VMDK", "--s3-export-location",
		fmt.Sprintf("S3Bucket=%v,S3Prefix=%v/", awsExportBucket, awsExportFolder)})
	if err != nil {
		return "", err
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(output, &resp); err != nil {
		return "", err
	}
	tid := resp["ExportImageTaskId"]
	if tid == "" {
		return "", fmt.Errorf("Empty task id returned.")
	}
	log.Println("export image task id: ", tid)

	//repeat query
	for {
		// aws ec2 describe-export-image-tasks --export-image-task-ids export-ami-0f7900141ff1f3caa
		output, err = runCmdAndGetOutputWithoutLog("aws", []string{"ec2", "describe-export-image-tasks", "--export-image-task-ids", fmt.Sprintf("%v", tid)})
		if err != nil {
			return "", err
		}
		var taskResp AwsTaskResponse
		if err := json.Unmarshal(output, &taskResp); err != nil {
			return "", err
		}
		if len(taskResp.ExportImageTasks) != 1 {
			return "", fmt.Errorf("Unexpected response of describe-export-image-tasks.")
		}
		log.Println(fmt.Sprintf("AWS export task status: %v, status message: %v, progress: %v", taskResp.ExportImageTasks[0].Status, taskResp.ExportImageTasks[0].StatusMessage, taskResp.ExportImageTasks[0].Progress))

		if taskResp.ExportImageTasks[0].Status != "active" {
			if taskResp.ExportImageTasks[0].Status != "completed" {
				return "", fmt.Errorf("AWS export task wasn't completed successfully.")
			}
			break
		}
		time.Sleep(time.Millisecond * 3000)
	}
	log.Println("AWS export task is completed!")

	return fmt.Sprintf("s3://%v/%v/%v.vmdk", awsExportBucket, awsExportFolder, tid), nil
}

func stream(awsBucket string, awsKey string, size int64, writer *bufferedWriter) (error) {
	log.Println("Downloading from s3 ...")
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	client := s3.New(sess)
	wg := new(sync.WaitGroup)
	sem := make(chan struct{}, downloadBufNum)
	var dmutex sync.Mutex
	readSize, err := humanize.ParseBytes(downloadBufSize)
	if err != nil {
		return err
	}
	readers := int(math.Ceil(float64(size) / float64(readSize)))

	log.Println("readers:", readers)
	start := time.Now()
	for i := 0; i < readers; i++ {
		sem <- struct{}{}
		wg.Add(1)
		offset := i * int(readSize)
		readRange := strconv.Itoa(offset) + "-" + strconv.Itoa(offset+int(readSize)-1)

		res, err := client.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(awsBucket),
			Key:    aws.String(awsKey),
			Range:  aws.String("bytes=" + readRange),
		})

		if err != nil {
			log.Println(fmt.Sprintf("Error in downloading from bucket %v, key %v: %v \n", awsBucket, awsKey, err))
			return err
		}
		log.Println("downloaded")
		go func(writer *bufferedWriter, res *s3.GetObjectOutput) {
			defer wg.Done()
			dmutex.Lock()
			defer dmutex.Unlock()
			defer res.Body.Close()
			io.Copy(writer, res.Body)
			<-sem
			log.Println("uploaded")
		}(writer, res)
	}

	wg.Wait()
	if err := writer.Close(); err != nil {
		return err
	}
	since := time.Since(start)
	log.Printf("Finished transferring in %s.", since)
	return nil
}

func copyToGcs(awsBucket string, awsKey string, size int64, gcsFilePath string, oauth string) error {
	log.Println("Copying from ec2 to s3...")
	bs, err := humanize.ParseBytes(uploadBufSize)
	if err != nil {
		return err
	}

	ctx := context.Background()
	bkt, obj, err := storageutils.GetGCSObjectPathElements(gcsFilePath)
	if err != nil {
		log.Fatal(err)
	}
	writer := newBuffer(ctx, oauth, int64(bs), int64(workers), "/tmp", bkt, obj)
	err = stream(awsBucket, awsKey, size, writer)
	if err != nil {
		return err
	}
	log.Println("Copied.")

	return nil

	//   gsutil cp tmp gs://tzz-noogler-3-daisy-bkt/amazon1.vmdk
	// 	log.Println("Copying from cache to gcs...")
	// 	err = runCmd("gsutil", []string{"cp", awsFilePath, gcsFilePath})
	// 	if err != nil {
	// 		return err
	// 	}
	// 	log.Println("Copied.")
	//
	// 	return nil
}

func runCmd(cmdString string, args []string) error {
	log.Printf("Running command: '%s %s'", cmdString, strings.Join(args, " "))
	cmd := exec.Command(cmdString, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCmdAndGetOutput(cmdString string, args []string) ([]byte, error) {
	log.Printf("Running command: '%s %s'", cmdString, strings.Join(args, " "))
	return runCmdAndGetOutputWithoutLog(cmdString, args)
}

func runCmdAndGetOutputWithoutLog(cmdString string, args []string) ([]byte, error) {
	output, err := exec.Command(cmdString, args...).Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}
