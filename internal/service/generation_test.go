package service_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"leonardo-cli/internal/domain"
	"leonardo-cli/internal/service"
)

// fakeLeonardoClient implements ports.LeonardoClient for testing the service
// layer at the port boundary. We stub only the port — never internal
// collaborators — following Cooper's guidance on hexagonal testing.
type fakeLeonardoClient struct {
	createFn   func(req domain.GenerationRequest) (domain.GenerationResponse, error)
	statusFn   func(id string) (domain.GenerationStatus, error)
	deleteFn   func(id string) (domain.DeleteResponse, error)
	userFn     func() (domain.UserInfo, error)
	listFn     func(userID string, offset, limit int) (domain.GenerationListResponse, error)
	downloadFn func(url, destPath string) error
}

func (f *fakeLeonardoClient) CreateGeneration(req domain.GenerationRequest) (domain.GenerationResponse, error) {
	return f.createFn(req)
}

func (f *fakeLeonardoClient) GetGenerationStatus(id string) (domain.GenerationStatus, error) {
	return f.statusFn(id)
}

func (f *fakeLeonardoClient) DeleteGeneration(id string) (domain.DeleteResponse, error) {
	return f.deleteFn(id)
}

func (f *fakeLeonardoClient) GetUserInfo() (domain.UserInfo, error) {
	return f.userFn()
}

func (f *fakeLeonardoClient) ListGenerations(userID string, offset, limit int) (domain.GenerationListResponse, error) {
	return f.listFn(userID, offset, limit)
}

func (f *fakeLeonardoClient) DownloadImage(url, destPath string) error {
	return f.downloadFn(url, destPath)
}

// --- Behavior: Creating a generation ---

