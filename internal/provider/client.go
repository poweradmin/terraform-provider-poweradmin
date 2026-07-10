// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
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
	APIVersion string // "v2" for Poweradmin 4.1.0+

	zoneNames sync.Map // zone ID (int64) → zone name, memoized for name normalization
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
	Timestamp string `json:"timestamp,omitempty"`
}

// APIError represents error information in API responses.
type APIError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// errorMessage picks the most specific message from the response,
// falling back to the caller-supplied default.
func (r *APIResponse) errorMessage(fallback string) string {
	if r.Error != nil && r.Error.Message != "" {
		return r.Error.Message
	}
	if r.Message != "" {
		return r.Message
	}
	return fallback
}

// Pagination represents pagination metadata.
type Pagination struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
	LastPage    int `json:"last_page"`
}

// NewClient creates a new Poweradmin API client.
func NewClient(config *PoweradminProviderModel) (*Client, error) {
	if config.ApiUrl.IsNull() || config.ApiUrl.ValueString() == "" {
		return nil, fmt.Errorf("api_url is required")
	}

	baseURL := config.ApiUrl.ValueString()
	// Ensure URL doesn't end with a slash
	baseURL = strings.TrimRight(baseURL, "/")

	// Validate URL shape early so typos fail here, not on the first API call
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid api_url: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("invalid api_url %q: must be an http(s) URL with a host, e.g. https://dns.example.com", baseURL)
	}

	// Determine API version
	apiVersion := "v2" // default to stable (4.1.0+)
	if !config.ApiVersion.IsNull() && config.ApiVersion.ValueString() != "" {
		apiVersion = config.ApiVersion.ValueString()
	}

	// Create HTTP client with timeout and TLS config
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		// Don't follow redirects: Go rewrites POST/PUT/DELETE into GETs on
		// 301/302/303, silently no-oping writes and forwarding credential
		// headers to the target. Surfacing the 3xx exposes a bad api_url.
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Configure TLS if insecure mode is enabled. This is an explicit, opt-in
	// escape hatch (insecure = true) for self-signed or internal endpoints;
	// it is off by default. TLS 1.2 is still enforced so a skipped-verify
	// connection cannot be downgraded to an older protocol.
	if !config.Insecure.IsNull() && config.Insecure.ValueBool() {
		// Clone the default transport so proxy settings and sane defaults survive
		defaultTransport, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			defaultTransport = &http.Transport{Proxy: http.ProxyFromEnvironment}
		}
		transport := defaultTransport.Clone()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // G402: opt-in via insecure provider attribute
			MinVersion:         tls.VersionTLS12,
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
// Uses /api/{version}/ where version is v2 (Poweradmin 4.1.0+).
func (c *Client) buildURL(path string) string {
	// Remove leading slash if present
	path = strings.TrimLeft(path, "/")

	// Use dynamic API version prefix
	return fmt.Sprintf("%s/api/%s/%s", c.BaseURL, c.APIVersion, path)
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

// maxResponseBytes caps how much of an API response body is read; Poweradmin
// responses are small JSON, so the cap only stops runaway/misrouted endpoints.
const maxResponseBytes = 1 << 20

// parseResponse parses the API response and handles errors.
func (c *Client) parseResponse(ctx context.Context, resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	tflog.Debug(ctx, "API response", map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(body),
	})

	// Handle 204 No Content - successful deletion with no response body
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	// Handle non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(body)
		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err == nil {
			msg = apiResp.errorMessage(msg)
		}
		return &apiHTTPError{StatusCode: resp.StatusCode, Message: msg}
	}

	// Parse response
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	if !apiResp.Success {
		return fmt.Errorf("API operation failed: %s", apiResp.errorMessage("unknown error"))
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

// DeleteWithBody sends a DELETE request with a JSON body.
func (c *Client) DeleteWithBody(ctx context.Context, path string, body interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, path, body)
	if err != nil {
		return err
	}
	return c.parseResponse(ctx, resp, nil)
}

// apiHTTPError is a non-2xx API response carrying the status code, so callers
// can branch on it without matching error strings.
type apiHTTPError struct {
	StatusCode int
	Message    string
}

func (e *apiHTTPError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// IsNotFoundError checks if an error is a 404 Not Found API response.
func IsNotFoundError(err error) bool {
	var apiErr *apiHTTPError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}
