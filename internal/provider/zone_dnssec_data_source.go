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

var _ datasource.DataSource = &ZoneDnssecDataSource{}

func NewZoneDnssecDataSource() datasource.DataSource {
	return &ZoneDnssecDataSource{}
}

// ZoneDnssecDataSource reads the signing status of a zone.
type ZoneDnssecDataSource struct {
	client *Client
}

// ZoneDnssecDataSourceModel describes the data source data model.
type ZoneDnssecDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	ZoneID    types.Int64  `tfsdk:"zone_id"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	Presigned types.Bool   `tfsdk:"presigned"`
	DSRecords types.List   `tfsdk:"ds_records"`
	DNSKEY    types.String `tfsdk:"dnskey"`
}

func (d *ZoneDnssecDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_dnssec"
}

func (d *ZoneDnssecDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the DNSSEC signing status of a zone, with the DS records and DNSKEY needed for registry submission. " +
			"Requires Poweradmin 4.5.0+ with the PowerDNS API configured.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The zone ID",
			},
			"zone_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "ID of the zone",
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the zone is DNSSEC signed",
			},
			"presigned": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the zone is presigned (DNSSEC managed at the primary server)",
			},
			"ds_records": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "DS records of every key of the zone; empty when the zone is not signed",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key_tag":     schema.Int64Attribute{Computed: true, MarkdownDescription: "Key tag"},
						"algorithm":   schema.Int64Attribute{Computed: true, MarkdownDescription: "DNSSEC algorithm number"},
						"digest_type": schema.Int64Attribute{Computed: true, MarkdownDescription: "Digest type (2 = SHA-256, 4 = SHA-384)"},
						"digest":      schema.StringAttribute{Computed: true, MarkdownDescription: "Digest"},
					},
				},
			},
			"dnskey": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "DNSKEY record content of the first key that carries DS records (the KSK or CSK); null when the zone is not signed",
			},
		},
	}
}

func (d *ZoneDnssecDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZoneDnssecDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ZoneDnssecDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()

	tflog.Debug(ctx, "Reading zone DNSSEC status", map[string]interface{}{"zone_id": zoneID})

	status, err := d.client.GetZoneDnssec(ctx, int(zoneID))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Zone DNSSEC Status",
			fmt.Sprintf("Could not read DNSSEC status of zone %d: %s", zoneID, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(zoneID, 10))
	data.Enabled = types.BoolValue(status.Enabled)
	data.Presigned = types.BoolValue(status.Presigned)
	if status.DNSKEY != nil {
		data.DNSKEY = types.StringValue(*status.DNSKEY)
	} else {
		data.DNSKEY = types.StringNull()
	}
	list, diags := dnssecDSRecordsValue(status.DSRecords)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.DSRecords = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
