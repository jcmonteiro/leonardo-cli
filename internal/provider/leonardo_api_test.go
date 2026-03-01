package provider_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"leonardo-cli/internal/domain"
	"leonardo-cli/internal/provider"
)

// These tests verify the APIClient adapter's behavior by running it against
// a real HTTP server (httptest.Server). We test the adapter at its boundary
// — the HTTP contract — rather than mocking internal collaborators.

// --- Behavior: Creating a generation via HTTP ---

func TestAPIClient_CreateGeneration_SendsCorrectHTTPRequest(t *testing.T) {
	var receivedBody map[string]interface{}
	var receivedHeaders http.Header
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedHeaders = r.Header
		body, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"sdGenerationJob":{"generationId":"gen-from-server"}}`))
	}))
	defer server.Close()

	client := provider.NewAPIClient("test-api-key-123", server.Client())
	// We need to override the base URL — but the provider hardcodes it.
	// Instead, we test through a server that accepts any path and verify
	// the payload shape and headers. For full URL testing, see integration tests.
	// Here we redirect through a transport.
	client = newClientWithBaseURL("test-api-key-123", server.URL)

	req := domain.GenerationRequest{
		NumImages: 3,
		Private:   true,
		Metadata: domain.GenerationMetadata{
			Prompt:         "a beautiful landscape",
			NegativePrompt: "low quality",
			ModelID:        "model-abc",
			Width:          1024,
			Height:         768,
			Seed:           42,
			Alchemy:        true,
		},
	}

	resp, err := client.CreateGeneration(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != "POST" {
		t.Errorf("expected POST, got %s", receivedMethod)
	}
	if receivedPath != "/api/rest/v1/generations" {
		t.Errorf("expected path /api/rest/v1/generations, got %s", receivedPath)
	}
	if receivedHeaders.Get("Authorization") != "Bearer test-api-key-123" {
		t.Errorf("expected Authorization header %q, got %q", "Bearer test-api-key-123", receivedHeaders.Get("Authorization"))
	}
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", receivedHeaders.Get("Content-Type"))
	}
	// Verify payload fields
	if receivedBody["prompt"] != "a beautiful landscape" {
		t.Errorf("expected prompt %q, got %v", "a beautiful landscape", receivedBody["prompt"])
	}
	if receivedBody["num_images"] != 3.0 { // JSON numbers are float64
		t.Errorf("expected num_images 3, got %v", receivedBody["num_images"])
	}
	if receivedBody["modelId"] != "model-abc" {
		t.Errorf("expected modelId %q, got %v", "model-abc", receivedBody["modelId"])
	}
	if receivedBody["negative_prompt"] != "low quality" {
		t.Errorf("expected negative_prompt %q, got %v", "low quality", receivedBody["negative_prompt"])
	}
	if receivedBody["width"] != 1024.0 {
		t.Errorf("expected width 1024, got %v", receivedBody["width"])
	}
	if receivedBody["height"] != 768.0 {
		t.Errorf("expected height 768, got %v", receivedBody["height"])
	}
	if receivedBody["alchemy"] != true {
		t.Errorf("expected alchemy true, got %v", receivedBody["alchemy"])
	}
	if receivedBody["public"] != false {
		t.Errorf("expected public false, got %v", receivedBody["public"])
	}
	if receivedBody["seed"] != 42.0 {
		t.Errorf("expected seed 42, got %v", receivedBody["seed"])
	}
	if resp.GenerationID != "gen-from-server" {
		t.Errorf("expected generation ID %q, got %q", "gen-from-server", resp.GenerationID)
	}
}

func TestAPIClient_CreateGeneration_OmitsZeroValueOptionalFields(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"sdGenerationJob":{"generationId":"gen-minimal"}}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	// Only prompt and num_images provided — all other fields are zero-value
	req := domain.GenerationRequest{
		NumImages: 1,
		Metadata: domain.GenerationMetadata{
			Prompt: "minimal request",
		},
	}
	_, err := client.CreateGeneration(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// These optional fields should NOT be present in the payload
	for _, key := range []string{"modelId", "negative_prompt", "width", "height", "public", "alchemy", "ultra", "styleUUID", "contrast", "guidance_scale", "seed"} {
		if _, exists := receivedBody[key]; exists {
			t.Errorf("expected optional field %q to be omitted, but it was present with value %v", key, receivedBody[key])
		}
	}
	// Required fields should be present
	if receivedBody["prompt"] != "minimal request" {
		t.Errorf("expected prompt %q, got %v", "minimal request", receivedBody["prompt"])
	}
}

func TestAPIClient_CreateGeneration_ReturnsErrorOnNon2xxStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("bad-key", server.URL)

	_, err := client.CreateGeneration(domain.GenerationRequest{
		NumImages: 1,
		Metadata: domain.GenerationMetadata{
			Prompt: "test",
		},
	})
	if err == nil {
		t.Fatal("expected error for 401 status, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to mention status 401, got %q", err.Error())
	}
}

func TestAPIClient_CreateGeneration_IncludesAllOptionalFields(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"sdGenerationJob":{"generationId":"gen-full"}}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	req := domain.GenerationRequest{
		NumImages: 2,
		Metadata: domain.GenerationMetadata{
			Prompt:         "fully loaded request",
			NegativePrompt: "bad anatomy",
			ModelID:        "model-full",
			Width:          512,
			Height:         512,
			Seed:           777,
			Alchemy:        true,
			Ultra:          true,
			StyleUUID:      "style-123",
			Contrast:       2.5,
			GuidanceScale:  8.0,
		},
	}
	_, err := client.CreateGeneration(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["ultra"] != true {
		t.Errorf("expected ultra true, got %v", receivedBody["ultra"])
	}
	if receivedBody["styleUUID"] != "style-123" {
		t.Errorf("expected styleUUID %q, got %v", "style-123", receivedBody["styleUUID"])
	}
	if receivedBody["negative_prompt"] != "bad anatomy" {
		t.Errorf("expected negative_prompt %q, got %v", "bad anatomy", receivedBody["negative_prompt"])
	}
	if receivedBody["contrast"] != 2.5 {
		t.Errorf("expected contrast 2.5, got %v", receivedBody["contrast"])
	}
	if receivedBody["guidance_scale"] != 8.0 {
		t.Errorf("expected guidance_scale 8.0, got %v", receivedBody["guidance_scale"])
	}
	if receivedBody["seed"] != 777.0 {
		t.Errorf("expected seed 777, got %v", receivedBody["seed"])
	}
}

// --- Behavior: Checking generation status via HTTP ---

func TestAPIClient_GetGenerationStatus_SendsCorrectHTTPRequest(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"generations_by_pk":{"status":"COMPLETE","generated_images":[{"url":"https://cdn.leonardo.ai/img1.png"}]}}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("my-api-key", server.URL)

	status, err := client.GetGenerationStatus("gen-id-789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/api/rest/v1/generations/gen-id-789" {
		t.Errorf("expected path /api/rest/v1/generations/gen-id-789, got %s", receivedPath)
	}
	if receivedHeaders.Get("Authorization") != "Bearer my-api-key" {
		t.Errorf("expected Authorization header %q, got %q", "Bearer my-api-key", receivedHeaders.Get("Authorization"))
	}
	if status.Status != "COMPLETE" {
		t.Errorf("expected status %q, got %q", "COMPLETE", status.Status)
	}
	if len(status.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(status.Images))
	}
	if status.Images[0] != "https://cdn.leonardo.ai/img1.png" {
		t.Errorf("expected image URL %q, got %q", "https://cdn.leonardo.ai/img1.png", status.Images[0])
	}
}

func TestAPIClient_GetGenerationStatus_ParsesMultipleImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"generations_by_pk":{
				"status":"COMPLETE",
				"generated_images":[
					{"url":"https://cdn.leonardo.ai/img1.png"},
					{"url":"https://cdn.leonardo.ai/img2.png"},
					{"url":"https://cdn.leonardo.ai/img3.png"}
				]
			}
		}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	status, err := client.GetGenerationStatus("gen-multi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(status.Images) != 3 {
		t.Fatalf("expected 3 images, got %d", len(status.Images))
	}
	expected := []string{
		"https://cdn.leonardo.ai/img1.png",
		"https://cdn.leonardo.ai/img2.png",
		"https://cdn.leonardo.ai/img3.png",
	}
	for i, want := range expected {
		if status.Images[i] != want {
			t.Errorf("image %d: expected %q, got %q", i, want, status.Images[i])
		}
	}
}

func TestAPIClient_GetGenerationStatus_PendingHasNoImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"generations_by_pk":{"status":"PENDING","generated_images":[]}}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	status, err := client.GetGenerationStatus("gen-pending")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "PENDING" {
		t.Errorf("expected status %q, got %q", "PENDING", status.Status)
	}
	if len(status.Images) != 0 {
		t.Errorf("expected 0 images for pending, got %d", len(status.Images))
	}
}

func TestAPIClient_GetGenerationStatus_ReturnsErrorOnNon2xxStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"generation not found"}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	_, err := client.GetGenerationStatus("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for 404 status, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention status 404, got %q", err.Error())
	}
}

func TestAPIClient_GetGenerationStatus_ReturnsRawResponseAlways(t *testing.T) {
	expectedJSON := `{"generations_by_pk":{"status":"COMPLETE","custom_field":"extra"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedJSON))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	status, err := client.GetGenerationStatus("gen-raw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(status.Raw) != expectedJSON {
		t.Errorf("expected raw response %q, got %q", expectedJSON, string(status.Raw))
	}
}

