// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
	"sync"
)

// zoneMetadataPath is the API path of one metadata kind of a zone.
func zoneMetadataPath(zoneID int, kind string) string {
	return fmt.Sprintf("zones/%d/metadata/%s", zoneID, url.PathEscape(kind))
}

// lockZoneMetadata serializes metadata writes to one zone and returns the unlock
// function. Poweradmin's SQL backend rewrites every kind of a zone on each write,
// so two parallel writes to different kinds of the same zone lose one of them.
func (c *Client) lockZoneMetadata(zoneID int) func() {
	stored, _ := c.zoneMetadataLocks.LoadOrStore(zoneID, &sync.Mutex{})
	lock, _ := stored.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}

// ListZoneMetadata retrieves every metadata kind of a zone, sorted by kind.
func (c *Client) ListZoneMetadata(ctx context.Context, zoneID int) ([]ZoneMetadata, error) {
	var resp ZoneMetadataListResponse
	if err := c.Get(ctx, fmt.Sprintf("zones/%d/metadata", zoneID), &resp); err != nil {
		return nil, err
	}
	return resp.Metadata, nil
}

// GetZoneMetadata retrieves the values of one metadata kind. A 404 means the
// zone is gone or the kind has no values.
func (c *Client) GetZoneMetadata(ctx context.Context, zoneID int, kind string) (*ZoneMetadata, error) {
	var meta ZoneMetadata
	if err := c.Get(ctx, zoneMetadataPath(zoneID, kind), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// SetZoneMetadata creates a metadata kind or replaces all of its values.
func (c *Client) SetZoneMetadata(ctx context.Context, zoneID int, kind string, values []string) error {
	defer c.lockZoneMetadata(zoneID)()
	return c.Put(ctx, zoneMetadataPath(zoneID, kind), SetZoneMetadataRequest{Values: values}, nil)
}

// DeleteZoneMetadata deletes every value of a metadata kind.
func (c *Client) DeleteZoneMetadata(ctx context.Context, zoneID int, kind string) error {
	defer c.lockZoneMetadata(zoneID)()
	return c.Delete(ctx, zoneMetadataPath(zoneID, kind))
}
