package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client communicates with the AI content generation service
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new AI service client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateRequest represents a content generation request
type GenerateRequest struct {
	ProductName   string  `json:"product_name"`
	Price         float64 `json:"price"`
	Category      string  `json:"category"`
	Platform      string  `json:"platform"`
	Persona       string  `json:"persona"`
	Format        string  `json:"format"`
	LinkPlacement string  `json:"link_placement"`
	ShortURL      string  `json:"short_url"`
}

// GenerateResponse represents the AI-generated content
type GenerateResponse struct {
	Content             string   `json:"content"`
	Hashtags            []string `json:"hashtags"`
	Persona             string   `json:"persona"`
	Format              string   `json:"format"`
	LinkPlacement       string   `json:"link_placement"`
	EstimatedEngagement string   `json:"estimated_engagement"`
}

// Generate calls the AI service to generate post content
func (c *Client) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call AI service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service returned status %d", resp.StatusCode)
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// Health checks if the AI service is available
func (c *Client) Health(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("AI service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AI service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