// --- Behavior: Deleting a generation via HTTP ---

func TestAPIClient_DeleteGeneration_SendsCorrectHTTPRequest(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"delete_generations_by_pk":{"id":"gen-del-123"}}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("my-api-key", server.URL)

	resp, err := client.DeleteGeneration("gen-del-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != "DELETE" {
		t.Errorf("expected DELETE, got %s", receivedMethod)
	}
	if receivedPath != "/api/rest/v1/generations/gen-del-123" {
		t.Errorf("expected path /api/rest/v1/generations/gen-del-123, got %s", receivedPath)
	}
	if receivedHeaders.Get("Authorization") != "Bearer my-api-key" {
		t.Errorf("expected Authorization header %q, got %q", "Bearer my-api-key", receivedHeaders.Get("Authorization"))
	}
	if resp.ID != "gen-del-123" {
		t.Errorf("expected deleted ID %q, got %q", "gen-del-123", resp.ID)
	}
}

func TestAPIClient_DeleteGeneration_ReturnsErrorOnNon2xxStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"generation not found"}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	_, err := client.DeleteGeneration("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for 404 status, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention status 404, got %q", err.Error())
	}
}

func TestAPIClient_DeleteGeneration_ReturnsRawResponseAlways(t *testing.T) {
	expectedJSON := `{"delete_generations_by_pk":{"id":"gen-raw-del"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedJSON))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	resp, err := client.DeleteGeneration("gen-raw-del")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.Raw) != expectedJSON {
		t.Errorf("expected raw response %q, got %q", expectedJSON, string(resp.Raw))
	}
}

// --- Behavior: Getting user info via HTTP ---

func TestAPIClient_GetUserInfo_SendsCorrectHTTPRequest(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"user_details":[{"user":{"id":"user-uuid-1","username":"testuser"},"apiSubscriptionTokens":10000,"apiPaidTokens":5000,"apiPlanTokenRenewalDate":"2026-03-01T00:00:00.000Z"}]}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("my-api-key", server.URL)

	info, err := client.GetUserInfo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/api/rest/v1/me" {
		t.Errorf("expected path /api/rest/v1/me, got %s", receivedPath)
	}
	if receivedHeaders.Get("Authorization") != "Bearer my-api-key" {
		t.Errorf("expected Authorization header %q, got %q", "Bearer my-api-key", receivedHeaders.Get("Authorization"))
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
	if info.TokenRenewalDate != "2026-03-01T00:00:00.000Z" {
		t.Errorf("expected tokenRenewalDate %q, got %q", "2026-03-01T00:00:00.000Z", info.TokenRenewalDate)
	}
}

func TestAPIClient_GetUserInfo_ReturnsErrorOnNon2xxStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("bad-key", server.URL)

	_, err := client.GetUserInfo()
	if err == nil {
		t.Fatal("expected error for 401 status, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to mention status 401, got %q", err.Error())
	}
}

func TestAPIClient_GetUserInfo_ReturnsRawResponseAlways(t *testing.T) {
	expectedJSON := `{"user_details":[{"user":{"id":"u1","username":"u"},"apiSubscriptionTokens":0,"apiPaidTokens":0}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedJSON))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	info, err := client.GetUserInfo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(info.Raw) != expectedJSON {
		t.Errorf("expected raw response %q, got %q", expectedJSON, string(info.Raw))
	}
}

