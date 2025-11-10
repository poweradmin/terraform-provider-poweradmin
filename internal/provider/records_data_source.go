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
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RecordsDataSource{}

func NewRecordsDataSource() datasource.DataSource {
	return &RecordsDataSource{}
}

// RecordsDataSource defines the data source implementation.
type RecordsDataSource struct {
	client *Client
}

// RecordsDataSourceModel describes the data source data model.
type RecordsDataSourceModel struct {
	ZoneID  types.Int64       `tfsdk:"zone_id"`
	Type    types.String      `tfsdk:"type"`
	Name    types.String      `tfsdk:"name"`
	Records []RecordDataModel `tfsdk:"records"`
}

// RecordDataModel describes a single record.
type RecordDataModel struct {
	ID       types.Int64  `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Content  types.String `tfsdk:"content"`
	TTL      types.Int64  `tfsdk:"ttl"`
	Priority types.Int64  `tfsdk:"priority"`
	Disabled types.Bool   `tfsdk:"disabled"`
}

func (d *RecordsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_records"
}

func (d *RecordsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This describes the data source.
		MarkdownDescription: "Fetches a list of DNS records from a zone. You can filter by record type and/or name.",

		Attributes: map[string]schema.Attribute{
			"zone_id": schema.Int64Attribute{
				MarkdownDescription: "Zone ID to query records from",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Filter by record type (e.g., A, AAAA, CNAME). Optional.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Filter by exact record name. Optional.",
				Optional:            true,
			},
			"records": schema.ListNestedAttribute{
				MarkdownDescription: "List of matching DNS records",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "Record ID",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Record name (FQDN)",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Record type",
							Computed:            true,
						},
						"content": schema.StringAttribute{
							MarkdownDescription: "Record content",
							Computed:            true,
						},
						"ttl": schema.Int64Attribute{
							MarkdownDescription: "Time to live",
							Computed:            true,
						},
						"priority": schema.Int64Attribute{
							MarkdownDescription: "Priority (for MX, SRV records)",
							Computed:            true,
						},
						"disabled": schema.BoolAttribute{
							MarkdownDescription: "Whether the record is disabled",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RecordsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RecordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RecordsDataSourceModel

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
	if data.Name.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("name"),
			"Unknown name value",
			"The name value is unknown at plan time. Data sources cannot be read until all configuration values are known.",
		)
		return
	}

	// Call API to list records
	recordType := ""
	if !data.Type.IsNull() {
		recordType = data.Type.ValueString()
	}

	records, err := d.client.ListRecords(ctx, data.ZoneID.ValueInt64(), recordType)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read records, got error: %s", err))
		return
	}

	// Filter by name if specified
	var filteredRecords []Record
	if !data.Name.IsNull() {
		filterName := data.Name.ValueString()
		for _, rec := range records {
			if rec.Name == filterName {
				filteredRecords = append(filteredRecords, rec)
			}
		}
	} else {
		filteredRecords = records
	}

	// Map response to model
	recordModels := make([]RecordDataModel, len(filteredRecords))
	for i, rec := range filteredRecords {
		recordModels[i] = RecordDataModel{
			ID:       types.Int64Value(int64(rec.ID)),
			Name:     types.StringValue(rec.Name),
			Type:     types.StringValue(rec.Type),
			Content:  types.StringValue(rec.Content),
			TTL:      types.Int64Value(int64(rec.TTL)),
			Priority: types.Int64Value(int64(rec.Priority)),
			Disabled: types.BoolValue(rec.Disabled),
		}
	}

	data.Records = recordModels

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
