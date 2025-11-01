// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
)

// GetPermission retrieves a permission by ID.
func (c *Client) GetPermission(ctx context.Context, permissionID int) (*Permission, error) {
	path := fmt.Sprintf("permissions/%d", permissionID)
	var result PermissionResponse
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result.Permission, nil
}

// ListPermissions retrieves all permissions.
func (c *Client) ListPermissions(ctx context.Context) ([]Permission, error) {
	var permissions []Permission
	if err := c.Get(ctx, "permissions", &permissions); err != nil {
		return nil, err
	}
	return permissions, nil
}

// FindPermissionByName finds a permission by name.
func (c *Client) FindPermissionByName(ctx context.Context, name string) (*Permission, error) {
	permissions, err := c.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}

	for _, permission := range permissions {
		if permission.Name == name {
			return &permission, nil
		}
	}

	return nil, fmt.Errorf("permission not found: %s", name)
}
