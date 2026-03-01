package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"leonardo-cli/internal/domain"
	"leonardo-cli/internal/provider"
	"leonardo-cli/internal/service"
)

// printUsage prints the top level usage instructions.
func printUsage() {
	program := os.Args[0]
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [options]\n", program)
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  create   Create a new image generation")
	fmt.Fprintln(os.Stderr, "  status   Check the status of an existing generation")
	fmt.Fprintln(os.Stderr, "  delete   Delete an existing generation")
	fmt.Fprintln(os.Stderr, "  me       Show account info and token balances")
	fmt.Fprintln(os.Stderr, "  list     List recent generations")
	fmt.Fprintln(os.Stderr, "  download Download images for a completed generation")
	fmt.Fprintln(os.Stderr, "  inspect  Inspect a sidecar metadata JSON file")
	fmt.Fprintln(os.Stderr, "Use \"", program, " <command> -h\" for more information about a command.")
}

// ensureAPIKey retrieves the API key from the environment and returns it.
func ensureAPIKey() (string, error) {
	key := os.Getenv("LEONARDO_API_TOKEN")
	if strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("environment variable LEONARDO_API_TOKEN is not set")
	}
	return key, nil
}

// defaultPrivateFromEnv returns whether image generations should default to private.
func defaultPrivateFromEnv() bool {
	privateValue := strings.TrimSpace(os.Getenv("LEONARDO_PRIVATE"))
	if privateValue == "" {
		return false
	}
	private, err := strconv.ParseBool(privateValue)
	if err != nil {
		return false
	}
	return private
}

// createGeneration wraps the service call to create a generation and outputs
// relevant information to the user.  It accepts a GenerationService and a
// GenerationRequest built from CLI flags.
func createGeneration(svc *service.GenerationService, req domain.GenerationRequest) error {
	res, err := svc.Create(req)
	if err != nil {
		return err
	}
	sidecarPath, err := writeSidecarMetadata(req, res.GenerationID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(res.GenerationID) != "" {
		fmt.Println("Generation ID:", res.GenerationID)
	}
	fmt.Println("Sidecar metadata:", sidecarPath)
	prettyPrintJSON(res.Raw)
	return nil
}

// checkGenerationStatus wraps the service call to obtain the status of a
// generation and outputs relevant information to the user.
func checkGenerationStatus(svc *service.GenerationService, id string) error {
	status, err := svc.Status(id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(status.Status) != "" {
		fmt.Println("Status:", status.Status)
	}
	for i, url := range status.Images {
		fmt.Printf("Image %d URL: %s\n", i+1, url)
	}
	prettyPrintJSON(status.Raw)
	return nil
}

// deleteGeneration wraps the service call to delete a generation and outputs
// the result to the user.
func deleteGeneration(svc *service.GenerationService, id string) error {
	resp, err := svc.Delete(id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(resp.ID) != "" {
		fmt.Println("Deleted generation:", resp.ID)
	}
	prettyPrintJSON(resp.Raw)
	return nil
}

// showUserInfo wraps the service call to retrieve account information and
// outputs it to the user.
func showUserInfo(svc *service.GenerationService) error {
	info, err := svc.UserInfo()
	if err != nil {
		return err
	}
	if strings.TrimSpace(info.UserID) != "" {
		fmt.Println("User ID:", info.UserID)
	}
	if strings.TrimSpace(info.Username) != "" {
		fmt.Println("Username:", info.Username)
	}
	fmt.Println("API Subscription Tokens:", info.APISubscriptionTokens)
	fmt.Println("API Paid Tokens:", info.APIPaidTokens)
	if strings.TrimSpace(info.TokenRenewalDate) != "" {
		fmt.Println("Token Renewal Date:", info.TokenRenewalDate)
	}
	prettyPrintJSON(info.Raw)
	return nil
}

// listGenerations wraps the service call to list user generations and outputs
// a summary to the user.
func listGenerations(svc *service.GenerationService, userID string, offset, limit int) error {
	resp, err := svc.ListGenerations(userID, offset, limit)
	if err != nil {
		return err
	}
	for _, gen := range resp.Generations {
		fmt.Printf("[%s] %s — %s", gen.Status, gen.ID, gen.Prompt)
		if len(gen.Images) > 0 {
			fmt.Printf(" (%d images)", len(gen.Images))
		}
		fmt.Println()
	}
	prettyPrintJSON(resp.Raw)
	return nil
}

// downloadImages wraps the service call to download all generated images for a
// generation and outputs the saved file paths to the user.
func downloadImages(svc *service.GenerationService, id, outputDir string) error {
	result, err := svc.Download(id, outputDir)
	if err != nil {
		return err
	}
	for i, fp := range result.FilePaths {
		fmt.Printf("Image %d saved: %s\n", i+1, fp)
	}
	return nil
}

// writeSidecarMetadata writes a JSON metadata sidecar file named
// {generationID}.json in the current directory.
func writeSidecarMetadata(req domain.GenerationRequest, generationID string) (string, error) {
	if strings.TrimSpace(generationID) == "" {
		return "", fmt.Errorf("generation ID is empty; cannot write sidecar metadata")
	}
	metadata := req.Metadata
	timestamp := time.Now().UTC().Format(time.RFC3339)
	sidecar := map[string]interface{}{
		"prompt":        metadata.Prompt,
		"num_images":    req.NumImages,
		"generation_id": generationID,
		"timestamp":     timestamp,
		"private":       req.Private,
		"alchemy":       metadata.Alchemy,
		"ultra":         metadata.Ultra,
	}
	if metadata.HasNegativePrompt() {
		sidecar["negative_prompt"] = metadata.NegativePrompt
	}
	if metadata.HasModelID() {
		sidecar["model_id"] = metadata.ModelID
	}
	if metadata.HasStyleUUID() {
		sidecar["style_uuid"] = metadata.StyleUUID
	}
	if metadata.HasSeed() {
		sidecar["seed"] = metadata.Seed
	}
	if metadata.HasWidth() {
		sidecar["width"] = metadata.Width
	}
	if metadata.HasHeight() {
		sidecar["height"] = metadata.Height
	}
	if metadata.HasTags() {
		sidecar["tags"] = metadata.Tags
	}
	if metadata.HasContrast() {
		sidecar["contrast"] = metadata.Contrast
	}
	if metadata.HasGuidanceScale() {
		sidecar["guidance_scale"] = metadata.GuidanceScale
	}
	data, err := json.MarshalIndent(sidecar, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding sidecar metadata: %w", err)
	}
	path := filepath.Join(".", fmt.Sprintf("%s.json", generationID))
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("writing sidecar metadata: %w", err)
	}
	return path, nil
}

// inspectSidecar loads and displays a sidecar metadata JSON file.
func inspectSidecar(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading sidecar metadata: %w", err)
	}
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing sidecar metadata: %w", err)
	}
	prettyPrintJSON(data)
	return nil
}

