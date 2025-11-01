// Copyright (c) Poweradmin Development Team
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

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &PermissionDataSource{}

func NewPermissionDataSource() datasource.DataSource {
	return &PermissionDataSource{}
}

// PermissionDataSource defines the data source implementation.
type PermissionDataSource struct {
	client *Client
}

// PermissionDataSourceModel describes the data source data model.
type PermissionDataSourceModel struct {
	ID    types.Int64  `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Descr types.String `tfsdk:"descr"`
}

func (d *PermissionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

func (d *PermissionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a permission in Poweradmin. You can look up a permission by ID or name. Permissions are read-only and define what actions users can perform.",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The permission ID. Either id or name must be specified.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The permission name (e.g., 'zone_content_view_own'). Either id or name must be specified.",
				Optional:            true,
				Computed:            true,
			},
			"descr": schema.StringAttribute{
				MarkdownDescription: "Description of what this permission allows",
				Computed:            true,
			},
		},
	}
}

func (d *PermissionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (d *PermissionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PermissionDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that either ID or Name is provided
	hasID := !data.ID.IsNull()
	hasName := !data.Name.IsNull() && data.Name.ValueString() != ""

	if !hasID && !hasName {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'id' or 'name' must be specified to look up a permission",
		)
		return
	}

	var permission *Permission
	var err error

	if hasID {
		// Look up by ID
		permissionID := int(data.ID.ValueInt64())

		tflog.Debug(ctx, "Looking up permission by ID", map[string]interface{}{
			"id": permissionID,
		})

		permission, err = d.client.GetPermission(ctx, permissionID)
	} else {
		// Look up by name
		tflog.Debug(ctx, "Looking up permission by name", map[string]interface{}{
			"name": data.Name.ValueString(),
		})

		permission, err = d.client.FindPermissionByName(ctx, data.Name.ValueString())
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Permission",
			fmt.Sprintf("Could not read permission: %s", err.Error()),
		)
		return
	}

	// Update the model with the fetched data
	data.ID = types.Int64Value(int64(permission.ID))
	data.Name = types.StringValue(permission.Name)
	data.Descr = types.StringValue(permission.Descr)

	tflog.Debug(ctx, "Permission data source read successfully")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
