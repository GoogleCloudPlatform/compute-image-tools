package importer

import (
	"bufio"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testWriteCloser struct {
	*bufio.Writer
	closeReturnVal error
}

func (wc testWriteCloser) Close() error {
	wc.Writer.Flush()
	return wc.closeReturnVal
}

type mockPopulator struct {
	project         string
	zone            string
	region          string
	scratchBucket   string
	storageLocation string
	err             error
}

func (m mockPopulator) PopulateMissingParameters(project *string, zone *string, region *string,
	scratchBucketGcsPath *string, file string, storageLocation *string) error {
	if m.err != nil {
		return m.err
	}
	if *project == "" {
		*project = m.project
	}
	if *zone == "" {
		*zone = m.zone
	}
	if *region == "" {
		*region = m.region
	}
	if *scratchBucketGcsPath == "" {
		*scratchBucketGcsPath = m.scratchBucket
	}
	if *storageLocation == "" {
		*storageLocation = m.storageLocation
	}
	return nil
}

func expectSuccessfulParse(t *testing.T, input ...string) *OneStepImportArguments {
	args := setUpArgs("", input...)
	importArgs, err := NewOneStepImportArguments(args)
	assert.NoError(t, err)
	err = importArgs.validate()

	assert.NoError(t, err)
	return importArgs
}

func setUpAWSArgs(requiredFlagToTest string, needsExport bool, args ...string) []string {
	args = setUpArgs("", args...)

	if requiredFlagToTest != awsAccessKeyIDFlag {
		args = append(args, "-aws_access_key_id=my-access-key")
	}
	if requiredFlagToTest != awsSecretAccessKeyFlag {
		args = append(args, "-aws_secret_access_key=my-secret-key")
	}
	if requiredFlagToTest != awsRegionFlag {
		args = append(args, "-aws_region=my-region")
	}
	if requiredFlagToTest != awsSessionTokenFlag {
		args = append(args, "-aws_session_token=my-token")
	}

	if needsExport {
		if requiredFlagToTest != awsAMIIDFlag {
			args = append(args, "-aws_ami_id=my-ami-id")
		}
		if requiredFlagToTest != awsExportLocationFlag {
			args = append(args, "-aws_export_location=s3://bucket")
		}
	} else {
		if requiredFlagToTest != awsExportedAMIPathFlag {
			args = append(args, "-aws_exported_ami_path=s3://bucket/object")
		}
	}
	return args
}

func getAWSImportArgs(args []string) *awsImportArguments {
	importerArgs, _ := NewOneStepImportArguments(args)
	return newAWSImportArguments(importerArgs)
}
