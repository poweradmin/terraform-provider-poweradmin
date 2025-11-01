// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
)

// GetZone retrieves a zone by ID.
func (c *Client) GetZone(ctx context.Context, zoneID int) (*Zone, error) {
	path := fmt.Sprintf("zones/%d", zoneID)
	var result ZoneResponse
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result.Zone, nil
}

// ListZones retrieves all zones.
func (c *Client) ListZones(ctx context.Context) ([]Zone, error) {
	var result ZoneListResponse
	if err := c.Get(ctx, "zones", &result); err != nil {
		return nil, err
	}
	return result.Zones, nil
}

// CreateZone creates a new zone and returns the zone ID.
func (c *Client) CreateZone(ctx context.Context, req CreateZoneRequest) (int, error) {
	var result CreateZoneResponse
	if err := c.Post(ctx, "zones", req, &result); err != nil {
		return 0, err
	}
	return result.ZoneID, nil
}

// UpdateZone updates an existing zone.
func (c *Client) UpdateZone(ctx context.Context, zoneID int, req UpdateZoneRequest) (*Zone, error) {
	path := fmt.Sprintf("zones/%d", zoneID)
	var result ZoneResponse
	if err := c.Put(ctx, path, req, &result); err != nil {
		return nil, err
	}
	return &result.Zone, nil
}

// DeleteZone deletes a zone.
func (c *Client) DeleteZone(ctx context.Context, zoneID int) error {
	path := fmt.Sprintf("zones/%d", zoneID)
	return c.Delete(ctx, path)
}

// FindZoneByName finds a zone by its name.
func (c *Client) FindZoneByName(ctx context.Context, name string) (*Zone, error) {
	zones, err := c.ListZones(ctx)
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		if zone.Name == name {
			return &zone, nil
		}
	}

	return nil, fmt.Errorf("zone not found: %s", name)
}
