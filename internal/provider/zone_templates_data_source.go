// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ZoneTemplatesDataSource{}

func NewZoneTemplatesDataSource() datasource.DataSource {
	return &ZoneTemplatesDataSource{}
}

// ZoneTemplatesDataSource defines the data source implementation.
type ZoneTemplatesDataSource struct {
	client *Client
}

// ZoneTemplateSummaryModel describes a template entry returned by the list endpoint.
type ZoneTemplateSummaryModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Owner       types.Int64  `tfsdk:"owner"`
	IsGlobal    types.Bool   `tfsdk:"is_global"`
	ZonesLinked types.Int64  `tfsdk:"zones_linked"`
}

// ZoneTemplatesDataSourceModel describes the data source data model.
type ZoneTemplatesDataSourceModel struct {
	Templates []ZoneTemplateSummaryModel `tfsdk:"templates"`
}

func (d *ZoneTemplatesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_templates"
}

func (d *ZoneTemplatesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all Poweradmin zone templates visible to the authenticated caller. Requires Poweradmin 4.2.0+.",

		Attributes: map[string]schema.Attribute{
			"templates": schema.ListNestedAttribute{
				MarkdownDescription: "Zone templates visible to the caller",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "Zone template ID",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Zone template name",
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
					},
				},
			},
		},
	}
}

func (d *ZoneTemplatesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZoneTemplatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ZoneTemplatesDataSourceModel

	templates, err := d.client.ListZoneTemplates(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Zone Templates",
			fmt.Sprintf("Could not list zone templates: %s", err.Error()),
		)
		return
	}

	models := make([]ZoneTemplateSummaryModel, len(templates))
	for i, t := range templates {
		models[i] = ZoneTemplateSummaryModel{
			ID:          types.Int64Value(int64(t.ID)),
			Name:        types.StringValue(t.Name),
			Description: types.StringValue(t.Description),
			Owner:       types.Int64Value(int64(t.Owner)),
			IsGlobal:    types.BoolValue(t.IsGlobal),
			ZonesLinked: types.Int64Value(int64(t.ZonesLinked)),
		}
	}
	data.Templates = models

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
