// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZoneDataSource{}

func NewZoneDataSource() datasource.DataSource {
	return &ZoneDataSource{}
}

// ZoneDataSource defines the data source implementation.
type ZoneDataSource struct {
	client *Client
}

// ZoneDataSourceModel describes the data source data model.
type ZoneDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	Masters     types.String `tfsdk:"masters"`
	Account     types.String `tfsdk:"account"`
	Description types.String `tfsdk:"description"`
}

func (d *ZoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone"
}

func (d *ZoneDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a DNS zone in Poweradmin. You can look up a zone by ID or name.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The zone ID. Either id or name must be specified.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The zone name (e.g., example.com). Either id or name must be specified.",
				Optional:            true,
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Zone type (MASTER, SLAVE, or NATIVE)",
				Computed:            true,
			},
			"masters": schema.StringAttribute{
				MarkdownDescription: "Comma-separated list of master nameservers (for SLAVE zones)",
				Computed:            true,
			},
			"account": schema.StringAttribute{
				MarkdownDescription: "Account name for the zone",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the zone",
				Computed:            true,
			},
		},
	}
}

func (d *ZoneDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ZoneDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that either ID or Name is provided
	hasID := !data.ID.IsNull() && data.ID.ValueString() != ""
	hasName := !data.Name.IsNull() && data.Name.ValueString() != ""

	if !hasID && !hasName {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'id' or 'name' must be specified to look up a zone",
		)
		return
	}

	var zone *Zone
	var err error

	if hasID {
		// Look up by ID
		zoneID, parseErr := strconv.Atoi(data.ID.ValueString())
		if parseErr != nil {
			resp.Diagnostics.AddError(
				"Invalid Zone ID",
				fmt.Sprintf("Could not parse zone ID: %s", parseErr.Error()),
			)
			return
		}

		tflog.Debug(ctx, "Looking up zone by ID", map[string]interface{}{
			"id": zoneID,
		})

		zone, err = d.client.GetZone(ctx, zoneID)
	} else {
		// Look up by name
		tflog.Debug(ctx, "Looking up zone by name", map[string]interface{}{
			"name": data.Name.ValueString(),
		})

		zone, err = d.client.FindZoneByName(ctx, data.Name.ValueString())
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Zone",
			fmt.Sprintf("Could not read zone: %s", err.Error()),
		)
		return
	}

	// Map API response to data source model
	data.ID = types.StringValue(strconv.Itoa(zone.ID))
	data.Name = types.StringValue(zone.Name)
	data.Type = types.StringValue(zone.Type)

	if zone.Masters != "" {
		data.Masters = types.StringValue(zone.Masters)
	} else {
		data.Masters = types.StringNull()
	}

	if zone.Account != "" {
		data.Account = types.StringValue(zone.Account)
	} else {
		data.Account = types.StringNull()
	}

	if zone.Description != "" {
		data.Description = types.StringValue(zone.Description)
	} else {
		data.Description = types.StringNull()
	}

	tflog.Trace(ctx, "Read zone data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