func TestCreate_ReturnsGenerationIDAndRawResponse(t *testing.T) {
	fake := &fakeLeonardoClient{
		createFn: func(req domain.GenerationRequest) (domain.GenerationResponse, error) {
			return domain.GenerationResponse{
				GenerationID: "gen-abc-123",
				Raw:          []byte(`{"sdGenerationJob":{"generationId":"gen-abc-123"}}`),
			}, nil
		},
	}
	svc := service.NewGenerationService(fake)

	resp, err := svc.Create(domain.GenerationRequest{Prompt: "a sunset over the ocean"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.GenerationID != "gen-abc-123" {
		t.Errorf("expected generation ID %q, got %q", "gen-abc-123", resp.GenerationID)
	}
	if len(resp.Raw) == 0 {
		t.Error("expected non-empty raw response")
	}
}

func TestCreate_PassesAllRequestFieldsToClient(t *testing.T) {
	var captured domain.GenerationRequest
	fake := &fakeLeonardoClient{
		createFn: func(req domain.GenerationRequest) (domain.GenerationResponse, error) {
			captured = req
			return domain.GenerationResponse{GenerationID: "gen-xyz"}, nil
		},
	}
	svc := service.NewGenerationService(fake)

	req := domain.GenerationRequest{
		Prompt:        "a castle in the clouds",
		ModelID:       "model-42",
		Width:         1920,
		Height:        1080,
		NumImages:     4,
		Alchemy:       true,
		Ultra:         true,
		StyleUUID:     "style-uuid-99",
		Contrast:      3.5,
		GuidanceScale: 7.0,
	}
	_, err := svc.Create(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Prompt != req.Prompt {
		t.Errorf("Prompt: got %q, want %q", captured.Prompt, req.Prompt)
	}
	if captured.ModelID != req.ModelID {
		t.Errorf("ModelID: got %q, want %q", captured.ModelID, req.ModelID)
	}
	if captured.Width != req.Width {
		t.Errorf("Width: got %d, want %d", captured.Width, req.Width)
	}
	if captured.Height != req.Height {
		t.Errorf("Height: got %d, want %d", captured.Height, req.Height)
	}
	if captured.NumImages != req.NumImages {
		t.Errorf("NumImages: got %d, want %d", captured.NumImages, req.NumImages)
	}
	if captured.Alchemy != req.Alchemy {
		t.Errorf("Alchemy: got %v, want %v", captured.Alchemy, req.Alchemy)
	}
	if captured.Ultra != req.Ultra {
		t.Errorf("Ultra: got %v, want %v", captured.Ultra, req.Ultra)
	}
	if captured.StyleUUID != req.StyleUUID {
		t.Errorf("StyleUUID: got %q, want %q", captured.StyleUUID, req.StyleUUID)
	}
	if captured.Contrast != req.Contrast {
		t.Errorf("Contrast: got %f, want %f", captured.Contrast, req.Contrast)
	}
	if captured.GuidanceScale != req.GuidanceScale {
		t.Errorf("GuidanceScale: got %f, want %f", captured.GuidanceScale, req.GuidanceScale)
	}
}

func TestCreate_PropagatesClientError(t *testing.T) {
	fake := &fakeLeonardoClient{
		createFn: func(req domain.GenerationRequest) (domain.GenerationResponse, error) {
			return domain.GenerationResponse{}, errors.New("API returned status 401")
		},
	}
	svc := service.NewGenerationService(fake)

	_, err := svc.Create(domain.GenerationRequest{Prompt: "anything"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "API returned status 401" {
		t.Errorf("expected error message %q, got %q", "API returned status 401", err.Error())
	}
}

// --- Behavior: Checking generation status ---

func TestStatus_ReturnsCompletedStatusWithImageURLs(t *testing.T) {
	fake := &fakeLeonardoClient{
		statusFn: func(id string) (domain.GenerationStatus, error) {
			return domain.GenerationStatus{
				Status: "COMPLETE",
				Images: []string{
					"https://cdn.leonardo.ai/image1.png",
					"https://cdn.leonardo.ai/image2.png",
				},
				Raw: []byte(`{"generations_by_pk":{"status":"COMPLETE"}}`),
			}, nil
		},
	}
	svc := service.NewGenerationService(fake)

	status, err := svc.Status("gen-abc-123")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status.Status != "COMPLETE" {
		t.Errorf("expected status %q, got %q", "COMPLETE", status.Status)
	}
	if len(status.Images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(status.Images))
	}
	if status.Images[0] != "https://cdn.leonardo.ai/image1.png" {
		t.Errorf("expected first image URL %q, got %q", "https://cdn.leonardo.ai/image1.png", status.Images[0])
	}
}

func TestStatus_PendingGenerationReturnsNoImages(t *testing.T) {
	fake := &fakeLeonardoClient{
		statusFn: func(id string) (domain.GenerationStatus, error) {
			return domain.GenerationStatus{
				Status: "PENDING",
				Images: nil,
				Raw:    []byte(`{"generations_by_pk":{"status":"PENDING","generated_images":[]}}`),
			}, nil
		},
	}
	svc := service.NewGenerationService(fake)

	status, err := svc.Status("gen-pending-456")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status.Status != "PENDING" {
		t.Errorf("expected status %q, got %q", "PENDING", status.Status)
	}
	if len(status.Images) != 0 {
		t.Errorf("expected 0 images for pending generation, got %d", len(status.Images))
	}
}

func TestStatus_PassesGenerationIDToClient(t *testing.T) {
	var capturedID string
	fake := &fakeLeonardoClient{
		statusFn: func(id string) (domain.GenerationStatus, error) {
			capturedID = id
			return domain.GenerationStatus{Status: "COMPLETE"}, nil
		},
	}
	svc := service.NewGenerationService(fake)

	_, _ = svc.Status("my-specific-gen-id")

	if capturedID != "my-specific-gen-id" {
		t.Errorf("expected ID %q passed to client, got %q", "my-specific-gen-id", capturedID)
	}
}

func TestStatus_PropagatesClientError(t *testing.T) {
	fake := &fakeLeonardoClient{
		statusFn: func(id string) (domain.GenerationStatus, error) {
			return domain.GenerationStatus{}, errors.New("API returned status 404")
		},
	}
	svc := service.NewGenerationService(fake)

	_, err := svc.Status("nonexistent-id")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "API returned status 404" {
		t.Errorf("expected error message %q, got %q", "API returned status 404", err.Error())
	}
}

// --- Behavior: Deleting a generation ---

func TestDelete_ReturnsDeletedIDAndRawResponse(t *testing.T) {
	fake := &fakeLeonardoClient{
		deleteFn: func(id string) (domain.DeleteResponse, error) {
			return domain.DeleteResponse{
				ID:  "gen-del-456",
				Raw: []byte(`{"delete_generations_by_pk":{"id":"gen-del-456"}}`),
			}, nil
		},
	}
	svc := service.NewGenerationService(fake)

	resp, err := svc.Delete("gen-del-456")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ID != "gen-del-456" {
		t.Errorf("expected deleted ID %q, got %q", "gen-del-456", resp.ID)
	}
	if len(resp.Raw) == 0 {
		t.Error("expected non-empty raw response")
	}
}

func TestDelete_PassesGenerationIDToClient(t *testing.T) {
	var capturedID string
	fake := &fakeLeonardoClient{
		deleteFn: func(id string) (domain.DeleteResponse, error) {
			capturedID = id
			return domain.DeleteResponse{ID: id}, nil
		},
	}
	svc := service.NewGenerationService(fake)

	_, _ = svc.Delete("my-gen-to-delete")

	if capturedID != "my-gen-to-delete" {
		t.Errorf("expected ID %q passed to client, got %q", "my-gen-to-delete", capturedID)
	}
}

func TestDelete_PropagatesClientError(t *testing.T) {
	fake := &fakeLeonardoClient{
		deleteFn: func(id string) (domain.DeleteResponse, error) {
			return domain.DeleteResponse{}, errors.New("API returned status 404")
		},
	}
	svc := service.NewGenerationService(fake)

	_, err := svc.Delete("nonexistent-id")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "API returned status 404" {
		t.Errorf("expected error message %q, got %q", "API returned status 404", err.Error())
	}
}

// --- Behavior: Getting user info ---

func TestUserInfo_ReturnsUserDetailsAndTokenBalances(t *testing.T) {
	fake := &fakeLeonardoClient{
		userFn: func() (domain.UserInfo, error) {
			return domain.UserInfo{
				UserID:                "user-uuid-1",
				Username:              "testuser",
				APISubscriptionTokens: 10000,
				APIPaidTokens:         5000,
				TokenRenewalDate:      "2026-03-01T00:00:00.000Z",
				Raw:                   []byte(`{"user_details":[{}]}`),
			}, nil
		},
	}
	svc := service.NewGenerationService(fake)

	info, err := svc.UserInfo()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if info.UserID != "user-uuid-1" {
		t.Errorf("expected user ID %q, got %q", "user-uuid-1", info.UserID)
	}
	if info.Username != "testuser" {
		t.Errorf("expected username %q, got %q", "testuser", info.Username)
	}
	if info.APISubscriptionTokens != 10000 {
		t.Errorf("expected apiSubscriptionTokens 10000, got %d", info.APISubscriptionTokens)
	}
	if info.APIPaidTokens != 5000 {
		t.Errorf("expected apiPaidTokens 5000, got %d", info.APIPaidTokens)
	}
	if len(info.Raw) == 0 {
		t.Error("expected non-empty raw response")
	}
}

func TestUserInfo_PropagatesClientError(t *testing.T) {
	fake := &fakeLeonardoClient{
		userFn: func() (domain.UserInfo, error) {
			return domain.UserInfo{}, errors.New("API returned status 401")
		},
	}
	svc := service.NewGenerationService(fake)

	_, err := svc.UserInfo()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "API returned status 401" {
		t.Errorf("expected error message %q, got %q", "API returned status 401", err.Error())
	}
}

// --- Behavior: Listing generations ---

func TestListGenerations_ReturnsGenerationsFromClient(t *testing.T) {
	fake := &fakeLeonardoClient{
		listFn: func(userID string, offset, limit int) (domain.GenerationListResponse, error) {
			return domain.GenerationListResponse{
				Generations: []domain.GenerationListItem{
					{ID: "gen-1", Status: "COMPLETE", Prompt: "sunset"},
					{ID: "gen-2", Status: "PENDING", Prompt: "mountain"},
				},
				Raw: []byte(`{"generations":[{},{}]}`),
			}, nil
		},
	}
	svc := service.NewGenerationService(fake)

	resp, err := svc.ListGenerations("user-1", 0, 10)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.Generations) != 2 {
		t.Fatalf("expected 2 generations, got %d", len(resp.Generations))
	}
	if resp.Generations[0].ID != "gen-1" {
		t.Errorf("expected first generation ID %q, got %q", "gen-1", resp.Generations[0].ID)
	}
	if resp.Generations[1].Status != "PENDING" {
		t.Errorf("expected second generation status %q, got %q", "PENDING", resp.Generations[1].Status)
	}
}

func TestListGenerations_PassesParametersToClient(t *testing.T) {
	var capturedUserID string
	var capturedOffset, capturedLimit int
	fake := &fakeLeonardoClient{
		listFn: func(userID string, offset, limit int) (domain.GenerationListResponse, error) {
			capturedUserID = userID
			capturedOffset = offset
			capturedLimit = limit
			return domain.GenerationListResponse{}, nil
		},
	}
	svc := service.NewGenerationService(fake)

	_, _ = svc.ListGenerations("user-xyz", 5, 25)

	if capturedUserID != "user-xyz" {
		t.Errorf("expected userID %q, got %q", "user-xyz", capturedUserID)
	}
	if capturedOffset != 5 {
		t.Errorf("expected offset 5, got %d", capturedOffset)
	}
	if capturedLimit != 25 {
		t.Errorf("expected limit 25, got %d", capturedLimit)
	}
}

func TestListGenerations_PropagatesClientError(t *testing.T) {
	fake := &fakeLeonardoClient{
		listFn: func(userID string, offset, limit int) (domain.GenerationListResponse, error) {
			return domain.GenerationListResponse{}, errors.New("API returned status 403")
		},
	}
	svc := service.NewGenerationService(fake)

	_, err := svc.ListGenerations("user-1", 0, 10)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "API returned status 403" {
		t.Errorf("expected error message %q, got %q", "API returned status 403", err.Error())
	}
}

// --- Behavior: Downloading images for a generation ---

func TestDownload_DownloadsAllImagesAndReturnsFilePaths(t *testing.T) {
	fake := &fakeLeonardoClient{
		statusFn: func(id string) (domain.GenerationStatus, error) {
			return domain.GenerationStatus{
				Status: "COMPLETE",
				Images: []string{
					"https://cdn.leonardo.ai/img1.png",
					"https://cdn.leonardo.ai/img2.png",
				},
				Raw: []byte(`{}`),
			}, nil
		},
		downloadFn: func(url, destPath string) error {
			// Simulate successful download by creating the file
			return os.WriteFile(destPath, []byte("fake-image"), 0644)
		},
	}
	svc := service.NewGenerationService(fake)

	outputDir := t.TempDir()
	result, err := svc.Download("gen-abc-123", outputDir)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.FilePaths) != 2 {
		t.Fatalf("expected 2 file paths, got %d", len(result.FilePaths))
	}
	for _, fp := range result.FilePaths {
		if _, statErr := os.Stat(fp); os.IsNotExist(statErr) {
			t.Errorf("expected file %q to exist", fp)
		}
	}
}

func TestDownload_UsesGenerationIDAndIndexInFilenames(t *testing.T) {
	fake := &fakeLeonardoClient{
		statusFn: func(id string) (domain.GenerationStatus, error) {
			return domain.GenerationStatus{
				Status: "COMPLETE",
				Images: []string{"https://cdn.leonardo.ai/img1.png"},
				Raw:    []byte(`{}`),
			}, nil
		},
		downloadFn: func(url, destPath string) error {
			return os.WriteFile(destPath, []byte("data"), 0600)
		},
	}
	svc := service.NewGenerationService(fake)

	outputDir := t.TempDir()
	result, err := svc.Download("gen-xyz", outputDir)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.FilePaths) != 1 {
		t.Fatalf("expected 1 file path, got %d", len(result.FilePaths))
	}
	expected := filepath.Join(outputDir, "gen-xyz_1.png")
	if result.FilePaths[0] != expected {
		t.Errorf("expected file path %q, got %q", expected, result.FilePaths[0])
	}
}

func TestDownload_ReturnsErrorWhenGenerationNotComplete(t *testing.T) {
	fake := &fakeLeonardoClient{
		statusFn: func(id string) (domain.GenerationStatus, error) {
			return domain.GenerationStatus{
				Status: "PENDING",
				Images: nil,
				Raw:    []byte(`{}`),
			}, nil
		},
	}
	svc := service.NewGenerationService(fake)

	_, err := svc.Download("gen-pending", t.TempDir())

	if err == nil {
		t.Fatal("expected error for non-complete generation, got nil")
	}
	if !strings.Contains(err.Error(), "PENDING") {
		t.Errorf("expected error to mention status PENDING, got %q", err.Error())
	}
}

func TestDownload_ReturnsErrorWhenNoImages(t *testing.T) {
	fake := &fakeLeonardoClient{
		statusFn: func(id string) (domain.GenerationStatus, error) {
			return domain.GenerationStatus{
				Status: "COMPLETE",
				Images: []string{},
				Raw:    []byte(`{}`),
			}, nil
		},
	}
	svc := service.NewGenerationService(fake)

	_, err := svc.Download("gen-no-images", t.TempDir())

	if err == nil {
		t.Fatal("expected error when no images available, got nil")
	}
	if !strings.Contains(err.Error(), "no images") {
		t.Errorf("expected error to mention 'no images', got %q", err.Error())
	}
}

func TestDownload_PropagatesStatusError(t *testing.T) {
	fake := &fakeLeonardoClient{
		statusFn: func(id string) (domain.GenerationStatus, error) {
			return domain.GenerationStatus{}, errors.New("API returned status 404")
		},
	}
	svc := service.NewGenerationService(fake)

	_, err := svc.Download("nonexistent", t.TempDir())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "API returned status 404" {
		t.Errorf("expected error message %q, got %q", "API returned status 404", err.Error())
	}
}

func TestDownload_PropagatesDownloadError(t *testing.T) {
	fake := &fakeLeonardoClient{
		statusFn: func(id string) (domain.GenerationStatus, error) {
			return domain.GenerationStatus{
				Status: "COMPLETE",
				Images: []string{"https://cdn.leonardo.ai/img1.png"},
				Raw:    []byte(`{}`),
			}, nil
		},
		downloadFn: func(url, destPath string) error {
			return errors.New("download failed: connection refused")
		},
	}
	svc := service.NewGenerationService(fake)

	_, err := svc.Download("gen-fail", t.TempDir())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "download failed") {
		t.Errorf("expected error to mention 'download failed', got %q", err.Error())
	}
}

