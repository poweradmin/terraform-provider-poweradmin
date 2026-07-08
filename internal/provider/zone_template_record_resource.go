// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ZoneTemplateRecordResource{}
var _ resource.ResourceWithImportState = &ZoneTemplateRecordResource{}

func NewZoneTemplateRecordResource() resource.Resource {
	return &ZoneTemplateRecordResource{}
}

// ZoneTemplateRecordResource defines the resource implementation.
type ZoneTemplateRecordResource struct {
	client *Client
}

// ZoneTemplateRecordResourceModel describes the resource data model.
type ZoneTemplateRecordResourceModel struct {
	ID         types.String `tfsdk:"id"`
	TemplateID types.Int64  `tfsdk:"template_id"`
	RecordID   types.Int64  `tfsdk:"record_id"`
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	Content    types.String `tfsdk:"content"`
	TTL        types.Int64  `tfsdk:"ttl"`
	Priority   types.Int64  `tfsdk:"priority"`
}

func (r *ZoneTemplateRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_template_record"
}

func (r *ZoneTemplateRecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a record inside a Poweradmin zone template. Template records can use the placeholders `[ZONE]`, `[NS1]`, `[NS2]`, `[HOSTMASTER]`, and `[SERIAL]`, which are substituted when a zone is created from the template. Requires Poweradmin 4.2.0+.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier in the format `template_id/record_id`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"template_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the zone template that owns this record. Changing this forces recreation.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"record_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Numeric record ID assigned by Poweradmin",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Record name. Supports the `[ZONE]` placeholder.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "DNS record type (e.g. `A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, `SOA`).",
				Required:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "Record content. Supports the same placeholders as `name`.",
				Required:            true,
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "Record TTL in seconds. Defaults to `86400`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(86400),
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "Record priority (used by `MX`, `SRV`). Defaults to `0`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
		},
	}
}

func (r *ZoneTemplateRecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ZoneTemplateRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ZoneTemplateRecordResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := int(data.TemplateID.ValueInt64())

	createReq := CreateZoneTemplateRecordRequest{
		Name:     data.Name.ValueString(),
		Type:     data.Type.ValueString(),
		Content:  data.Content.ValueString(),
		TTL:      int(data.TTL.ValueInt64()),
		Priority: int(data.Priority.ValueInt64()),
	}

	tflog.Debug(ctx, "Creating zone template record", map[string]interface{}{
		"template_id": templateID,
		"name":        createReq.Name,
		"type":        createReq.Type,
	})

	recordID, err := r.client.CreateZoneTemplateRecord(ctx, templateID, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Zone Template Record",
			fmt.Sprintf("Could not create record in zone template %d: %s", templateID, err.Error()),
		)
		return
	}

	record, err := r.client.GetZoneTemplateRecord(ctx, templateID, recordID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Zone Template Record",
			fmt.Sprintf("Could not read created record %d in template %d: %s", recordID, templateID, err.Error()),
		)
		return
	}

	r.applyToModel(templateID, record, &data)

	tflog.Debug(ctx, "Zone template record created successfully", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneTemplateRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ZoneTemplateRecordResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := int(data.TemplateID.ValueInt64())
	recordID := int(data.RecordID.ValueInt64())

	tflog.Debug(ctx, "Reading zone template record", map[string]interface{}{
		"template_id": templateID,
		"record_id":   recordID,
	})

	record, err := r.client.GetZoneTemplateRecord(ctx, templateID, recordID)
	if err != nil {
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Zone Template Record",
			fmt.Sprintf("Could not read record %d in template %d: %s", recordID, templateID, err.Error()),
		)
		return
	}

	r.applyToModel(templateID, record, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneTemplateRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ZoneTemplateRecordResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := int(data.TemplateID.ValueInt64())
	recordID := int(data.RecordID.ValueInt64())

	ttl := int(data.TTL.ValueInt64())
	priority := int(data.Priority.ValueInt64())

	updateReq := UpdateZoneTemplateRecordRequest{
		Name:     data.Name.ValueString(),
		Type:     data.Type.ValueString(),
		Content:  data.Content.ValueString(),
		TTL:      &ttl,
		Priority: &priority,
	}

	tflog.Debug(ctx, "Updating zone template record", map[string]interface{}{
		"template_id": templateID,
		"record_id":   recordID,
	})

	if err := r.client.UpdateZoneTemplateRecord(ctx, templateID, recordID, updateReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Zone Template Record",
			fmt.Sprintf("Could not update record %d in template %d: %s", recordID, templateID, err.Error()),
		)
		return
	}

	record, err := r.client.GetZoneTemplateRecord(ctx, templateID, recordID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Zone Template Record",
			fmt.Sprintf("Could not read updated record %d in template %d: %s", recordID, templateID, err.Error()),
		)
		return
	}

	r.applyToModel(templateID, record, &data)

	tflog.Debug(ctx, "Zone template record updated successfully")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneTemplateRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ZoneTemplateRecordResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := int(data.TemplateID.ValueInt64())
	recordID := int(data.RecordID.ValueInt64())

	tflog.Debug(ctx, "Deleting zone template record", map[string]interface{}{
		"template_id": templateID,
		"record_id":   recordID,
	})

	if err := r.client.DeleteZoneTemplateRecord(ctx, templateID, recordID); err != nil {
		if IsNotFoundError(err) {
			tflog.Info(ctx, "Zone template record already deleted, ignoring error", map[string]interface{}{
				"template_id": templateID,
				"record_id":   recordID,
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Zone Template Record",
			fmt.Sprintf("Could not delete record %d in template %d: %s", recordID, templateID, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Zone template record deleted successfully")
}

func (r *ZoneTemplateRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Error Importing Zone Template Record",
			"Import ID must be in the format 'template_id/record_id'",
		)
		return
	}

	templateID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Zone Template Record",
			fmt.Sprintf("Could not parse template_id '%s': %s", parts[0], err.Error()),
		)
		return
	}

	recordID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Zone Template Record",
			fmt.Sprintf("Could not parse record_id '%s': %s", parts[1], err.Error()),
		)
		return
	}

	data := ZoneTemplateRecordResourceModel{
		ID:         types.StringValue(req.ID),
		TemplateID: types.Int64Value(templateID),
		RecordID:   types.Int64Value(recordID),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneTemplateRecordResource) applyToModel(templateID int, record *ZoneTemplateRecord, data *ZoneTemplateRecordResourceModel) {
	data.TemplateID = types.Int64Value(int64(templateID))
	data.RecordID = types.Int64Value(int64(record.ID))
	data.ID = types.StringValue(fmt.Sprintf("%d/%d", templateID, record.ID))
	data.Name = types.StringValue(record.Name)
	data.Type = types.StringValue(normalizeTypeCase(data.Type.ValueString(), record.Type))
	// Template records are stored verbatim except TXT auto-quoting (no dot stripping)
	data.Content = types.StringValue(normalizeTXTQuotes(data.Content.ValueString(), record.Content, record.Type))
	data.TTL = types.Int64Value(int64(record.TTL))
	data.Priority = types.Int64Value(int64(record.Priority))
}
