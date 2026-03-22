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

var _ resource.Resource = &GroupZoneAssignmentResource{}
var _ resource.ResourceWithImportState = &GroupZoneAssignmentResource{}

func NewGroupZoneAssignmentResource() resource.Resource {
	return &GroupZoneAssignmentResource{}
}

// GroupZoneAssignmentResource defines the resource implementation.
type GroupZoneAssignmentResource struct {
	client *Client
}

// GroupZoneAssignmentResourceModel describes the resource data model.
type GroupZoneAssignmentResourceModel struct {
	ID      types.String `tfsdk:"id"`
	GroupID types.Int64  `tfsdk:"group_id"`
	ZoneID  types.Int64  `tfsdk:"zone_id"`
}

func (r *GroupZoneAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_zone_assignment"
}

func (r *GroupZoneAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Assigns a DNS zone to a Poweradmin group. All members of the group gain access to the zone. Requires Poweradmin 4.2.0+.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier in the format `group_id/zone_id`",
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
			"zone_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the zone to assign to the group",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *GroupZoneAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupZoneAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GroupZoneAssignmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := int(data.GroupID.ValueInt64())
	zoneID := int(data.ZoneID.ValueInt64())

	tflog.Debug(ctx, "Assigning zone to group", map[string]interface{}{
		"group_id": groupID,
		"zone_id":  zoneID,
	})

	err := r.client.AssignZoneToGroup(ctx, groupID, zoneID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Assigning Zone to Group",
			fmt.Sprintf("Could not assign zone %d to group %d: %s", zoneID, groupID, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%d", groupID, zoneID))

	tflog.Debug(ctx, "Zone assigned to group successfully")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupZoneAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GroupZoneAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := int(data.GroupID.ValueInt64())
	zoneID := int(data.ZoneID.ValueInt64())

	tflog.Debug(ctx, "Reading group zone assignment", map[string]interface{}{
		"group_id": groupID,
		"zone_id":  zoneID,
	})

	// Verify assignment by listing group zones
	zones, err := r.client.ListGroupZones(ctx, groupID)
	if err != nil {
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Group Zone Assignment",
			fmt.Sprintf("Could not read zones of group %d: %s", groupID, err.Error()),
		)
		return
	}

	// Check if zone is still assigned
	found := false
	for _, zone := range zones {
		if zone.ZoneID == zoneID {
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

func (r *GroupZoneAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes use RequiresReplace, so Update should never be called
	resp.Diagnostics.AddError(
		"Error Updating Group Zone Assignment",
		"Group zone assignment does not support in-place updates. All changes require replacement.",
	)
}

func (r *GroupZoneAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GroupZoneAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := int(data.GroupID.ValueInt64())
	zoneID := int(data.ZoneID.ValueInt64())

	tflog.Debug(ctx, "Unassigning zone from group", map[string]interface{}{
		"group_id": groupID,
		"zone_id":  zoneID,
	})

	err := r.client.UnassignZoneFromGroup(ctx, groupID, zoneID)
	if err != nil {
		if IsNotFoundError(err) {
			tflog.Info(ctx, "Zone assignment already removed, ignoring error", map[string]interface{}{
				"group_id": groupID,
				"zone_id":  zoneID,
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Unassigning Zone from Group",
			fmt.Sprintf("Could not unassign zone %d from group %d: %s", zoneID, groupID, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Zone unassigned from group successfully")
}

func (r *GroupZoneAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Error Importing Group Zone Assignment",
			"Import ID must be in the format 'group_id/zone_id'",
		)
		return
	}

	groupID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Group Zone Assignment",
			fmt.Sprintf("Could not parse group_id '%s': %s", parts[0], err.Error()),
		)
		return
	}

	zoneID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Group Zone Assignment",
			fmt.Sprintf("Could not parse zone_id '%s': %s", parts[1], err.Error()),
		)
		return
	}

	data := GroupZoneAssignmentResourceModel{
		ID:      types.StringValue(req.ID),
		GroupID: types.Int64Value(groupID),
		ZoneID:  types.Int64Value(zoneID),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
