package domain

// GenerationRequest defines the parameters necessary to start an image generation.
// Only a subset of Leonardo’s many parameters are exposed here; additional fields
// can be added as required.  Fields with zero values will be omitted from the
// request body by the provider layer.
type GenerationRequest struct {
	NumImages int  // optional number of images (default 1)
	Private   bool // when true, request private images; false keeps API default visibility
	Metadata  GenerationMetadata
}

// HasNumImages indicates whether request includes an explicit number of images.
func (r GenerationRequest) HasNumImages() bool {
	return r.NumImages > 0
}

// NumImagesOrDefault returns the requested image count or the default value.
func (r GenerationRequest) NumImagesOrDefault() int {
	if r.HasNumImages() {
		return r.NumImages
	}
	return 1
}

// HasPrivate indicates whether request asks for private generation visibility.
func (r GenerationRequest) HasPrivate() bool {
	return r.Private
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
	Timestamp      string
	Tags           []string
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

// HasAlchemy indicates whether metadata contains Alchemy enabled.
func (m GenerationMetadata) HasAlchemy() bool {
	return m.Alchemy
}

// HasUltra indicates whether metadata contains Ultra enabled.
func (m GenerationMetadata) HasUltra() bool {
	return m.Ultra
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
