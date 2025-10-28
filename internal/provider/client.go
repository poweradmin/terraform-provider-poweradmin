// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Client represents a Poweradmin API client.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string
	Username   string
	Password   string
	APIVersion string // "v1" for stable (4.0.x), "dev" for development (master)
}

// APIResponse represents a standard Poweradmin API response.
type APIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
	Meta    *APIMeta        `json:"meta,omitempty"`
	Error   *APIError       `json:"error,omitempty"`
}

// APIMeta represents metadata in API responses.
type APIMeta struct {
	Timestamp string                 `json:"timestamp,omitempty"`
	Extra     map[string]interface{} `json:",inline"`
}

// APIError represents error information in API responses.
type APIError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Pagination represents pagination metadata.
type Pagination struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	TotalPages  int `json:"total_pages"`
	TotalItems  int `json:"total_items"`
}

// NewClient creates a new Poweradmin API client.
func NewClient(config *PoweradminProviderModel) (*Client, error) {
	if config.ApiUrl.IsNull() || config.ApiUrl.ValueString() == "" {
		return nil, fmt.Errorf("api_url is required")
	}

	baseURL := config.ApiUrl.ValueString()
	// Ensure URL doesn't end with a slash
	baseURL = strings.TrimRight(baseURL, "/")

	// Validate URL
	_, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid api_url: %w", err)
	}

	// Determine API version
	apiVersion := "v1" // default to stable
	if !config.ApiVersion.IsNull() && config.ApiVersion.ValueString() != "" {
		apiVersion = config.ApiVersion.ValueString()
	}

	// Create HTTP client with timeout and TLS config
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Configure TLS if insecure mode is enabled
	if !config.Insecure.IsNull() && config.Insecure.ValueBool() {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		httpClient.Transport = transport
	}

	client := &Client{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
		APIVersion: apiVersion,
	}

	// Set authentication
	if !config.ApiKey.IsNull() && config.ApiKey.ValueString() != "" {
		client.APIKey = config.ApiKey.ValueString()
	} else if !config.Username.IsNull() && config.Username.ValueString() != "" {
		client.Username = config.Username.ValueString()
		if !config.Password.IsNull() {
			client.Password = config.Password.ValueString()
		}
	} else {
		return nil, fmt.Errorf("either api_key or username/password must be provided")
	}

	return client, nil
}

// buildURL constructs the full URL for an API endpoint.
// For v1 API (stable): /api/v1/{path}.
// For dev API (master): /api/v1/{path} (same structure, might have additional features).
func (c *Client) buildURL(path string) string {
	// Remove leading slash if present
	path = strings.TrimLeft(path, "/")

	// Always use /api/v1/ prefix as both versions use the same API structure
	return fmt.Sprintf("%s/api/v1/%s", c.BaseURL, path)
}

// doRequest executes an HTTP request with authentication and returns the response.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := c.buildURL(path)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
		tflog.Debug(ctx, "Request body", map[string]interface{}{
			"body": string(jsonBody),
		})
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add authentication
	if c.APIKey != "" {
		// Prefer API key authentication
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
		req.Header.Set("X-API-Key", c.APIKey)
	} else if c.Username != "" {
		// Fall back to basic auth
		req.SetBasicAuth(c.Username, c.Password)
	}

	tflog.Debug(ctx, "Making API request", map[string]interface{}{
		"method":      method,
		"url":         url,
		"api_version": c.APIVersion,
	})

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// parseResponse parses the API response and handles errors.
func (c *Client) parseResponse(ctx context.Context, resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	tflog.Debug(ctx, "API response", map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(body),
	})

	// Handle non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.Error != nil {
			return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, apiResp.Error.Message)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	if !apiResp.Success {
		errMsg := apiResp.Message
		if apiResp.Error != nil {
			errMsg = apiResp.Error.Message
		}
		return fmt.Errorf("API operation failed: %s", errMsg)
	}

	// Unmarshal data into result if provided
	if result != nil && apiResp.Data != nil {
		if err := json.Unmarshal(apiResp.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	return nil
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	return c.parseResponse(ctx, resp, result)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.parseResponse(ctx, resp, result)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}
	return c.parseResponse(ctx, resp, result)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.parseResponse(ctx, resp, nil)
}

// IsNotFoundError checks if an error is a 404 Not Found error.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "HTTP 404") || strings.Contains(errStr, "404")
}
