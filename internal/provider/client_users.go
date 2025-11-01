// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
)

// GetUser retrieves a user by ID.
func (c *Client) GetUser(ctx context.Context, userID int) (*User, error) {
	path := fmt.Sprintf("users/%d", userID)
	var result UserResponse
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result.User, nil
}

// ListUsers retrieves all users.
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	var users []User
	if err := c.Get(ctx, "users", &users); err != nil {
		return nil, err
	}
	return users, nil
}

// CreateUser creates a new user and returns the created user.
func (c *Client) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	var result CreateUserResponse
	if err := c.Post(ctx, "users", req, &result); err != nil {
		return nil, err
	}

	// Fetch the created user to get full details
	return c.GetUser(ctx, result.UserID)
}

// UpdateUser updates an existing user.
func (c *Client) UpdateUser(ctx context.Context, userID int, req UpdateUserRequest) (*User, error) {
	path := fmt.Sprintf("users/%d", userID)
	var result UserResponse
	if err := c.Put(ctx, path, req, &result); err != nil {
		return nil, err
	}

	// Fetch the updated user to get full details
	return c.GetUser(ctx, userID)
}

// DeleteUser deletes a user.
func (c *Client) DeleteUser(ctx context.Context, userID int, transferToUserID *int) error {
	path := fmt.Sprintf("users/%d", userID)

	if transferToUserID != nil {
		// Include transfer_to_user_id in request body
		body := map[string]int{
			"transfer_to_user_id": *transferToUserID,
		}
		// Use DeleteWithBody helper if it exists, otherwise implement directly
		return c.DeleteWithBody(ctx, path, body)
	}

	return c.Delete(ctx, path)
}

// FindUserByUsername finds a user by username.
func (c *Client) FindUserByUsername(ctx context.Context, username string) (*User, error) {
	users, err := c.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.Username == username {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("user not found: %s", username)
}
