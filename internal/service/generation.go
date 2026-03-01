package service

import (
	"fmt"
	"path/filepath"

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

// Download fetches the status of a generation and downloads all generated
// images to the specified output directory.  Files are named using the pattern
// {generationID}_{index}.png.  It returns an error if the generation is not
// complete or has no images.
func (s *GenerationService) Download(id, outputDir string) (domain.DownloadResult, error) {
	status, err := s.client.GetGenerationStatus(id)
	if err != nil {
		return domain.DownloadResult{}, err
	}
	if status.Status != "COMPLETE" {
		return domain.DownloadResult{}, fmt.Errorf("generation is not complete, current status: %s", status.Status)
	}
	if len(status.Images) == 0 {
		return domain.DownloadResult{}, fmt.Errorf("no images available for generation %s", id)
	}
	var filePaths []string
	for i, imgURL := range status.Images {
		destPath := filepath.Join(outputDir, fmt.Sprintf("%s_%d.png", id, i+1))
		if err := s.client.DownloadImage(imgURL, destPath); err != nil {
			return domain.DownloadResult{}, fmt.Errorf("downloading image %d: %w", i+1, err)
		}
		filePaths = append(filePaths, destPath)
	}
	return domain.DownloadResult{FilePaths: filePaths}, nil
}
