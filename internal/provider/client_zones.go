// Copyright Poweradmin Development Team 2025, 2026
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

// GetZoneName returns the zone's name, memoized per client instance since
// zone names are immutable (renames require replacement).
func (c *Client) GetZoneName(ctx context.Context, zoneID int64) (string, error) {
	if cached, ok := c.zoneNames.Load(zoneID); ok {
		if name, ok := cached.(string); ok {
			return name, nil
		}
	}
	zone, err := c.GetZone(ctx, int(zoneID))
	if err != nil {
		return "", err
	}
	c.zoneNames.Store(zoneID, zone.Name)
	return zone.Name, nil
}

// FindZoneByName finds a zone by its name, matching case, a trailing dot and
// Unicode vs punycode spellings of the same name.
func (c *Client) FindZoneByName(ctx context.Context, name string) (*Zone, error) {
	zones, err := c.ListZones(ctx)
	if err != nil {
		return nil, err
	}

	// Exact match first, so a second spelling of the same name cannot win by list order.
	for _, zone := range zones {
		if zone.Name == name {
			return &zone, nil
		}
	}
	// Pre-4.5.2 data can hold a raw Unicode and a punycode copy of one name.
	var found *Zone
	for i := range zones {
		if !dnsNamesEqual(zones[i].Name, name) {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("zone name %s is ambiguous: it matches zones %q (ID %d) and %q (ID %d); look the zone up by ID", name, found.Name, found.ZoneID(), zones[i].Name, zones[i].ZoneID())
		}
		found = &zones[i]
	}
	if found != nil {
		return found, nil
	}

	return nil, fmt.Errorf("zone not found: %s", name)
}