// --- Behavior: Listing user generations via HTTP ---

func TestAPIClient_ListGenerations_SendsCorrectHTTPRequest(t *testing.T) {
	var receivedMethod, receivedPath, receivedQuery string
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedQuery = r.URL.RawQuery
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"generations":[{"id":"gen-1","status":"COMPLETE","createdAt":"2026-02-26T10:00:00.000Z","prompt":"test prompt","generated_images":[{"url":"https://cdn.leonardo.ai/img1.png"}]}]}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("my-api-key", server.URL)

	resp, err := client.ListGenerations("user-uuid-1", 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/api/rest/v1/generations/user/user-uuid-1" {
		t.Errorf("expected path /api/rest/v1/generations/user/user-uuid-1, got %s", receivedPath)
	}
	if !strings.Contains(receivedQuery, "offset=0") {
		t.Errorf("expected query to contain offset=0, got %q", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "limit=10") {
		t.Errorf("expected query to contain limit=10, got %q", receivedQuery)
	}
	if receivedHeaders.Get("Authorization") != "Bearer my-api-key" {
		t.Errorf("expected Authorization header %q, got %q", "Bearer my-api-key", receivedHeaders.Get("Authorization"))
	}
	if len(resp.Generations) != 1 {
		t.Fatalf("expected 1 generation, got %d", len(resp.Generations))
	}
	gen := resp.Generations[0]
	if gen.ID != "gen-1" {
		t.Errorf("expected generation ID %q, got %q", "gen-1", gen.ID)
	}
	if gen.Status != "COMPLETE" {
		t.Errorf("expected status %q, got %q", "COMPLETE", gen.Status)
	}
	if gen.Prompt != "test prompt" {
		t.Errorf("expected prompt %q, got %q", "test prompt", gen.Prompt)
	}
	if len(gen.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(gen.Images))
	}
	if gen.Images[0] != "https://cdn.leonardo.ai/img1.png" {
		t.Errorf("expected image URL %q, got %q", "https://cdn.leonardo.ai/img1.png", gen.Images[0])
	}
}

