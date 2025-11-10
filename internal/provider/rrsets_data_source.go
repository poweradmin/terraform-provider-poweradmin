// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RRSetsDataSource{}

func NewRRSetsDataSource() datasource.DataSource {
	return &RRSetsDataSource{}
}

// RRSetsDataSource defines the data source implementation.
type RRSetsDataSource struct {
	client *Client
}

// RRSetsDataSourceModel describes the data source data model.
type RRSetsDataSourceModel struct {
	ZoneID types.Int64      `tfsdk:"zone_id"`
	Type   types.String     `tfsdk:"type"`
	RRSets []RRSetDataModel `tfsdk:"rrsets"`
}

// RRSetDataModel describes an individual RRSet in the data source.
type RRSetDataModel struct {
	Name    types.String           `tfsdk:"name"`
	Type    types.String           `tfsdk:"type"`
	TTL     types.Int64            `tfsdk:"ttl"`
	Records []RRSetRecordDataModel `tfsdk:"records"`
}

// RRSetRecordDataModel describes a record in an RRSet.
type RRSetRecordDataModel struct {
	Content  types.String `tfsdk:"content"`
	Disabled types.Bool   `tfsdk:"disabled"`
	Priority types.Int64  `tfsdk:"priority"`
}

func (d *RRSetsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rrsets"
}

func (d *RRSetsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all Resource Record Sets (RRSets) from a zone. RRSets represent DNS-correct grouping of records with the same name and type.",

		Attributes: map[string]schema.Attribute{
			"zone_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the zone to retrieve RRSets from",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Optional filter by record type (e.g., 'A', 'AAAA', 'MX')",
				Optional:            true,
			},
			"rrsets": schema.ListNestedAttribute{
				MarkdownDescription: "List of RRSets in the zone",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The record name",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The record type",
							Computed:            true,
						},
						"ttl": schema.Int64Attribute{
							MarkdownDescription: "Time to live in seconds",
							Computed:            true,
						},
						"records": schema.ListNestedAttribute{
							MarkdownDescription: "List of records in this RRSet",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"content": schema.StringAttribute{
										MarkdownDescription: "Record content/value",
										Computed:            true,
									},
									"disabled": schema.BoolAttribute{
										MarkdownDescription: "Whether the record is disabled",
										Computed:            true,
									},
									"priority": schema.Int64Attribute{
										MarkdownDescription: "Priority for MX, SRV records",
										Computed:            true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *RRSetsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RRSetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RRSetsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check for unknown values - data sources cannot be read until all inputs are known
	if data.ZoneID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("zone_id"),
			"Unknown zone_id value",
			"The zone_id value is unknown at plan time. Data sources cannot be read until all configuration values are known.",
		)
		return
	}
	if data.Type.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("type"),
			"Unknown type value",
			"The type value is unknown at plan time. Data sources cannot be read until all configuration values are known.",
		)
		return
	}

	zoneID := data.ZoneID.ValueInt64()
	recordType := ""
	if !data.Type.IsNull() {
		recordType = data.Type.ValueString()
	}

	tflog.Debug(ctx, "Reading RRSets", map[string]interface{}{
		"zone_id": zoneID,
		"type":    recordType,
	})

	// Get RRSets from API
	rrsets, err := d.client.ListRRSets(ctx, zoneID, recordType)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading RRSets",
			fmt.Sprintf("Could not read RRSets for zone %d: %s", zoneID, err.Error()),
		)
		return
	}

	// Map response to model
	data.RRSets = make([]RRSetDataModel, len(rrsets))
	for i, rrset := range rrsets {
		records := make([]RRSetRecordDataModel, len(rrset.Records))
		for j, record := range rrset.Records {
			records[j] = RRSetRecordDataModel{
				Content:  types.StringValue(record.Content),
				Disabled: types.BoolValue(record.Disabled),
				Priority: types.Int64Value(record.Priority),
			}
		}

		data.RRSets[i] = RRSetDataModel{
			Name:    types.StringValue(rrset.Name),
			Type:    types.StringValue(rrset.Type),
			TTL:     types.Int64Value(rrset.TTL),
			Records: records,
		}
	}

	tflog.Trace(ctx, "Read RRSets", map[string]interface{}{
		"count": len(rrsets),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
