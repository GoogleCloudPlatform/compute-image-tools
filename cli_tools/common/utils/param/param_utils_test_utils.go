package paramutils

import (
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"testing"
)

func RunTestPopulateMissingParametersDoesNotChangeProvidedScratchBucketAndUsesItsRegion(
	t *testing.T, zone *string, region *string, scratchBucketGcsPath *string, file *string,
	project *string, expectedBucketName string, expectedRegion string, expectedZone string) error {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockZoneRetriever.EXPECT().GetZone(expectedRegion, *project).Return(expectedZone, nil).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs(expectedBucketName).Return(&storage.BucketAttrs{Location: expectedRegion}, nil)

	return PopulateMissingParameters(project, zone, region, scratchBucketGcsPath, *file,
		mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)
}

func RunTestPopulateMissingParametersCreatesScratchBucketIfNotProvided(
	t *testing.T, zone *string, region *string, scratchBucketGcsPath *string, file *string,
	project *string, expectedProject string, expectedBucket string, expectedRegion string,
	expectedZone string) error {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)

	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().
		CreateScratchBucket(*file, *project).
		Return(expectedBucket, expectedRegion, nil).
		Times(1)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockZoneRetriever.EXPECT().GetZone(expectedRegion, expectedProject).Return(expectedZone, nil).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	return PopulateMissingParameters(project, zone, region, scratchBucketGcsPath,
		*file, mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)
}

func RunTestPopulateProjectIfMissingProjectPopulatedFromGCE(
	t *testing.T, project *string, expectedProject string) error {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return(expectedProject, nil)

	return PopulateProjectIfMissing(mockMetadataGce, project)
}
