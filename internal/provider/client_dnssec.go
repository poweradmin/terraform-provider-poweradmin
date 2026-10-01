// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
)

// ListDnssecKeys retrieves all DNSSEC keys of a zone.
func (c *Client) ListDnssecKeys(ctx context.Context, zoneID int) ([]DnssecKey, error) {
	var keys []DnssecKey
	if err := c.Get(ctx, fmt.Sprintf("zones/%d/dnssec/keys", zoneID), &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

// GetDnssecKey retrieves one DNSSEC key. A 404 means the zone or key is gone;
// a 502 means PowerDNS did not answer and must not be read as "gone".
func (c *Client) GetDnssecKey(ctx context.Context, zoneID, keyID int) (*DnssecKey, error) {
	var key DnssecKey
	if err := c.Get(ctx, fmt.Sprintf("zones/%d/dnssec/keys/%d", zoneID, keyID), &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// CreateDnssecKey adds a DNSSEC key to a zone and returns it.
func (c *Client) CreateDnssecKey(ctx context.Context, zoneID int, req CreateDnssecKeyRequest) (*DnssecKey, error) {
	var key DnssecKey
	if err := c.Post(ctx, fmt.Sprintf("zones/%d/dnssec/keys", zoneID), req, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// SetDnssecKeyActive activates or deactivates a DNSSEC key and returns it.
func (c *Client) SetDnssecKeyActive(ctx context.Context, zoneID, keyID int, active bool) (*DnssecKey, error) {
	var key DnssecKey
	path := fmt.Sprintf("zones/%d/dnssec/keys/%d", zoneID, keyID)
	if err := c.Patch(ctx, path, UpdateDnssecKeyRequest{Active: active}, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// DeleteDnssecKey deletes a DNSSEC key from a zone.
func (c *Client) DeleteDnssecKey(ctx context.Context, zoneID, keyID int) error {
	return c.Delete(ctx, fmt.Sprintf("zones/%d/dnssec/keys/%d", zoneID, keyID))
}

// GetZoneDnssec retrieves the signing status of a zone.
func (c *Client) GetZoneDnssec(ctx context.Context, zoneID int) (*ZoneDnssecStatus, error) {
	var status ZoneDnssecStatus
	if err := c.Get(ctx, fmt.Sprintf("zones/%d/dnssec", zoneID), &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// SetZoneDnssec signs or unsigns a zone and returns the resulting status.
// Unsigning deletes every key of the zone in PowerDNS.
func (c *Client) SetZoneDnssec(ctx context.Context, zoneID int, enabled bool) (*ZoneDnssecStatus, error) {
	var status ZoneDnssecStatus
	path := fmt.Sprintf("zones/%d/dnssec", zoneID)
	if err := c.Post(ctx, path, SetZoneDnssecRequest{Enabled: enabled}, &status); err != nil {
		return nil, err
	}
	return &status, nil
}
