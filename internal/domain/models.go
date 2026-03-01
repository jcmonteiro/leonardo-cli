package domain

// GenerationRequest defines the parameters necessary to start an image generation.
// Only a subset of Leonardo’s many parameters are exposed here; additional fields
// can be added as required.  Fields with zero values will be omitted from the
// request body by the provider layer.
type GenerationRequest struct {
	Prompt         string   // required text prompt
	NegativePrompt string   // optional negative prompt
	ModelID        string   // optional model identifier
	Width          int      // optional image width
	Height         int      // optional image height
	NumImages      int      // optional number of images (default 1)
	Seed           int      // optional generation seed
	Tags           []string // optional user-defined metadata tags
	Private        bool     // when true, request private images; false keeps API default visibility
	Alchemy        bool     // optional flag to enable Alchemy
	Ultra          bool     // optional flag to enable Ultra
	StyleUUID      string   // optional style UUID
	Contrast       float64  // optional contrast adjustment
	GuidanceScale  float64  // optional guidance scale
}

// GenerationMetadata captures generation details stored in a local sidecar JSON
// file when a generation request is created.
type GenerationMetadata struct {
	Prompt         string   `json:"prompt"`
	NegativePrompt string   `json:"negative_prompt,omitempty"`
	ModelID        string   `json:"model_id,omitempty"`
	StyleUUID      string   `json:"style_uuid,omitempty"`
	Seed           int      `json:"seed,omitempty"`
	Width          int      `json:"width,omitempty"`
	Height         int      `json:"height,omitempty"`
	NumImages      int      `json:"num_images,omitempty"`
	GenerationID   string   `json:"generation_id"`
	Timestamp      string   `json:"timestamp"`
	Tags           []string `json:"tags,omitempty"`
	Private        bool     `json:"private"`
	Alchemy        bool     `json:"alchemy"`
	Ultra          bool     `json:"ultra"`
	Contrast       float64  `json:"contrast,omitempty"`
	GuidanceScale  float64  `json:"guidance_scale,omitempty"`
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

// DeleteResponse represents the result of deleting a generation.
// The ID field contains the identifier of the deleted generation.
type DeleteResponse struct {
	ID  string
	Raw []byte
}

// UserInfo represents the authenticated user's account information including
// identifiers and token balances used for API credit tracking.
type UserInfo struct {
	UserID                string
	Username              string
	APISubscriptionTokens int
	APIPaidTokens         int
	TokenRenewalDate      string
	Raw                   []byte
}

// GenerationListItem represents a single generation in a list response.
// It contains a subset of generation metadata along with any image URLs.
type GenerationListItem struct {
	ID        string
	Status    string
	CreatedAt string
	Prompt    string
	Images    []string
}

// GenerationListResponse represents a paginated list of user generations.
type GenerationListResponse struct {
	Generations []GenerationListItem
	Raw         []byte
}

// DownloadResult represents the outcome of downloading generated images
// for a single generation.  It contains the list of file paths where images
// were saved.
type DownloadResult struct {
	FilePaths []string
}
