package zvuk

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	mock_zvuk_client "github.com/oshokin/zvuk-grabber/internal/client/zvuk/mocks"
	"github.com/oshokin/zvuk-grabber/internal/config"
)

// mockURLProcessor is a mock implementation of the URLProcessor interface.
type mockURLProcessor struct{}

func (m *mockURLProcessor) ExtractDownloadItems(
	_ context.Context,
	_ []string,
) (*ExtractDownloadItemsResponse, error) {
	return &ExtractDownloadItemsResponse{}, nil
}

func (m *mockURLProcessor) DeduplicateDownloadItems(items []*DownloadItem) []*DownloadItem {
	return items
}

// mockTemplateManager is a mock implementation of the TemplateManager interface.
type mockTemplateManager struct{}

func (m *mockTemplateManager) GetTrackFilename(
	_ context.Context,
	_ bool,
	_ map[string]string,
	_ int64,
) string {
	return "test_track.mp3"
}

func (m *mockTemplateManager) GetAlbumFolderName(_ context.Context, _ map[string]string) string {
	return "test_album"
}

// mockTagProcessor is a mock implementation of the TagProcessor interface.
type mockTagProcessor struct{}

func (m *mockTagProcessor) WriteTags(_ context.Context, _ *WriteTagsRequest) error {
	return nil
}

// TestNewService tests the NewService function.
func TestNewService(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &config.Config{
		OutputPath: "/tmp/test",
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := &mockURLProcessor{}
	mockTemplateManager := &mockTemplateManager{}
	mockTagProcessor := &mockTagProcessor{}

	service := NewService(
		config,
		mockClient,
		mockURLProcessor,
		mockTemplateManager,
		mockTagProcessor,
	)

	assert.NotNil(t, service)
}

// TestServiceImpl_DownloadURLs tests the DownloadURLs method.
func TestServiceImpl_DownloadURLs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &config.Config{
		OutputPath: "/tmp/test",
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := &mockURLProcessor{}
	mockTemplateManager := &mockTemplateManager{}
	mockTagProcessor := &mockTagProcessor{}

	// Setup mock expectations
	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(&zvuk.UserProfile{
		Subscription: &zvuk.UserSubscription{
			Title:      "Premium",
			Expiration: 1234567890,
		},
	}, nil).AnyTimes()

	service := NewService(
		config,
		mockClient,
		mockURLProcessor,
		mockTemplateManager,
		mockTagProcessor,
	)

	ctx := context.Background()
	urls := []string{"https://zvuk.com/track/123"}

	// This should not panic
	service.DownloadURLs(ctx, urls)
}

// TestServiceImpl_DownloadURLs_EmptyURLs tests DownloadURLs with empty URLs.
func TestServiceImpl_DownloadURLs_EmptyURLs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &config.Config{
		OutputPath: "/tmp/test",
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := &mockURLProcessor{}
	mockTemplateManager := &mockTemplateManager{}
	mockTagProcessor := &mockTagProcessor{}

	// Setup mock expectations for empty URLs.
	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(&zvuk.UserProfile{
		Subscription: &zvuk.UserSubscription{
			Title:      "Premium",
			Expiration: 1234567890,
		},
	}, nil).AnyTimes()

	service := NewService(
		config,
		mockClient,
		mockURLProcessor,
		mockTemplateManager,
		mockTagProcessor,
	)

	ctx := context.Background()
	urls := []string{}

	// This should not panic
	service.DownloadURLs(ctx, urls)
}

// TestServiceImpl_DownloadURLs_NilURLs tests DownloadURLs with nil URLs.
func TestServiceImpl_DownloadURLs_NilURLs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &config.Config{
		OutputPath: "/tmp/test",
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := &mockURLProcessor{}
	mockTemplateManager := &mockTemplateManager{}
	mockTagProcessor := &mockTagProcessor{}

	// Setup mock expectations for nil URLs.
	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(&zvuk.UserProfile{
		Subscription: &zvuk.UserSubscription{
			Title:      "Premium",
			Expiration: 1234567890,
		},
	}, nil).AnyTimes()

	service := NewService(
		config,
		mockClient,
		mockURLProcessor,
		mockTemplateManager,
		mockTagProcessor,
	)

	ctx := context.Background()

	var urls []string

	// This should not panic
	service.DownloadURLs(ctx, urls)
}
