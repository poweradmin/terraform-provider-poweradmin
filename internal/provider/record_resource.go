// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RecordResource{}
var _ resource.ResourceWithImportState = &RecordResource{}

func NewRecordResource() resource.Resource {
	return &RecordResource{}
}

// RecordResource defines the resource implementation.
type RecordResource struct {
	client *Client
}

// RecordResourceModel describes the resource data model.
type RecordResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ZoneID    types.Int64  `tfsdk:"zone_id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Content   types.String `tfsdk:"content"`
	TTL       types.Int64  `tfsdk:"ttl"`
	Priority  types.Int64  `tfsdk:"priority"`
	Disabled  types.Bool   `tfsdk:"disabled"`
	CreatePTR types.Bool   `tfsdk:"create_ptr"`
}

func (r *RecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_record"
}

func (r *RecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DNS record in a Poweradmin zone. Supports all standard DNS record types (A, AAAA, CNAME, MX, TXT, SRV, etc.).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for the record",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the zone this record belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The record name (e.g., 'www' for www.example.com, or '@' for the zone apex)",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The record type (A, AAAA, CNAME, MX, TXT, SRV, NS, PTR, etc.)",
				Required:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The record content/value",
				Required:            true,
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "Time to Live in seconds. Defaults to 3600.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3600),
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "Priority for MX and SRV records. Defaults to 0.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the record is disabled. Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"create_ptr": schema.BoolAttribute{
				MarkdownDescription: "Automatically create a PTR (reverse DNS) record for this record. Only applicable to A and AAAA records. Requires a matching reverse zone. Defaults to false. Changing this value requires resource replacement.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *RecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *RecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RecordResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build create request
	createReq := CreateRecordRequest{
		Name:      data.Name.ValueString(),
		Type:      data.Type.ValueString(),
		Content:   data.Content.ValueString(),
		TTL:       int(data.TTL.ValueInt64()),
		CreatePTR: data.CreatePTR.ValueBool(),
	}

	if !data.Priority.IsNull() {
		createReq.Priority = int(data.Priority.ValueInt64())
	}
	if !data.Disabled.IsNull() {
		createReq.Disabled = data.Disabled.ValueBool()
	}

	zoneID := data.ZoneID.ValueInt64()

	tflog.Debug(ctx, "Creating record", map[string]interface{}{
		"zone_id": zoneID,
		"name":    createReq.Name,
		"type":    createReq.Type,
	})

	// Create the record via API
	record, err := r.client.CreateRecord(ctx, zoneID, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Record",
			fmt.Sprintf("Could not create record %s in zone %d: %s", data.Name.ValueString(), zoneID, err.Error()),
		)
		return
	}

	// Map response back to model
	data.ID = types.StringValue(strconv.Itoa(record.ID))
	data.ZoneID = types.Int64Value(int64(record.ZoneID))
	data.Name = types.StringValue(record.Name)
	data.Type = types.StringValue(record.Type)
	data.Content = types.StringValue(record.Content)
	data.TTL = types.Int64Value(int64(record.TTL))
	data.Priority = types.Int64Value(int64(record.Priority))
	data.Disabled = types.BoolValue(record.Disabled)
	// API doesn't persist create_ptr - preserve from plan, or default to false if null
	if data.CreatePTR.IsNull() {
		data.CreatePTR = types.BoolValue(false)
	}

	tflog.Trace(ctx, "Created record", map[string]interface{}{
		"id": record.ID,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RecordResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse record ID
	recordID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Record ID",
			fmt.Sprintf("Could not parse record ID: %s", err.Error()),
		)
		return
	}

	zoneID := data.ZoneID.ValueInt64()

	tflog.Debug(ctx, "Reading record", map[string]interface{}{
		"zone_id":   zoneID,
		"record_id": recordID,
	})

	// Get the record from API
	record, err := r.client.GetRecord(ctx, zoneID, recordID)
	if err != nil {
		// If the record was deleted outside of Terraform, remove it from state
		if IsNotFoundError(err) {
			tflog.Info(ctx, "Record not found, removing from state", map[string]interface{}{
				"zone_id":   zoneID,
				"record_id": recordID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Record",
			fmt.Sprintf("Could not read record ID %d in zone %d: %s", recordID, zoneID, err.Error()),
		)
		return
	}

	// Update model with fresh data
	data.ID = types.StringValue(strconv.Itoa(record.ID))
	data.ZoneID = types.Int64Value(int64(record.ZoneID))
	data.Name = types.StringValue(record.Name)
	data.Type = types.StringValue(record.Type)
	data.Content = types.StringValue(record.Content)
	data.TTL = types.Int64Value(int64(record.TTL))
	data.Priority = types.Int64Value(int64(record.Priority))
	data.Disabled = types.BoolValue(record.Disabled)
	// API doesn't persist create_ptr - preserve from state, or default to false if null (for upgrades/imports)
	if data.CreatePTR.IsNull() {
		data.CreatePTR = types.BoolValue(false)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RecordResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse record ID
	recordID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Record ID",
			fmt.Sprintf("Could not parse record ID: %s", err.Error()),
		)
		return
	}

	zoneID := data.ZoneID.ValueInt64()

	// Build update request
	// Always send TTL and Priority (even if zero) to allow setting them to 0
	updateReq := UpdateRecordRequest{
		Name:    data.Name.ValueString(),
		Type:    data.Type.ValueString(),
		Content: data.Content.ValueString(),
	}

	// TTL - always send the value (even if 0) since it's computed with a default
	ttl := int(data.TTL.ValueInt64())
	updateReq.TTL = &ttl

	// Priority - always send the value (even if 0) since it's computed with a default
	priority := int(data.Priority.ValueInt64())
	updateReq.Priority = &priority

	// Disabled - always send the value since it's computed with a default
	if !data.Disabled.IsNull() {
		disabled := data.Disabled.ValueBool()
		updateReq.Disabled = &disabled
	}

	tflog.Debug(ctx, "Updating record", map[string]interface{}{
		"zone_id":   zoneID,
		"record_id": recordID,
	})

	// Update the record via API
	record, err := r.client.UpdateRecord(ctx, zoneID, recordID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Record",
			fmt.Sprintf("Could not update record ID %d in zone %d: %s", recordID, zoneID, err.Error()),
		)
		return
	}

	// Update model with response
	data.Name = types.StringValue(record.Name)
	data.Type = types.StringValue(record.Type)
	data.Content = types.StringValue(record.Content)
	data.TTL = types.Int64Value(int64(record.TTL))
	data.Priority = types.Int64Value(int64(record.Priority))
	data.Disabled = types.BoolValue(record.Disabled)
	// API doesn't persist create_ptr - preserve from plan, or default to false if null
	if data.CreatePTR.IsNull() {
		data.CreatePTR = types.BoolValue(false)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RecordResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse record ID
	recordID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Record ID",
			fmt.Sprintf("Could not parse record ID: %s", err.Error()),
		)
		return
	}

	zoneID := data.ZoneID.ValueInt64()

	tflog.Debug(ctx, "Deleting record", map[string]interface{}{
		"zone_id":   zoneID,
		"record_id": recordID,
	})

	// Delete the record via API
	err = r.client.DeleteRecord(ctx, zoneID, recordID)
	if err != nil {
		// If the record was already deleted outside of Terraform, that's fine
		if IsNotFoundError(err) {
			tflog.Info(ctx, "Record already deleted, ignoring error", map[string]interface{}{
				"zone_id":   zoneID,
				"record_id": recordID,
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Record",
			fmt.Sprintf("Could not delete record ID %d in zone %d: %s", recordID, zoneID, err.Error()),
		)
		return
	}

	tflog.Trace(ctx, "Deleted record", map[string]interface{}{
		"id": recordID,
	})
}

func (r *RecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "zone_id/record_id"
	// Example: terraform import poweradmin_record.www 123/456
	tflog.Debug(ctx, "Importing record", map[string]interface{}{
		"import_id": req.ID,
	})

	// Parse the import ID
	var zoneID, recordID int
	_, err := fmt.Sscanf(req.ID, "%d/%d", &zoneID, &recordID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'zone_id/record_id', got: %s", req.ID),
		)
		return
	}

	// Set both IDs in state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(recordID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), int64(zoneID))...)
}