func TestDownload_PassesCorrectURLsToClient(t *testing.T) {
	var capturedURLs []string
	fake := &fakeLeonardoClient{
		statusFn: func(id string) (domain.GenerationStatus, error) {
			return domain.GenerationStatus{
				Status: "COMPLETE",
				Images: []string{
					"https://cdn.leonardo.ai/first.png",
					"https://cdn.leonardo.ai/second.png",
				},
				Raw: []byte(`{}`),
			}, nil
		},
		downloadFn: func(url, destPath string) error {
			capturedURLs = append(capturedURLs, url)
			return os.WriteFile(destPath, []byte("data"), 0644)
		},
	}
	svc := service.NewGenerationService(fake)

	_, err := svc.Download("gen-urls", t.TempDir())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(capturedURLs) != 2 {
		t.Fatalf("expected 2 download calls, got %d", len(capturedURLs))
	}
	if capturedURLs[0] != "https://cdn.leonardo.ai/first.png" {
		t.Errorf("expected first URL %q, got %q", "https://cdn.leonardo.ai/first.png", capturedURLs[0])
	}
	if capturedURLs[1] != "https://cdn.leonardo.ai/second.png" {
		t.Errorf("expected second URL %q, got %q", "https://cdn.leonardo.ai/second.png", capturedURLs[1])
	}
}

