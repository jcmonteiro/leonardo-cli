package provider_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"leonardo-cli/internal/domain"
	"leonardo-cli/internal/provider"
)

// Integration tests that hit the real Leonardo.Ai API.
//
// These tests require a valid LEONARDO_API_KEY environment variable and
// sufficient API credits. They are skipped when running with -short or
// when the environment variable is absent.
//
// Run them explicitly:
//
//   LEONARDO_API_KEY=your-key go test ./internal/provider/ -run Integration -v
//

func requireAPIKey(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	key := os.Getenv("LEONARDO_API_KEY")
	if strings.TrimSpace(key) == "" {
		t.Skip("skipping integration test: LEONARDO_API_KEY not set")
	}
	return key
}

func TestIntegration_CreateAndPollGeneration(t *testing.T) {
	apiKey := requireAPIKey(t)

	client := provider.NewAPIClient(apiKey, nil)

	// Create a generation with a simple prompt
	req := domain.GenerationRequest{
		Prompt:    "A simple red circle on a white background",
		NumImages: 1,
	}

	resp, err := client.CreateGeneration(req)
	if err != nil {
		t.Fatalf("CreateGeneration failed: %v", err)
	}
	if resp.GenerationID == "" {
		t.Fatal("expected a non-empty generation ID")
	}
	t.Logf("Created generation: %s", resp.GenerationID)

	// Poll for status — wait up to 2 minutes for completion
	deadline := time.Now().Add(2 * time.Minute)
	var status domain.GenerationStatus
	for time.Now().Before(deadline) {
		status, err = client.GetGenerationStatus(resp.GenerationID)
		if err != nil {
			t.Fatalf("GetGenerationStatus failed: %v", err)
		}
		t.Logf("Status: %s (images: %d)", status.Status, len(status.Images))

		if status.Status == "COMPLETE" {
			break
		}
		time.Sleep(5 * time.Second)
	}

	if status.Status != "COMPLETE" {
		t.Fatalf("generation did not complete within timeout, last status: %s", status.Status)
	}
	if len(status.Images) == 0 {
		t.Error("expected at least one image URL after completion")
	}
	for i, url := range status.Images {
		t.Logf("Image %d: %s", i+1, url)
		if !strings.HasPrefix(url, "https://") {
			t.Errorf("image %d URL doesn't start with https://: %s", i+1, url)
		}
	}
}

func TestIntegration_GetGenerationStatus_InvalidID(t *testing.T) {
	apiKey := requireAPIKey(t)

	client := provider.NewAPIClient(apiKey, nil)

	// Querying a nonsense ID should return an error or empty status
	status, err := client.GetGenerationStatus("nonexistent-generation-id-12345")
	if err != nil {
		// API may return 4xx — this is expected behavior
		t.Logf("Expected error for invalid generation ID: %v", err)
		return
	}
	// Some APIs return 200 with null/empty data instead of an error
	if status.Status != "" {
		t.Errorf("expected empty status for invalid ID, got %q", status.Status)
	}
}
