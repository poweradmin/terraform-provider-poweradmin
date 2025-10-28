// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
)

// GetRecord retrieves a record by zone ID and record ID.
func (c *Client) GetRecord(ctx context.Context, zoneID int, recordID int) (*Record, error) {
	path := fmt.Sprintf("zones/%d/records/%d", zoneID, recordID)
	var result RecordResponse
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result.Record, nil
}

// ListRecords retrieves all records for a zone.
func (c *Client) ListRecords(ctx context.Context, zoneID int) ([]Record, error) {
	path := fmt.Sprintf("zones/%d/records", zoneID)
	var result RecordListResponse
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result.Records, nil
}

// CreateRecord creates a new record in a zone.
func (c *Client) CreateRecord(ctx context.Context, zoneID int, req CreateRecordRequest) (*Record, error) {
	path := fmt.Sprintf("zones/%d/records", zoneID)
	var result RecordResponse
	if err := c.Post(ctx, path, req, &result); err != nil {
		return nil, err
	}
	return &result.Record, nil
}

// UpdateRecord updates an existing record.
func (c *Client) UpdateRecord(ctx context.Context, zoneID int, recordID int, req UpdateRecordRequest) (*Record, error) {
	path := fmt.Sprintf("zones/%d/records/%d", zoneID, recordID)
	var result RecordResponse
	if err := c.Put(ctx, path, req, &result); err != nil {
		return nil, err
	}
	return &result.Record, nil
}

// DeleteRecord deletes a record.
func (c *Client) DeleteRecord(ctx context.Context, zoneID int, recordID int) error {
	path := fmt.Sprintf("zones/%d/records/%d", zoneID, recordID)
	return c.Delete(ctx, path)
}