func TestDownload_WritesJSONSidecarForEachImage(t *testing.T) {
	fake := &fakeLeonardoClient{
		statusFn: func(id string) (domain.GenerationStatus, error) {
			return domain.GenerationStatus{
				Status: "COMPLETE",
				Images: []string{"https://cdn.leonardo.ai/img1.png"},
				Raw:    []byte(`{"generations_by_pk":{"prompt":"sidecar prompt","modelId":"model-1","num_images":1}}`),
			}, nil
		},
		downloadFn: func(url, destPath string) error {
			return os.WriteFile(destPath, []byte("data"), 0600)
		},
	}
	svc := service.NewGenerationService(fake)

	outputDir := t.TempDir()
	result, err := svc.Download("gen-sidecar", outputDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.FilePaths) != 1 {
		t.Fatalf("expected 1 file path, got %d", len(result.FilePaths))
	}

	sidecarPath := result.FilePaths[0] + ".json"
	sidecarBytes, err := os.ReadFile(sidecarPath)
	if err != nil {
		t.Fatalf("expected sidecar file to exist: %v", err)
	}

	var sidecar map[string]interface{}
	if err := json.Unmarshal(sidecarBytes, &sidecar); err != nil {
		t.Fatalf("expected valid sidecar JSON, got error: %v", err)
	}
	if sidecar["generation_id"] != "gen-sidecar" {
		t.Errorf("expected generation_id %q, got %v", "gen-sidecar", sidecar["generation_id"])
	}
	if sidecar["image_url"] != "https://cdn.leonardo.ai/img1.png" {
		t.Errorf("expected image_url %q, got %v", "https://cdn.leonardo.ai/img1.png", sidecar["image_url"])
	}
	if _, ok := sidecar["timestamp"]; !ok {
		t.Error("expected sidecar timestamp to be present")
	}
	parameters, ok := sidecar["parameters"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected parameters map in sidecar, got %T", sidecar["parameters"])
	}
	if parameters["prompt"] != "sidecar prompt" {
		t.Errorf("expected prompt in sidecar parameters, got %v", parameters["prompt"])
	}
}
