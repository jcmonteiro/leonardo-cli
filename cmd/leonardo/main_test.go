package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"leonardo-cli/internal/domain"
)

func TestWriteSidecarMetadata_WritesExpectedJSON(t *testing.T) {
	tempDir := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting current working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("changing working directory: %v", err)
	}
	defer os.Chdir(origWD)

	metadata := domain.GenerationMetadata{
		Prompt:         "a lighthouse at dusk",
		NegativePrompt: "low quality",
		ModelID:        "model-123",
		StyleUUID:      "style-456",
		Seed:           99,
		Width:          1024,
		Height:         768,
		NumImages:      2,
		Tags:           []string{"landscape", "sunset"},
		Private:        true,
		Alchemy:        true,
		Ultra:          false,
		Contrast:       2.5,
		GuidanceScale:  7.0,
	}

	path, err := writeSidecarMetadata(metadata, "gen-abc")
	if err != nil {
		t.Fatalf("unexpected error writing sidecar: %v", err)
	}
	if filepath.Clean(path) != filepath.Clean("./gen-abc.json") {
		t.Errorf("expected sidecar path %q, got %q", "./gen-abc.json", path)
	}

	data, err := os.ReadFile(filepath.Join(tempDir, "gen-abc.json"))
	if err != nil {
		t.Fatalf("reading sidecar file: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parsing sidecar json: %v", err)
	}

	if got["prompt"] != metadata.Prompt {
		t.Errorf("expected prompt %q, got %v", metadata.Prompt, got["prompt"])
	}
	if got["negative_prompt"] != metadata.NegativePrompt {
		t.Errorf("expected negative_prompt %q, got %v", metadata.NegativePrompt, got["negative_prompt"])
	}
	if got["generation_id"] != "gen-abc" {
		t.Errorf("expected generation_id %q, got %v", "gen-abc", got["generation_id"])
	}
	timestamp, ok := got["timestamp"].(string)
	if !ok || strings.TrimSpace(timestamp) == "" {
		t.Fatal("expected non-empty timestamp")
	}
	if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
		t.Fatalf("expected RFC3339 timestamp, got %q: %v", timestamp, err)
	}
}

func TestInspectSidecar_PrintsSidecarJSON(t *testing.T) {
	tempDir := t.TempDir()
	sidecarPath := filepath.Join(tempDir, "gen-test.json")
	if err := os.WriteFile(sidecarPath, []byte(`{"generation_id":"gen-test","prompt":"hello"}`), 0644); err != nil {
		t.Fatalf("writing sidecar fixture: %v", err)
	}

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stdout pipe: %v", err)
	}
	os.Stdout = w

	callErr := inspectSidecar(sidecarPath)

	_ = w.Close()
	os.Stdout = originalStdout
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if callErr != nil {
		t.Fatalf("expected no error inspecting sidecar, got %v", callErr)
	}
	if !strings.Contains(buf.String(), `"generation_id": "gen-test"`) {
		t.Errorf("expected output to contain generation_id, got %q", buf.String())
	}
}

func TestInspectSidecar_ReturnsErrorForInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	sidecarPath := filepath.Join(tempDir, "invalid.json")
	if err := os.WriteFile(sidecarPath, []byte(`not-json`), 0644); err != nil {
		t.Fatalf("writing invalid sidecar fixture: %v", err)
	}

	err := inspectSidecar(sidecarPath)
	if err == nil {
		t.Fatal("expected error for invalid sidecar JSON, got nil")
	}
}

func TestParseTags_ParsesAndTrimsCommaSeparatedValues(t *testing.T) {
	got := parseTags(" tag1,tag2,  tag3 ,, ")
	if len(got) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(got))
	}
	if got[0] != "tag1" || got[1] != "tag2" || got[2] != "tag3" {
		t.Errorf("unexpected tags parsed: %#v", got)
	}
}
