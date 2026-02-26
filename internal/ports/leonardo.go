package ports

import "leonardo-cli/internal/domain"

// LeonardoClient defines the hexagonal port used by the application layer to
// interact with the Leonardo.Ai API.  Implementations of this interface may
// communicate over HTTP, mocks or other transports.
type LeonardoClient interface {
    // CreateGeneration initiates a new generation request and returns a response
    // containing the generation ID and raw response bytes.
    CreateGeneration(req domain.GenerationRequest) (domain.GenerationResponse, error)
    // GetGenerationStatus retrieves the status of a previously created generation
    // by its generation ID.  It returns the status string and any image URLs.
    GetGenerationStatus(id string) (domain.GenerationStatus, error)
}