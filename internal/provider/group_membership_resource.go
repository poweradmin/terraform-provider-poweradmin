// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &GroupMembershipResource{}
var _ resource.ResourceWithImportState = &GroupMembershipResource{}

func NewGroupMembershipResource() resource.Resource {
	return &GroupMembershipResource{}
}

// GroupMembershipResource defines the resource implementation.
type GroupMembershipResource struct {
	client *Client
}

// GroupMembershipResourceModel describes the resource data model.
type GroupMembershipResourceModel struct {
	ID      types.String `tfsdk:"id"`
	GroupID types.Int64  `tfsdk:"group_id"`
	UserID  types.Int64  `tfsdk:"user_id"`
}

func (r *GroupMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (r *GroupMembershipResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a user's membership in a Poweradmin group. Requires Poweradmin 4.2.0+.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier in the format `group_id/user_id`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the group",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the user to add to the group",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *GroupMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *GroupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GroupMembershipResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := int(data.GroupID.ValueInt64())
	userID := int(data.UserID.ValueInt64())

	tflog.Debug(ctx, "Adding member to group", map[string]interface{}{
		"group_id": groupID,
		"user_id":  userID,
	})

	err := r.client.AddGroupMember(ctx, groupID, userID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Adding Group Member",
			fmt.Sprintf("Could not add user %d to group %d: %s", userID, groupID, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%d", groupID, userID))

	tflog.Debug(ctx, "Group membership created successfully")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GroupMembershipResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := int(data.GroupID.ValueInt64())
	userID := int(data.UserID.ValueInt64())

	tflog.Debug(ctx, "Reading group membership", map[string]interface{}{
		"group_id": groupID,
		"user_id":  userID,
	})

	// Verify membership by listing group members
	members, err := r.client.ListGroupMembers(ctx, groupID)
	if err != nil {
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Group Membership",
			fmt.Sprintf("Could not read members of group %d: %s", groupID, err.Error()),
		)
		return
	}

	// Check if user is still a member
	found := false
	for _, member := range members {
		if member.UserID == userID {
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes use RequiresReplace, so Update should never be called
	resp.Diagnostics.AddError(
		"Error Updating Group Membership",
		"Group membership does not support in-place updates. All changes require replacement.",
	)
}

func (r *GroupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GroupMembershipResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := int(data.GroupID.ValueInt64())
	userID := int(data.UserID.ValueInt64())

	tflog.Debug(ctx, "Removing member from group", map[string]interface{}{
		"group_id": groupID,
		"user_id":  userID,
	})

	err := r.client.RemoveGroupMember(ctx, groupID, userID)
	if err != nil {
		if IsNotFoundError(err) {
			tflog.Info(ctx, "Group membership already removed, ignoring error", map[string]interface{}{
				"group_id": groupID,
				"user_id":  userID,
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Removing Group Member",
			fmt.Sprintf("Could not remove user %d from group %d: %s", userID, groupID, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Group membership deleted successfully")
}

func (r *GroupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Error Importing Group Membership",
			"Import ID must be in the format 'group_id/user_id'",
		)
		return
	}

	groupID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Group Membership",
			fmt.Sprintf("Could not parse group_id '%s': %s", parts[0], err.Error()),
		)
		return
	}

	userID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Group Membership",
			fmt.Sprintf("Could not parse user_id '%s': %s", parts[1], err.Error()),
		)
		return
	}

	data := GroupMembershipResourceModel{
		ID:      types.StringValue(req.ID),
		GroupID: types.Int64Value(groupID),
		UserID:  types.Int64Value(userID),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
