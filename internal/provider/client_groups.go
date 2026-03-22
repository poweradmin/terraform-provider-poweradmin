// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
)

// GetGroup retrieves a group by ID.
// The API returns the group directly in the data field (not wrapped).
func (c *Client) GetGroup(ctx context.Context, groupID int) (*Group, error) {
	path := fmt.Sprintf("groups/%d", groupID)
	var result Group
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListGroups retrieves all groups.
// The API returns a direct array of groups in the data field.
func (c *Client) ListGroups(ctx context.Context) ([]Group, error) {
	var groups []Group
	if err := c.Get(ctx, "groups", &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

// CreateGroup creates a new group and returns the group ID.
func (c *Client) CreateGroup(ctx context.Context, req CreateGroupRequest) (int, error) {
	var result CreateGroupResponse
	if err := c.Post(ctx, "groups", req, &result); err != nil {
		return 0, err
	}
	return result.ID, nil
}

// UpdateGroup updates an existing group.
// The API returns null data on success, so we re-fetch the group.
func (c *Client) UpdateGroup(ctx context.Context, groupID int, req UpdateGroupRequest) (*Group, error) {
	path := fmt.Sprintf("groups/%d", groupID)
	if err := c.Put(ctx, path, req, nil); err != nil {
		return nil, err
	}
	return c.GetGroup(ctx, groupID)
}

// DeleteGroup deletes a group.
func (c *Client) DeleteGroup(ctx context.Context, groupID int) error {
	path := fmt.Sprintf("groups/%d", groupID)
	return c.Delete(ctx, path)
}

// FindGroupByName finds a group by its name.
func (c *Client) FindGroupByName(ctx context.Context, name string) (*Group, error) {
	groups, err := c.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	for _, group := range groups {
		if group.Name == name {
			return &group, nil
		}
	}

	return nil, fmt.Errorf("group not found: %s", name)
}

// AddGroupMember adds a user to a group.
func (c *Client) AddGroupMember(ctx context.Context, groupID int, userID int) error {
	path := fmt.Sprintf("groups/%d/members", groupID)
	req := GroupMemberRequest{UserID: userID}
	return c.Post(ctx, path, req, nil)
}

// RemoveGroupMember removes a user from a group.
func (c *Client) RemoveGroupMember(ctx context.Context, groupID int, userID int) error {
	path := fmt.Sprintf("groups/%d/members/%d", groupID, userID)
	return c.Delete(ctx, path)
}

// ListGroupMembers lists all members of a group.
func (c *Client) ListGroupMembers(ctx context.Context, groupID int) ([]GroupMember, error) {
	path := fmt.Sprintf("groups/%d/members", groupID)
	var members []GroupMember
	if err := c.Get(ctx, path, &members); err != nil {
		return nil, err
	}
	return members, nil
}

// AssignZoneToGroup assigns a zone to a group.
func (c *Client) AssignZoneToGroup(ctx context.Context, groupID int, zoneID int) error {
	path := fmt.Sprintf("groups/%d/zones", groupID)
	req := GroupZoneRequest{ZoneID: zoneID}
	return c.Post(ctx, path, req, nil)
}

// UnassignZoneFromGroup removes a zone from a group.
func (c *Client) UnassignZoneFromGroup(ctx context.Context, groupID int, zoneID int) error {
	path := fmt.Sprintf("groups/%d/zones/%d", groupID, zoneID)
	return c.Delete(ctx, path)
}

// ListGroupZones lists all zones assigned to a group.
func (c *Client) ListGroupZones(ctx context.Context, groupID int) ([]GroupZone, error) {
	path := fmt.Sprintf("groups/%d/zones", groupID)
	var zones []GroupZone
	if err := c.Get(ctx, path, &zones); err != nil {
		return nil, err
	}
	return zones, nil
}
