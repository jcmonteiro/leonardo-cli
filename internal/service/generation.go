package service

import (
    "leonardo-cli/internal/domain"
    "leonardo-cli/internal/ports"
)

// GenerationService provides a clean application layer for starting and
// monitoring image generations.  It depends on a LeonardoClient port which
// abstracts the underlying API.
type GenerationService struct {
    client ports.LeonardoClient
}

// NewGenerationService constructs a new GenerationService given a client.
func NewGenerationService(client ports.LeonardoClient) *GenerationService {
    return &GenerationService{client: client}
}

// Create starts a new generation by delegating to the underlying client.
func (s *GenerationService) Create(req domain.GenerationRequest) (domain.GenerationResponse, error) {
    return s.client.CreateGeneration(req)
}

// Status retrieves the status of an existing generation by delegating to the client.
func (s *GenerationService) Status(id string) (domain.GenerationStatus, error) {
    return s.client.GetGenerationStatus(id)
}

// Delete removes a generation by its ID by delegating to the client.
func (s *GenerationService) Delete(id string) (domain.DeleteResponse, error) {
    return s.client.DeleteGeneration(id)
}

// UserInfo retrieves the authenticated user's account information by delegating to the client.
func (s *GenerationService) UserInfo() (domain.UserInfo, error) {
    return s.client.GetUserInfo()
}

// ListGenerations returns a paginated list of generations for a user by delegating to the client.
func (s *GenerationService) ListGenerations(userID string, offset, limit int) (domain.GenerationListResponse, error) {
    return s.client.ListGenerations(userID, offset, limit)
}