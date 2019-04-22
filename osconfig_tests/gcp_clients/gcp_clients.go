package gcpclients

import (
	"cloud.google.com/go/storage"
	"context"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/config"
	"google.golang.org/api/option"
)

var (
	storageClient  *storage.Client
	osconfigClient *osconfig.Client
)

func GetStorageClient(ctx context.Context) (*storage.Client, error) {
	if storageClient != nil {
		return storageClient, nil
	}

	return storage.NewClient(ctx, option.WithCredentialsFile(config.OauthPath()))
}

func GetOsConfigClient(ctx context.Context) (*osconfig.Client, error) {
	if osconfigClient != nil {
		return osconfigClient, nil
	}

	return osconfig.NewClient(ctx, option.WithEndpoint(config.SvcEndpoint()), option.WithCredentialsFile(config.OauthPath()))
}
