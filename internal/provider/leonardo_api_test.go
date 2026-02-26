package provider_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
		Prompt:    "a beautiful landscape",
		NumImages: 3,
		ModelID:   "model-abc",
		Width:     1024,
		Height:    768,
		Alchemy:   true,
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
	if receivedBody["width"] != 1024.0 {
		t.Errorf("expected width 1024, got %v", receivedBody["width"])
	}
	if receivedBody["height"] != 768.0 {
		t.Errorf("expected height 768, got %v", receivedBody["height"])
	}
	if receivedBody["alchemy"] != true {
		t.Errorf("expected alchemy true, got %v", receivedBody["alchemy"])
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
		Prompt:    "minimal request",
		NumImages: 1,
	}
	_, err := client.CreateGeneration(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// These optional fields should NOT be present in the payload
	for _, key := range []string{"modelId", "width", "height", "alchemy", "ultra", "styleUUID", "contrast", "guidance_scale"} {
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

	_, err := client.CreateGeneration(domain.GenerationRequest{Prompt: "test", NumImages: 1})
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
		Prompt:        "fully loaded request",
		NumImages:     2,
		ModelID:       "model-full",
		Width:         512,
		Height:        512,
		Alchemy:       true,
		Ultra:         true,
		StyleUUID:     "style-123",
		Contrast:      2.5,
		GuidanceScale: 8.0,
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
	if receivedBody["contrast"] != 2.5 {
		t.Errorf("expected contrast 2.5, got %v", receivedBody["contrast"])
	}
	if receivedBody["guidance_scale"] != 8.0 {
		t.Errorf("expected guidance_scale 8.0, got %v", receivedBody["guidance_scale"])
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