// parseTags converts a comma-separated tags value into a trimmed string slice.
func parseTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		tag := strings.TrimSpace(p)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

// prettyPrintJSON takes a raw JSON byte slice and prints it indented.
func prettyPrintJSON(data []byte) {
	var out bytes.Buffer
	if err := json.Indent(&out, data, "", "  "); err != nil {
		// If indentation fails, print raw data
		fmt.Println(string(data))
		return
	}
	fmt.Println(out.String())
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	apiKey, err := ensureAPIKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Construct the adapter and service once at program start.
	client := provider.NewAPIClient(apiKey, nil)
	svc := service.NewGenerationService(client)
	switch cmd {
	case "create":
		createCmd := flag.NewFlagSet("create", flag.ExitOnError)
		prompt := createCmd.String("prompt", "", "Text prompt for image generation (required)")
		negativePrompt := createCmd.String("negative-prompt", "", "Negative prompt to avoid undesired traits")
		modelId := createCmd.String("model-id", "", "Model ID to use for generation")
		width := createCmd.Int("width", 0, "Width of the generated image")
		height := createCmd.Int("height", 0, "Height of the generated image")
		numImages := createCmd.Int("num-images", 1, "Number of images to generate (1-8)")
		seed := createCmd.Int("seed", 0, "Optional generation seed")
		tags := createCmd.String("tags", "", "Optional comma-separated metadata tags")
		private := createCmd.Bool("private", defaultPrivateFromEnv(), "Generate private images (can be set with LEONARDO_PRIVATE)")
		alchemy := createCmd.Bool("alchemy", false, "Enable Alchemy for advanced generation")
		ultra := createCmd.Bool("ultra", false, "Enable ultra mode for high fidelity generation")
		styleUUID := createCmd.String("style-uuid", "", "Optional style UUID to influence generation")
		contrast := createCmd.Float64("contrast", 0.0, "Optional contrast adjustment (0-5)")
		guidanceScale := createCmd.Float64("guidance-scale", 0.0, "Optional guidance scale, typically between 1 and 10")
		// Parse flags
		createCmd.Parse(os.Args[2:])
		if strings.TrimSpace(*prompt) == "" {
			fmt.Fprintln(os.Stderr, "Error: --prompt is required")
			createCmd.Usage()
			os.Exit(1)
		}
		// Build a domain request object.
		req := domain.GenerationRequest{
			NumImages: *numImages,
			Private:   *private,
			Metadata: domain.GenerationMetadata{
				Prompt:         *prompt,
				NegativePrompt: *negativePrompt,
				ModelID:        *modelId,
				StyleUUID:      *styleUUID,
				Seed:           *seed,
				Width:          *width,
				Height:         *height,
				Tags:           parseTags(*tags),
				Alchemy:        *alchemy,
				Ultra:          *ultra,
				Contrast:       *contrast,
				GuidanceScale:  *guidanceScale,
			},
		}
		if err := createGeneration(svc, req); err != nil {
			fmt.Fprintln(os.Stderr, "Error creating generation:", err)
			os.Exit(1)
		}
	case "status":
		statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
		id := statusCmd.String("id", "", "Generation ID to check (required)")
		statusCmd.Parse(os.Args[2:])
		if strings.TrimSpace(*id) == "" {
			fmt.Fprintln(os.Stderr, "Error: --id is required")
			statusCmd.Usage()
			os.Exit(1)
		}
		if err := checkGenerationStatus(svc, *id); err != nil {
			fmt.Fprintln(os.Stderr, "Error checking status:", err)
			os.Exit(1)
		}
	case "delete":
		deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
		id := deleteCmd.String("id", "", "Generation ID to delete (required)")
		deleteCmd.Parse(os.Args[2:])
		if strings.TrimSpace(*id) == "" {
			fmt.Fprintln(os.Stderr, "Error: --id is required")
			deleteCmd.Usage()
			os.Exit(1)
		}
		if err := deleteGeneration(svc, *id); err != nil {
			fmt.Fprintln(os.Stderr, "Error deleting generation:", err)
			os.Exit(1)
		}
	case "me":
		if err := showUserInfo(svc); err != nil {
			fmt.Fprintln(os.Stderr, "Error getting user info:", err)
			os.Exit(1)
		}
	case "list":
		listCmd := flag.NewFlagSet("list", flag.ExitOnError)
		userID := listCmd.String("user-id", "", "User ID to list generations for (required, use 'me' command to find your ID)")
		offset := listCmd.Int("offset", 0, "Pagination offset")
		limit := listCmd.Int("limit", 10, "Number of generations to return")
		listCmd.Parse(os.Args[2:])
		if strings.TrimSpace(*userID) == "" {
			fmt.Fprintln(os.Stderr, "Error: --user-id is required (use 'me' command to find your user ID)")
			listCmd.Usage()
			os.Exit(1)
		}
		if err := listGenerations(svc, *userID, *offset, *limit); err != nil {
			fmt.Fprintln(os.Stderr, "Error listing generations:", err)
			os.Exit(1)
		}
	case "download":
		downloadCmd := flag.NewFlagSet("download", flag.ExitOnError)
		id := downloadCmd.String("id", "", "Generation ID to download images for (required)")
		outputDir := downloadCmd.String("output-dir", ".", "Directory to save downloaded images")
		downloadCmd.Parse(os.Args[2:])
		if strings.TrimSpace(*id) == "" {
			fmt.Fprintln(os.Stderr, "Error: --id is required")
			downloadCmd.Usage()
			os.Exit(1)
		}
		if err := downloadImages(svc, *id, *outputDir); err != nil {
			fmt.Fprintln(os.Stderr, "Error downloading images:", err)
			os.Exit(1)
		}
	case "inspect":
		inspectCmd := flag.NewFlagSet("inspect", flag.ExitOnError)
		filePath := inspectCmd.String("file", "", "Path to a sidecar metadata JSON file (required)")
		inspectCmd.Parse(os.Args[2:])
		if strings.TrimSpace(*filePath) == "" {
			fmt.Fprintln(os.Stderr, "Error: --file is required")
			inspectCmd.Usage()
			os.Exit(1)
		}
		if err := inspectSidecar(*filePath); err != nil {
			fmt.Fprintln(os.Stderr, "Error inspecting sidecar:", err)
			os.Exit(1)
		}
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}
