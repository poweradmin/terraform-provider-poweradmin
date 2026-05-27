// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
)

// ListZoneTemplates retrieves all zone templates visible to the caller.
func (c *Client) ListZoneTemplates(ctx context.Context) ([]ZoneTemplate, error) {
	var result []ZoneTemplate
	if err := c.Get(ctx, "zone-templates", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetZoneTemplate retrieves a zone template by ID (includes records).
func (c *Client) GetZoneTemplate(ctx context.Context, templateID int) (*ZoneTemplate, error) {
	path := fmt.Sprintf("zone-templates/%d", templateID)
	var result ZoneTemplate
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateZoneTemplate creates a new zone template and returns its ID.
func (c *Client) CreateZoneTemplate(ctx context.Context, req CreateZoneTemplateRequest) (int, error) {
	var result createResponseID
	if err := c.Post(ctx, "zone-templates", req, &result); err != nil {
		return 0, err
	}
	return result.ID, nil
}

// UpdateZoneTemplate updates an existing zone template.
func (c *Client) UpdateZoneTemplate(ctx context.Context, templateID int, req UpdateZoneTemplateRequest) error {
	path := fmt.Sprintf("zone-templates/%d", templateID)
	return c.Put(ctx, path, req, nil)
}

// DeleteZoneTemplate deletes a zone template.
func (c *Client) DeleteZoneTemplate(ctx context.Context, templateID int) error {
	path := fmt.Sprintf("zone-templates/%d", templateID)
	return c.Delete(ctx, path)
}

// FindZoneTemplateByName finds a zone template by its name.
func (c *Client) FindZoneTemplateByName(ctx context.Context, name string) (*ZoneTemplate, error) {
	templates, err := c.ListZoneTemplates(ctx)
	if err != nil {
		return nil, err
	}

	for _, tmpl := range templates {
		if tmpl.Name == name {
			return &tmpl, nil
		}
	}

	return nil, fmt.Errorf("zone template not found: %s", name)
}

// ListZoneTemplateRecords retrieves all records in a zone template.
func (c *Client) ListZoneTemplateRecords(ctx context.Context, templateID int) ([]ZoneTemplateRecord, error) {
	path := fmt.Sprintf("zone-templates/%d/records", templateID)
	var result []ZoneTemplateRecord
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetZoneTemplateRecord retrieves a single record from a zone template.
func (c *Client) GetZoneTemplateRecord(ctx context.Context, templateID, recordID int) (*ZoneTemplateRecord, error) {
	path := fmt.Sprintf("zone-templates/%d/records/%d", templateID, recordID)
	var result ZoneTemplateRecord
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateZoneTemplateRecord creates a new record inside a zone template
// and returns its ID.
func (c *Client) CreateZoneTemplateRecord(ctx context.Context, templateID int, req CreateZoneTemplateRecordRequest) (int, error) {
	path := fmt.Sprintf("zone-templates/%d/records", templateID)
	var result createResponseID
	if err := c.Post(ctx, path, req, &result); err != nil {
		return 0, err
	}
	return result.ID, nil
}

// UpdateZoneTemplateRecord updates a record inside a zone template.
func (c *Client) UpdateZoneTemplateRecord(ctx context.Context, templateID, recordID int, req UpdateZoneTemplateRecordRequest) error {
	path := fmt.Sprintf("zone-templates/%d/records/%d", templateID, recordID)
	return c.Put(ctx, path, req, nil)
}

// DeleteZoneTemplateRecord deletes a record from a zone template.
func (c *Client) DeleteZoneTemplateRecord(ctx context.Context, templateID, recordID int) error {
	path := fmt.Sprintf("zone-templates/%d/records/%d", templateID, recordID)
	return c.Delete(ctx, path)
}
