// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &GroupDataSource{}

func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

// GroupDataSource defines the data source implementation.
type GroupDataSource struct {
	client *Client
}

// GroupDataSourceModel describes the data source data model.
type GroupDataSourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	PermTemplID types.Int64  `tfsdk:"perm_templ_id"`
	MemberCount types.Int64  `tfsdk:"member_count"`
	ZoneCount   types.Int64  `tfsdk:"zone_count"`
}

func (d *GroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *GroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a group in Poweradmin. You can look up a group by ID or name. Requires Poweradmin 4.2.0+.",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The group ID. Either id or name must be specified.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The group name. Either id or name must be specified.",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the group",
				Computed:            true,
			},
			"perm_templ_id": schema.Int64Attribute{
				MarkdownDescription: "Permission template ID assigned to the group",
				Computed:            true,
			},
			"member_count": schema.Int64Attribute{
				MarkdownDescription: "Number of members in the group",
				Computed:            true,
			},
			"zone_count": schema.Int64Attribute{
				MarkdownDescription: "Number of zones assigned to the group",
				Computed:            true,
			},
		},
	}
}

func (d *GroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !data.ID.IsNull() && data.ID.ValueInt64() != 0
	hasName := !data.Name.IsNull() && data.Name.ValueString() != ""

	if !hasID && !hasName {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'id' or 'name' must be specified to look up a group",
		)
		return
	}

	var group *Group
	var err error

	if hasID {
		groupID := int(data.ID.ValueInt64())

		tflog.Debug(ctx, "Looking up group by ID", map[string]interface{}{
			"id": groupID,
		})

		group, err = d.client.GetGroup(ctx, groupID)
	} else {
		tflog.Debug(ctx, "Looking up group by name", map[string]interface{}{
			"name": data.Name.ValueString(),
		})

		group, err = d.client.FindGroupByName(ctx, data.Name.ValueString())
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Group",
			fmt.Sprintf("Could not read group: %s", err.Error()),
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

	data.MemberCount = types.Int64Value(int64(group.MemberCount))
	data.ZoneCount = types.Int64Value(int64(group.ZoneCount))

	tflog.Trace(ctx, "Read group data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
