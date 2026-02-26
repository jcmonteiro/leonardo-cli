package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "strings"

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

// createGeneration wraps the service call to create a generation and outputs
// relevant information to the user.  It accepts a GenerationService and a
// GenerationRequest built from CLI flags.
func createGeneration(svc *service.GenerationService, req domain.GenerationRequest) error {
    res, err := svc.Create(req)
    if err != nil {
        return err
    }
    if strings.TrimSpace(res.GenerationID) != "" {
        fmt.Println("Generation ID:", res.GenerationID)
    }
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
        // Build a domain request object.
        req := domain.GenerationRequest{
            Prompt:        *prompt,
            ModelID:       *modelId,
            Width:         *width,
            Height:        *height,
            NumImages:     *numImages,
            Alchemy:       *alchemy,
            Ultra:         *ultra,
            StyleUUID:     *styleUUID,
            Contrast:      *contrast,
            GuidanceScale: *guidanceScale,
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
    case "help", "--help", "-h":
        printUsage()
    default:
        fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
        printUsage()
        os.Exit(1)
    }
}