package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "strings"
    "time"
)

// printUsage prints the top level usage instructions.
func printUsage() {
    program := os.Args[0]
    fmt.Fprintf(os.Stderr, "Usage: %s <command> [options]\n", program)
    fmt.Fprintln(os.Stderr, "Commands:")
    fmt.Fprintln(os.Stderr, "  create   Create a new image generation")
    fmt.Fprintln(os.Stderr, "  status   Check the status of an existing generation")
    fmt.Fprintln(os.Stderr, "Use \"", program, " <command> -h\" for more information about a command.")
}

// makeHTTPClient returns a new HTTP client with reasonable defaults.
func makeHTTPClient() *http.Client {
    return &http.Client{Timeout: 60 * time.Second}
}

// ensureAPIKey retrieves the API key from environment and returns it.
func ensureAPIKey() (string, error) {
    key := os.Getenv("LEONARDO_API_KEY")
    if strings.TrimSpace(key) == "" {
        return "", fmt.Errorf("environment variable LEONARDO_API_KEY is not set")
    }
    return key, nil
}

// createGeneration sends a POST request to the Leonardo API to start an image generation.
func createGeneration(params map[string]interface{}) error {
    apiKey, err := ensureAPIKey()
    if err != nil {
        return err
    }
    // Marshal the request body.
    body, err := json.Marshal(params)
    if err != nil {
        return fmt.Errorf("failed to encode request body: %w", err)
    }

    req, err := http.NewRequest("POST", "https://cloud.leonardo.ai/api/rest/v1/generations", bytes.NewBuffer(body))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")
    req.Header.Set("Authorization", "Bearer "+apiKey)

    client := makeHTTPClient()
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("HTTP request failed: %w", err)
    }
    defer resp.Body.Close()

    data, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read response: %w", err)
    }
    if resp.StatusCode >= 300 {
        return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(data))
    }
    // Attempt to parse and display the generation ID if present.
    var decoded map[string]interface{}
    if err := json.Unmarshal(data, &decoded); err == nil {
        // According to the API documentation, the generation ID is returned under sdGenerationJob.generationId.
        if job, ok := decoded["sdGenerationJob"].(map[string]interface{}); ok {
            if genID, ok := job["generationId"].(string); ok {
                fmt.Println("Generation ID:", genID)
            }
        }
    }
    // Print full response as prettified JSON for transparency.
    prettyPrintJSON(data)
    return nil
}

// checkGenerationStatus retrieves the status of a generation by ID and prints it.
func checkGenerationStatus(id string) error {
    apiKey, err := ensureAPIKey()
    if err != nil {
        return err
    }
    url := fmt.Sprintf("https://cloud.leonardo.ai/api/rest/v1/generations/%s", id)
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }
    req.Header.Set("Accept", "application/json")
    req.Header.Set("Authorization", "Bearer "+apiKey)
    client := makeHTTPClient()
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("HTTP request failed: %w", err)
    }
    defer resp.Body.Close()
    data, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read response: %w", err)
    }
    if resp.StatusCode >= 300 {
        return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(data))
    }
    // Attempt to extract status from common fields.
    var decoded map[string]interface{}
    if err := json.Unmarshal(data, &decoded); err == nil {
        // Newer API responses include generations_by_pk.status
        if gen, ok := decoded["generations_by_pk"].(map[string]interface{}); ok {
            if status, ok := gen["status"].(string); ok {
                fmt.Println("Status:", status)
            }
            // Print image URLs if available
            if images, ok := gen["generated_images"].([]interface{}); ok {
                for i, img := range images {
                    if imap, ok := img.(map[string]interface{}); ok {
                        if url, ok := imap["url"].(string); ok {
                            fmt.Printf("Image %d URL: %s\n", i+1, url)
                        }
                    }
                }
            }
        }
    }
    // Print full response for debugging.
    prettyPrintJSON(data)
    return nil
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
    switch cmd {
    case "create":
        createCmd := flag.NewFlagSet("create", flag.ExitOnError)
        prompt := createCmd.String("prompt", "", "Text prompt for image generation (required)")
        modelId := createCmd.String("model-id", "", "Model ID to use for generation")
        width := createCmd.Int("width", 0, "Width of the generated image")
        height := createCmd.Int("height", 0, "Height of the generated image")
        numImages := createCmd.Int("num-images", 1, "Number of images to generate (1-8)")
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
        // Build request parameters. Only include fields with non-zero values.
        body := make(map[string]interface{})
        body["prompt"] = *prompt
        body["num_images"] = *numImages
        if *modelId != "" {
            body["modelId"] = *modelId
        }
        if *width > 0 {
            body["width"] = *width
        }
        if *height > 0 {
            body["height"] = *height
        }
        if *alchemy {
            body["alchemy"] = true
        }
        if *ultra {
            body["ultra"] = true
        }
        if *styleUUID != "" {
            body["styleUUID"] = *styleUUID
        }
        if *contrast > 0 {
            body["contrast"] = *contrast
        }
        if *guidanceScale > 0 {
            body["guidance_scale"] = *guidanceScale
        }
        if err := createGeneration(body); err != nil {
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
        if err := checkGenerationStatus(*id); err != nil {
            fmt.Fprintln(os.Stderr, "Error checking status:", err)
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