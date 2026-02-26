package provider

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "time"

    "leonardo-cli/internal/domain"
    "leonardo-cli/internal/ports"
)

// APIClient is a concrete implementation of the LeonardoClient port that
// communicates with the Leonardo.Ai REST API over HTTP.
type APIClient struct {
    apiKey string
    // HTTP client is configurable to allow overriding timeouts in tests.
    httpClient *http.Client
}

// NewAPIClient constructs a new APIClient.  The apiKey must be a valid
// Leonardo.Ai API key.  If httpClient is nil, a client with a 60 second
// timeout will be used.
func NewAPIClient(apiKey string, httpClient *http.Client) *APIClient {
    if httpClient == nil {
        httpClient = &http.Client{Timeout: 60 * time.Second}
    }
    return &APIClient{apiKey: apiKey, httpClient: httpClient}
}

// CreateGeneration implements the LeonardoClient interface.  It builds a JSON
// payload from the GenerationRequest and issues a POST to the /generations
// endpoint.  The response body is returned in the Raw field and the
// generation ID (if any) is extracted.
func (c *APIClient) CreateGeneration(req domain.GenerationRequest) (domain.GenerationResponse, error) {
    bodyMap := map[string]interface{}{
        "prompt":    req.Prompt,
        "num_images": req.NumImages,
    }
    if req.ModelID != "" {
        bodyMap["modelId"] = req.ModelID
    }
    if req.Width > 0 {
        bodyMap["width"] = req.Width
    }
    if req.Height > 0 {
        bodyMap["height"] = req.Height
    }
    if req.Alchemy {
        bodyMap["alchemy"] = true
    }
    if req.Ultra {
        bodyMap["ultra"] = true
    }
    if req.StyleUUID != "" {
        bodyMap["styleUUID"] = req.StyleUUID
    }
    if req.Contrast > 0 {
        bodyMap["contrast"] = req.Contrast
    }
    if req.GuidanceScale > 0 {
        bodyMap["guidance_scale"] = req.GuidanceScale
    }
    // Marshal payload
    payload, err := json.Marshal(bodyMap)
    if err != nil {
        return domain.GenerationResponse{}, fmt.Errorf("encoding request body: %w", err)
    }
    httpReq, err := http.NewRequest("POST", "https://cloud.leonardo.ai/api/rest/v1/generations", bytes.NewBuffer(payload))
    if err != nil {
        return domain.GenerationResponse{}, fmt.Errorf("creating request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Accept", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return domain.GenerationResponse{}, fmt.Errorf("executing request: %w", err)
    }
    defer resp.Body.Close()
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return domain.GenerationResponse{}, fmt.Errorf("reading response: %w", err)
    }
    if resp.StatusCode >= 300 {
        return domain.GenerationResponse{Raw: bodyBytes}, fmt.Errorf("API returned status %d", resp.StatusCode)
    }
    var decoded map[string]interface{}
    genID := ""
    if err := json.Unmarshal(bodyBytes, &decoded); err == nil {
        if job, ok := decoded["sdGenerationJob"].(map[string]interface{}); ok {
            if id, ok := job["generationId"].(string); ok {
                genID = id
            }
        }
    }
    return domain.GenerationResponse{GenerationID: genID, Raw: bodyBytes}, nil
}

// GetGenerationStatus implements the LeonardoClient interface.  It issues a
// GET request to the /generations/{id} endpoint and attempts to parse the
// status and image URLs.  The raw JSON is always included in the returned
// GenerationStatus.
func (c *APIClient) GetGenerationStatus(id string) (domain.GenerationStatus, error) {
    url := fmt.Sprintf("https://cloud.leonardo.ai/api/rest/v1/generations/%s", id)
    httpReq, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return domain.GenerationStatus{}, fmt.Errorf("creating request: %w", err)
    }
    httpReq.Header.Set("Accept", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return domain.GenerationStatus{}, fmt.Errorf("executing request: %w", err)
    }
    defer resp.Body.Close()
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return domain.GenerationStatus{}, fmt.Errorf("reading response: %w", err)
    }
    if resp.StatusCode >= 300 {
        return domain.GenerationStatus{Raw: bodyBytes}, fmt.Errorf("API returned status %d", resp.StatusCode)
    }
    status := domain.GenerationStatus{Raw: bodyBytes}
    var decoded map[string]interface{}
    if err := json.Unmarshal(bodyBytes, &decoded); err == nil {
        // Newer API responses structure the generation under generations_by_pk
        if gen, ok := decoded["generations_by_pk"].(map[string]interface{}); ok {
            if s, ok := gen["status"].(string); ok {
                status.Status = s
            }
            if imgs, ok := gen["generated_images"].([]interface{}); ok {
                for _, item := range imgs {
                    if im, ok := item.(map[string]interface{}); ok {
                        if url, ok := im["url"].(string); ok {
                            status.Images = append(status.Images, url)
                        }
                    }
                }
            }
        }
    }
    return status, nil
}

