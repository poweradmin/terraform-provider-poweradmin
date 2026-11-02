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

var _ datasource.DataSource = &ZoneMetadataDataSource{}

func NewZoneMetadataDataSource() datasource.DataSource {
	return &ZoneMetadataDataSource{}
}

// ZoneMetadataDataSource reads the values of one metadata kind of a zone.
type ZoneMetadataDataSource struct {
	client *Client
}

// ZoneMetadataDataSourceModel describes the data source data model.
type ZoneMetadataDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	ZoneID types.Int64  `tfsdk:"zone_id"`
	Kind   types.String `tfsdk:"kind"`
	Values types.List   `tfsdk:"values"`
}

func (d *ZoneMetadataDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_metadata"
}

func (d *ZoneMetadataDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the values of one PowerDNS metadata kind of a zone, such as `ALLOW-AXFR-FROM`. " +
			"Reading a kind that has no values is an error; use `poweradmin_zone_metadata_entries` to list the kinds a zone has. Requires Poweradmin 4.3.0+.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier in the format `zone_id/kind`",
			},
			"zone_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "ID of the zone",
			},
			"kind": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Metadata kind, e.g. `ALLOW-AXFR-FROM`; Poweradmin matches it case-insensitively",
			},
			"values": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Values of the kind, in the order the server returns them",
			},
		},
	}
}

func (d *ZoneMetadataDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZoneMetadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ZoneMetadataDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()
	kind := data.Kind.ValueString()

	tflog.Debug(ctx, "Reading zone metadata", map[string]interface{}{
		"zone_id": zoneID,
		"kind":    kind,
	})

	meta, err := d.client.GetZoneMetadata(ctx, int(zoneID), kind)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Zone Metadata",
			fmt.Sprintf("Could not read metadata kind %s of zone %d (a 404 means the zone does not exist or the kind has no values): %s", kind, zoneID, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%s", zoneID, meta.Kind))
	values := meta.Values
	if values == nil {
		values = []string{}
	}
	list, diags := types.ListValueFrom(ctx, types.StringType, values)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Values = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
