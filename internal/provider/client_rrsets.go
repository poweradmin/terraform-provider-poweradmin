// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
)

// RRSetRecord represents a single record in an RRSet
type RRSetRecord struct {
	Content  string `json:"content"`
	Disabled bool   `json:"disabled"`
	Priority int64  `json:"priority"`
}

// RRSet represents a Resource Record Set
type RRSet struct {
	Name    string        `json:"name"`
	Type    string        `json:"type"`
	TTL     int64         `json:"ttl"`
	Records []RRSetRecord `json:"records"`
}

// RRSetData is used for unwrapping the API response for GetRRSet
type RRSetData struct {
	RRSet RRSet `json:"rrset"`
}

// ListRRSets retrieves all RRSets for a zone, with optional type filtering.
func (c *Client) ListRRSets(ctx context.Context, zoneID int64, recordType string) ([]RRSet, error) {
	path := fmt.Sprintf("zones/%d/rrsets", zoneID)
	if recordType != "" {
		path += "?type=" + recordType
	}
	var result []RRSet
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetRRSet retrieves a specific RRSet by zone ID, name, and type.
func (c *Client) GetRRSet(ctx context.Context, zoneID int64, name, recordType string) (*RRSet, error) {
	path := fmt.Sprintf("zones/%d/rrsets/%s/%s", zoneID, name, recordType)
	var result RRSetData
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result.RRSet, nil
}

// CreateRRSet creates or replaces an RRSet in a zone.
func (c *Client) CreateRRSet(ctx context.Context, zoneID int64, rrsetData map[string]interface{}) error {
	path := fmt.Sprintf("zones/%d/rrsets", zoneID)
	// Put returns the response, but for RRSet creation we just need to know if it succeeded
	// The Put method will return an error if the API returns success: false
	if err := c.Put(ctx, path, rrsetData, nil); err != nil {
		return err
	}
	return nil
}

// UpdateRRSet updates an existing RRSet (same as CreateRRSet since PUT replaces).
func (c *Client) UpdateRRSet(ctx context.Context, zoneID int64, rrsetData map[string]interface{}) error {
	path := fmt.Sprintf("zones/%d/rrsets", zoneID)
	if err := c.Put(ctx, path, rrsetData, nil); err != nil {
		return err
	}
	return nil
}

// DeleteRRSet deletes an RRSet.
func (c *Client) DeleteRRSet(ctx context.Context, zoneID int64, name, recordType string) error {
	path := fmt.Sprintf("zones/%d/rrsets/%s/%s", zoneID, name, recordType)
	return c.Delete(ctx, path)
}