// DeleteGeneration implements the LeonardoClient interface.  It issues a
// DELETE request to the /generations/{id} endpoint.  The raw JSON is always
// included in the returned DeleteResponse.
func (c *APIClient) DeleteGeneration(id string) (domain.DeleteResponse, error) {
    url := fmt.Sprintf("https://cloud.leonardo.ai/api/rest/v1/generations/%s", id)
    httpReq, err := http.NewRequest("DELETE", url, nil)
    if err != nil {
        return domain.DeleteResponse{}, fmt.Errorf("creating request: %w", err)
    }
    httpReq.Header.Set("Accept", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return domain.DeleteResponse{}, fmt.Errorf("executing request: %w", err)
    }
    defer resp.Body.Close()
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return domain.DeleteResponse{}, fmt.Errorf("reading response: %w", err)
    }
    if resp.StatusCode >= 300 {
        return domain.DeleteResponse{Raw: bodyBytes}, fmt.Errorf("API returned status %d", resp.StatusCode)
    }
    result := domain.DeleteResponse{Raw: bodyBytes}
    var decoded map[string]interface{}
    if err := json.Unmarshal(bodyBytes, &decoded); err == nil {
        if del, ok := decoded["delete_generations_by_pk"].(map[string]interface{}); ok {
            if delID, ok := del["id"].(string); ok {
                result.ID = delID
            }
        }
    }
    return result, nil
}

// GetUserInfo implements the LeonardoClient interface.  It issues a GET
// request to the /me endpoint to retrieve the authenticated user's account
// information including token balances.
func (c *APIClient) GetUserInfo() (domain.UserInfo, error) {
    httpReq, err := http.NewRequest("GET", "https://cloud.leonardo.ai/api/rest/v1/me", nil)
    if err != nil {
        return domain.UserInfo{}, fmt.Errorf("creating request: %w", err)
    }
    httpReq.Header.Set("Accept", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return domain.UserInfo{}, fmt.Errorf("executing request: %w", err)
    }
    defer resp.Body.Close()
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return domain.UserInfo{}, fmt.Errorf("reading response: %w", err)
    }
    if resp.StatusCode >= 300 {
        return domain.UserInfo{Raw: bodyBytes}, fmt.Errorf("API returned status %d", resp.StatusCode)
    }
    info := domain.UserInfo{Raw: bodyBytes}
    var decoded map[string]interface{}
    if err := json.Unmarshal(bodyBytes, &decoded); err == nil {
        if details, ok := decoded["user_details"].([]interface{}); ok && len(details) > 0 {
            if detail, ok := details[0].(map[string]interface{}); ok {
                if user, ok := detail["user"].(map[string]interface{}); ok {
                    if id, ok := user["id"].(string); ok {
                        info.UserID = id
                    }
                    if name, ok := user["username"].(string); ok {
                        info.Username = name
                    }
                }
                if tokens, ok := detail["apiSubscriptionTokens"].(float64); ok {
                    info.APISubscriptionTokens = int(tokens)
                }
                if tokens, ok := detail["apiPaidTokens"].(float64); ok {
                    info.APIPaidTokens = int(tokens)
                }
                if date, ok := detail["apiPlanTokenRenewalDate"].(string); ok {
                    info.TokenRenewalDate = date
                }
            }
        }
    }
    return info, nil
}

// ListGenerations implements the LeonardoClient interface.  It issues a GET
// request to the /generations/user/{userId} endpoint with pagination query
// parameters.  The raw JSON is always included in the returned response.
func (c *APIClient) ListGenerations(userID string, offset, limit int) (domain.GenerationListResponse, error) {
    url := fmt.Sprintf("https://cloud.leonardo.ai/api/rest/v1/generations/user/%s?offset=%d&limit=%d", userID, offset, limit)
    httpReq, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return domain.GenerationListResponse{}, fmt.Errorf("creating request: %w", err)
    }
    httpReq.Header.Set("Accept", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return domain.GenerationListResponse{}, fmt.Errorf("executing request: %w", err)
    }
    defer resp.Body.Close()
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return domain.GenerationListResponse{}, fmt.Errorf("reading response: %w", err)
    }
    if resp.StatusCode >= 300 {
        return domain.GenerationListResponse{Raw: bodyBytes}, fmt.Errorf("API returned status %d", resp.StatusCode)
    }
    result := domain.GenerationListResponse{Raw: bodyBytes}
    var decoded map[string]interface{}
    if err := json.Unmarshal(bodyBytes, &decoded); err == nil {
        if gens, ok := decoded["generations"].([]interface{}); ok {
            for _, g := range gens {
                if gen, ok := g.(map[string]interface{}); ok {
                    item := domain.GenerationListItem{}
                    if id, ok := gen["id"].(string); ok {
                        item.ID = id
                    }
                    if s, ok := gen["status"].(string); ok {
                        item.Status = s
                    }
                    if ca, ok := gen["createdAt"].(string); ok {
                        item.CreatedAt = ca
                    }
                    if p, ok := gen["prompt"].(string); ok {
                        item.Prompt = p
                    }
                    if imgs, ok := gen["generated_images"].([]interface{}); ok {
                        for _, img := range imgs {
                            if im, ok := img.(map[string]interface{}); ok {
                                if u, ok := im["url"].(string); ok {
                                    item.Images = append(item.Images, u)
                                }
                            }
                        }
                    }
                    result.Generations = append(result.Generations, item)
                }
            }
        }
    }
    return result, nil
}

// Ensure APIClient satisfies the LeonardoClient interface at compile time.
var _ ports.LeonardoClient = (*APIClient)(nil)