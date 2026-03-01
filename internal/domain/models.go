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
	Metadata       GenerationMetadata
}

// GenerationMetadata captures generation details stored in a local sidecar file. It is written when a generation request is created.
type GenerationMetadata struct {
	Prompt         string
	NegativePrompt string
	ModelID        string
	StyleUUID      string
	Seed           int
	Width          int
	Height         int
	NumImages      int
	GenerationID   string
	Timestamp      string
	Tags           []string
	Private        bool
	Alchemy        bool
	Ultra          bool
	Contrast       float64
	GuidanceScale  float64
}

// HasNegativePrompt indicates whether metadata contains a negative prompt value.
func (m GenerationMetadata) HasNegativePrompt() bool {
	return m.NegativePrompt != ""
}

// HasModelID indicates whether metadata contains a model identifier.
func (m GenerationMetadata) HasModelID() bool {
	return m.ModelID != ""
}

// HasStyleUUID indicates whether metadata contains a style UUID.
func (m GenerationMetadata) HasStyleUUID() bool {
	return m.StyleUUID != ""
}

// HasSeed indicates whether metadata contains a seed value.
func (m GenerationMetadata) HasSeed() bool {
	return m.Seed > 0
}

// HasWidth indicates whether metadata contains a width value.
func (m GenerationMetadata) HasWidth() bool {
	return m.Width > 0
}

// HasHeight indicates whether metadata contains a height value.
func (m GenerationMetadata) HasHeight() bool {
	return m.Height > 0
}

// HasTags indicates whether metadata contains one or more tags.
func (m GenerationMetadata) HasTags() bool {
	return len(m.Tags) > 0
}

// HasContrast indicates whether metadata contains a contrast value.
func (m GenerationMetadata) HasContrast() bool {
	return m.Contrast != 0
}

// HasGuidanceScale indicates whether metadata contains a guidance scale value.
func (m GenerationMetadata) HasGuidanceScale() bool {
	return m.GuidanceScale != 0
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