func TestAPIClient_ListGenerations_ParsesMultipleGenerations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"generations":[
				{"id":"gen-1","status":"COMPLETE","createdAt":"2026-02-26T10:00:00.000Z","prompt":"first","generated_images":[]},
				{"id":"gen-2","status":"PENDING","createdAt":"2026-02-26T11:00:00.000Z","prompt":"second","generated_images":[]}
			]
		}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	resp, err := client.ListGenerations("user-1", 0, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

func TestAPIClient_ListGenerations_ReturnsErrorOnNon2xxStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	_, err := client.ListGenerations("user-1", 0, 10)
	if err == nil {
		t.Fatal("expected error for 403 status, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected error to mention status 403, got %q", err.Error())
	}
}

func TestAPIClient_ListGenerations_ReturnsRawResponseAlways(t *testing.T) {
	expectedJSON := `{"generations":[]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedJSON))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	resp, err := client.ListGenerations("user-1", 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.Raw) != expectedJSON {
		t.Errorf("expected raw response %q, got %q", expectedJSON, string(resp.Raw))
	}
}

// --- Behavior: Downloading an image via HTTP ---

func TestAPIClient_DownloadImage_SavesFileToDestPath(t *testing.T) {
	expectedContent := []byte("fake-png-image-data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(expectedContent)
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	destDir := t.TempDir()
	destPath := filepath.Join(destDir, "image.png")

	err := client.DownloadImage(server.URL+"/some/image.png", destPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file was created with correct content
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != string(expectedContent) {
		t.Errorf("expected file content %q, got %q", string(expectedContent), string(data))
	}
}

func TestAPIClient_DownloadImage_ReturnsErrorOnNon2xxStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	destDir := t.TempDir()
	destPath := filepath.Join(destDir, "should-not-exist.png")

	err := client.DownloadImage(server.URL+"/missing.png", destPath)
	if err == nil {
		t.Fatal("expected error for 404 status, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention status 404, got %q", err.Error())
	}

	// Verify no file was created
	if _, statErr := os.Stat(destPath); !os.IsNotExist(statErr) {
		t.Error("expected file to not exist after failed download")
	}
}

func TestAPIClient_DownloadImage_DoesNotSendAuthHeader(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("image-bytes"))
	}))
	defer server.Close()

	client := newClientWithBaseURL("secret-api-key", server.URL)

	destDir := t.TempDir()
	destPath := filepath.Join(destDir, "img.png")

	err := client.DownloadImage(server.URL+"/img.png", destPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// DownloadImage fetches from a CDN — it should NOT send the API Authorization header
	if auth := receivedHeaders.Get("Authorization"); auth != "" {
		t.Errorf("expected no Authorization header for image download, got %q", auth)
	}
}

func TestAPIClient_DownloadImage_UsesGETMethod(t *testing.T) {
	var receivedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("image-bytes"))
	}))
	defer server.Close()

	client := newClientWithBaseURL("key", server.URL)

	destDir := t.TempDir()
	destPath := filepath.Join(destDir, "img.png")

	err := client.DownloadImage(server.URL+"/img.png", destPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
}

// --- Behavior: Default HTTP client ---

func TestAPIClient_UsesDefaultHTTPClientWhenNilProvided(t *testing.T) {
	// Passing nil should not panic — the client creates its own http.Client.
	client := provider.NewAPIClient("some-key", nil)
	if client == nil {
		t.Fatal("expected non-nil client when nil http.Client provided")
	}
}

// newClientWithBaseURL creates an APIClient that targets a test server instead
// of the real Leonardo API. It does this by using a custom http.Transport that
// rewrites request URLs to point at the test server.
func newClientWithBaseURL(apiKey, baseURL string) *provider.APIClient {
	transport := &rewriteTransport{baseURL: baseURL}
	httpClient := &http.Client{Transport: transport}
	return provider.NewAPIClient(apiKey, httpClient)
}

// rewriteTransport is an http.RoundTripper that rewrites the host of every
// request to point at a local test server, preserving the path and query.
type rewriteTransport struct {
	baseURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace the scheme+host with our test server, keep the path
	req.URL.Scheme = "http"
	// Strip scheme from baseURL to get host:port
	host := strings.TrimPrefix(t.baseURL, "http://")
	host = strings.TrimPrefix(host, "https://")
	req.URL.Host = host
	return http.DefaultTransport.RoundTrip(req)
}
