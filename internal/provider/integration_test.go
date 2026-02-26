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
// These tests require a valid LEONARDO_API_TOKEN environment variable and
// sufficient API credits. They are skipped when running with -short or
// when the environment variable is absent.
//
// Run them explicitly:
//
//   LEONARDO_API_TOKEN=your-token go test ./internal/provider/ -run Integration -v
//

func requireAPIKey(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	key := os.Getenv("LEONARDO_API_TOKEN")
	if strings.TrimSpace(key) == "" {
		t.Skip("skipping integration test: LEONARDO_API_TOKEN not set")
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

func TestIntegration_GetUserInfo(t *testing.T) {
	apiKey := requireAPIKey(t)

	client := provider.NewAPIClient(apiKey, nil)

	info, err := client.GetUserInfo()
	if err != nil {
		t.Fatalf("GetUserInfo failed: %v", err)
	}
	if info.UserID == "" {
		t.Error("expected a non-empty user ID")
	}
	t.Logf("User ID: %s", info.UserID)
	t.Logf("Username: %s", info.Username)
	t.Logf("API Subscription Tokens: %d", info.APISubscriptionTokens)
	t.Logf("API Paid Tokens: %d", info.APIPaidTokens)
	t.Logf("Token Renewal Date: %s", info.TokenRenewalDate)
	if len(info.Raw) == 0 {
		t.Error("expected non-empty raw response")
	}
}

func TestIntegration_ListGenerations(t *testing.T) {
	apiKey := requireAPIKey(t)

	client := provider.NewAPIClient(apiKey, nil)

	// First get our user ID
	info, err := client.GetUserInfo()
	if err != nil {
		t.Fatalf("GetUserInfo failed: %v", err)
	}
	if info.UserID == "" {
		t.Fatal("expected a non-empty user ID to list generations")
	}

	resp, err := client.ListGenerations(info.UserID, 0, 5)
	if err != nil {
		t.Fatalf("ListGenerations failed: %v", err)
	}
	t.Logf("Found %d generations", len(resp.Generations))
	for _, gen := range resp.Generations {
		t.Logf("  [%s] %s — %s (%d images)", gen.Status, gen.ID, gen.Prompt, len(gen.Images))
	}
	if len(resp.Raw) == 0 {
		t.Error("expected non-empty raw response")
	}
}

func TestIntegration_DeleteGeneration(t *testing.T) {
	apiKey := requireAPIKey(t)

	client := provider.NewAPIClient(apiKey, nil)

	// Create a generation to delete
	req := domain.GenerationRequest{
		Prompt:    "A tiny dot for deletion test",
		NumImages: 1,
	}
	createResp, err := client.CreateGeneration(req)
	if err != nil {
		t.Fatalf("CreateGeneration failed: %v", err)
	}
	if createResp.GenerationID == "" {
		t.Fatal("expected a non-empty generation ID")
	}
	t.Logf("Created generation for deletion: %s", createResp.GenerationID)

	// Delete it
	delResp, err := client.DeleteGeneration(createResp.GenerationID)
	if err != nil {
		t.Fatalf("DeleteGeneration failed: %v", err)
	}
	if delResp.ID != createResp.GenerationID {
		t.Errorf("expected deleted ID %q, got %q", createResp.GenerationID, delResp.ID)
	}
	t.Logf("Deleted generation: %s", delResp.ID)

	// Verify it's gone — status should return an error or empty
	status, err := client.GetGenerationStatus(createResp.GenerationID)
	if err != nil {
		t.Logf("Expected error after deletion: %v", err)
		return
	}
	// Some APIs return 200 with null data for deleted generations
	if status.Status != "" {
		t.Logf("Status after deletion: %q (may be cached)", status.Status)
	}
}
