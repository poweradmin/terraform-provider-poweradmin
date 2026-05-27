// Copyright Poweradmin Development Team 2025, 2026
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

var _ datasource.DataSource = &ZoneTemplateDataSource{}

func NewZoneTemplateDataSource() datasource.DataSource {
	return &ZoneTemplateDataSource{}
}

// ZoneTemplateDataSource defines the data source implementation.
type ZoneTemplateDataSource struct {
	client *Client
}

// ZoneTemplateRecordModel describes a single zone template record in data sources.
type ZoneTemplateRecordModel struct {
	ID       types.Int64  `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Content  types.String `tfsdk:"content"`
	TTL      types.Int64  `tfsdk:"ttl"`
	Priority types.Int64  `tfsdk:"priority"`
}

// ZoneTemplateDataSourceModel describes the data source data model.
type ZoneTemplateDataSourceModel struct {
	ID          types.Int64               `tfsdk:"id"`
	Name        types.String              `tfsdk:"name"`
	Description types.String              `tfsdk:"description"`
	Owner       types.Int64               `tfsdk:"owner"`
	IsGlobal    types.Bool                `tfsdk:"is_global"`
	ZonesLinked types.Int64               `tfsdk:"zones_linked"`
	Records     []ZoneTemplateRecordModel `tfsdk:"records"`
}

func (d *ZoneTemplateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_template"
}

func (d *ZoneTemplateDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a Poweradmin zone template, including its records. Look up by `id` or `name`. Requires Poweradmin 4.2.0+.",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Zone template ID. Either `id` or `name` must be specified.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Zone template name. Either `id` or `name` must be specified.",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the zone template",
				Computed:            true,
			},
			"owner": schema.Int64Attribute{
				MarkdownDescription: "User ID that owns the template. `0` for global templates.",
				Computed:            true,
			},
			"is_global": schema.BoolAttribute{
				MarkdownDescription: "Whether the template is global",
				Computed:            true,
			},
			"zones_linked": schema.Int64Attribute{
				MarkdownDescription: "Number of zones linked to this template",
				Computed:            true,
			},
			"records": schema.ListNestedAttribute{
				MarkdownDescription: "Records defined in the template",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "Record ID",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Record name (may contain `[ZONE]` placeholder)",
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
							MarkdownDescription: "Record TTL in seconds",
							Computed:            true,
						},
						"priority": schema.Int64Attribute{
							MarkdownDescription: "Record priority (used by MX, SRV)",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *ZoneTemplateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZoneTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ZoneTemplateDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Unknown id value",
			"The id value is unknown at plan time. Data sources cannot be read until all configuration values are known.",
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

	hasID := !data.ID.IsNull() && data.ID.ValueInt64() != 0
	hasName := !data.Name.IsNull() && data.Name.ValueString() != ""

	if !hasID && !hasName {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'id' or 'name' must be specified to look up a zone template",
		)
		return
	}

	var template *ZoneTemplate
	var err error

	if hasID {
		templateID := int(data.ID.ValueInt64())
		tflog.Debug(ctx, "Looking up zone template by ID", map[string]interface{}{
			"id": templateID,
		})
		template, err = d.client.GetZoneTemplate(ctx, templateID)
	} else {
		name := data.Name.ValueString()
		tflog.Debug(ctx, "Looking up zone template by name", map[string]interface{}{
			"name": name,
		})
		// FindZoneTemplateByName uses the list endpoint, which doesn't include
		// records — fetch the full template by ID afterwards.
		var summary *ZoneTemplate
		summary, err = d.client.FindZoneTemplateByName(ctx, name)
		if err == nil {
			template, err = d.client.GetZoneTemplate(ctx, summary.ID)
		}
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Zone Template",
			fmt.Sprintf("Could not read zone template: %s", err.Error()),
		)
		return
	}

	data.ID = types.Int64Value(int64(template.ID))
	data.Name = types.StringValue(template.Name)
	data.Description = types.StringValue(template.Description)
	data.Owner = types.Int64Value(int64(template.Owner))
	data.IsGlobal = types.BoolValue(template.IsGlobal)
	data.ZonesLinked = types.Int64Value(int64(template.ZonesLinked))

	records := make([]ZoneTemplateRecordModel, len(template.Records))
	for i, rec := range template.Records {
		records[i] = ZoneTemplateRecordModel{
			ID:       types.Int64Value(int64(rec.ID)),
			Name:     types.StringValue(rec.Name),
			Type:     types.StringValue(rec.Type),
			Content:  types.StringValue(rec.Content),
			TTL:      types.Int64Value(int64(rec.TTL)),
			Priority: types.Int64Value(int64(rec.Priority)),
		}
	}
	data.Records = records

	tflog.Trace(ctx, "Read zone_template data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
