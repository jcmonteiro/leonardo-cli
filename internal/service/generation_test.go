package service_test

import (
	"errors"
	"testing"

	"leonardo-cli/internal/domain"
	"leonardo-cli/internal/service"
)

// fakeLeonardoClient implements ports.LeonardoClient for testing the service
// layer at the port boundary. We stub only the port — never internal
// collaborators — following Cooper's guidance on hexagonal testing.
type fakeLeonardoClient struct {
	createFn func(req domain.GenerationRequest) (domain.GenerationResponse, error)
	statusFn func(id string) (domain.GenerationStatus, error)
}

func (f *fakeLeonardoClient) CreateGeneration(req domain.GenerationRequest) (domain.GenerationResponse, error) {
	return f.createFn(req)
}

func (f *fakeLeonardoClient) GetGenerationStatus(id string) (domain.GenerationStatus, error) {
	return f.statusFn(id)
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
