// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
)

// BulkRecordOperations executes multiple record operations atomically.
// If any operation fails, all operations are rolled back.
func (c *Client) BulkRecordOperations(ctx context.Context, zoneID int64, req BulkRecordsRequest) (*BulkRecordsResponse, error) {
	path := fmt.Sprintf("zones/%d/records/bulk", zoneID)
	var result BulkRecordsResponse
	if err := c.Post(ctx, path, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
