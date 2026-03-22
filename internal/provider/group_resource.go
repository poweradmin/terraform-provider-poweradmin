// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &GroupResource{}
var _ resource.ResourceWithImportState = &GroupResource{}

func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

// GroupResource defines the resource implementation.
type GroupResource struct {
	client *Client
}

// GroupResourceModel describes the resource data model.
type GroupResourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	PermTemplID types.Int64  `tfsdk:"perm_templ_id"`
}

func (r *GroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *GroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a group in Poweradmin. Groups can have members and zones assigned to them. Requires Poweradmin 4.2.0+.",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for the group",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the group",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the group",
				Optional:            true,
			},
			"perm_templ_id": schema.Int64Attribute{
				MarkdownDescription: "Permission template ID for the group. Must reference a group-type permission template. Changing this forces recreation of the group.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *GroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := CreateGroupRequest{
		Name:        data.Name.ValueString(),
		PermTemplID: int(data.PermTemplID.ValueInt64()),
	}

	if !data.Description.IsNull() {
		createReq.Description = data.Description.ValueString()
	}

	tflog.Debug(ctx, "Creating group", map[string]interface{}{
		"name": createReq.Name,
	})

	groupID, err := r.client.CreateGroup(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Group",
			fmt.Sprintf("Could not create group: %s", err.Error()),
		)
		return
	}

	// Fetch the created group to get full details
	group, err := r.client.GetGroup(ctx, groupID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Group",
			fmt.Sprintf("Could not read created group ID %d: %s", groupID, err.Error()),
		)
		return
	}

	data.ID = types.Int64Value(int64(group.ID))
	data.Name = types.StringValue(group.Name)
	data.PermTemplID = types.Int64Value(int64(group.PermTemplID))
	if group.Description != "" {
		data.Description = types.StringValue(group.Description)
	}

	tflog.Debug(ctx, "Group created successfully", map[string]interface{}{
		"id": data.ID.ValueInt64(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := int(data.ID.ValueInt64())

	tflog.Debug(ctx, "Reading group", map[string]interface{}{
		"id": groupID,
	})

	group, err := r.client.GetGroup(ctx, groupID)
	if err != nil {
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Group",
			fmt.Sprintf("Could not read group ID %d: %s", groupID, err.Error()),
		)
		return
	}

	data.ID = types.Int64Value(int64(group.ID))
	data.Name = types.StringValue(group.Name)
	data.PermTemplID = types.Int64Value(int64(group.PermTemplID))
	if group.Description != "" {
		data.Description = types.StringValue(group.Description)
	} else {
		data.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data GroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := int(data.ID.ValueInt64())

	updateReq := UpdateGroupRequest{
		Name: data.Name.ValueString(),
	}

	if !data.Description.IsNull() {
		desc := data.Description.ValueString()
		updateReq.Description = &desc
	} else {
		// Explicitly send empty string to clear description on the server
		empty := ""
		updateReq.Description = &empty
	}

	tflog.Debug(ctx, "Updating group", map[string]interface{}{
		"id": groupID,
	})

	_, err := r.client.UpdateGroup(ctx, groupID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Group",
			fmt.Sprintf("Could not update group ID %d: %s", groupID, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Group updated successfully")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := int(data.ID.ValueInt64())

	tflog.Debug(ctx, "Deleting group", map[string]interface{}{
		"id": groupID,
	})

	err := r.client.DeleteGroup(ctx, groupID)
	if err != nil {
		if IsNotFoundError(err) {
			tflog.Info(ctx, "Group already deleted, ignoring error", map[string]interface{}{
				"id": groupID,
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Group",
			fmt.Sprintf("Could not delete group ID %d: %s", groupID, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Group deleted successfully")
}

func (r *GroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Group",
			fmt.Sprintf("Could not parse group ID '%s': %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
