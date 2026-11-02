// Copyright Poweradmin Development Team 2025, 2026
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

var _ datasource.DataSource = &ZoneMetadataEntriesDataSource{}

func NewZoneMetadataEntriesDataSource() datasource.DataSource {
	return &ZoneMetadataEntriesDataSource{}
}

// ZoneMetadataEntriesDataSource lists every metadata kind of a zone.
type ZoneMetadataEntriesDataSource struct {
	client *Client
}

// ZoneMetadataEntriesDataSourceModel describes the data source data model.
type ZoneMetadataEntriesDataSourceModel struct {
	ID      types.String             `tfsdk:"id"`
	ZoneID  types.Int64              `tfsdk:"zone_id"`
	Entries []ZoneMetadataEntryModel `tfsdk:"entries"`
}

// ZoneMetadataEntryModel is one kind in the list.
type ZoneMetadataEntryModel struct {
	Kind   types.String `tfsdk:"kind"`
	Values types.List   `tfsdk:"values"`
}

func (d *ZoneMetadataEntriesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_metadata_entries"
}

func (d *ZoneMetadataEntriesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every PowerDNS metadata kind of a zone with its values. Requires Poweradmin 4.3.0+.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The zone ID",
			},
			"zone_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "ID of the zone",
			},
			"entries": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Metadata kinds of the zone, sorted by kind; empty when the zone has no metadata",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"kind": schema.StringAttribute{Computed: true, MarkdownDescription: "Metadata kind, e.g. `ALLOW-AXFR-FROM`"},
						"values": schema.ListAttribute{
							Computed:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Values of the kind",
						},
					},
				},
			},
		},
	}
}

func (d *ZoneMetadataEntriesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZoneMetadataEntriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ZoneMetadataEntriesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()

	tflog.Debug(ctx, "Listing zone metadata", map[string]interface{}{"zone_id": zoneID})

	entries, err := d.client.ListZoneMetadata(ctx, int(zoneID))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Zone Metadata",
			fmt.Sprintf("Could not list metadata of zone %d: %s", zoneID, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(zoneID, 10))
	data.Entries = make([]ZoneMetadataEntryModel, 0, len(entries))
	for _, entry := range entries {
		values := entry.Values
		if values == nil {
			values = []string{}
		}
		list, diags := types.ListValueFrom(ctx, types.StringType, values)
		resp.Diagnostics.Append(diags...)
		data.Entries = append(data.Entries, ZoneMetadataEntryModel{
			Kind:   types.StringValue(entry.Kind),
			Values: list,
		})
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
