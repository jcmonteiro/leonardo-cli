package domain

// GenerationRequest defines the parameters necessary to start an image generation.
// Only a subset of Leonardo’s many parameters are exposed here; additional fields
// can be added as required.  Fields with zero values will be omitted from the
// request body by the provider layer.
type GenerationRequest struct {
    Prompt        string  // required text prompt
    ModelID       string  // optional model identifier
    Width         int     // optional image width
    Height        int     // optional image height
    NumImages     int     // optional number of images (default 1)
    Alchemy       bool    // optional flag to enable Alchemy
    Ultra         bool    // optional flag to enable Ultra
    StyleUUID     string  // optional style UUID
    Contrast      float64 // optional contrast adjustment
    GuidanceScale float64 // optional guidance scale
}

// GenerationResponse represents the response returned after creating a generation.
// It exposes the generation ID (if present) along with the raw JSON returned by the API.
type GenerationResponse struct {
    GenerationID string
    Raw          []byte
}

// GenerationStatus represents the status of a generation and any generated image URLs.
// The Raw field contains the full JSON payload returned by the API for transparency.
type GenerationStatus struct {
    Status string
    Images []string
    Raw    []byte
}