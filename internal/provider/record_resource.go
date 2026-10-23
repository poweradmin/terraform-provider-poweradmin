// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	Comment   types.String `tfsdk:"comment"`
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
				MarkdownDescription: "Unique identifier for the record. Numeric on SQL backends; on the PowerDNS API backend it encodes the record name, type, content and priority, so changing any of them in place changes the ID.",
				PlanModifiers: []planmodifier.String{
					recordIDPlanModifier{},
				},
			},
			"zone_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the zone this record belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					zoneIDRequiresReplace(func() *Client { return r.client }),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The record name. Accepts the relative form ('www', 'sub.www', '@' for the zone apex) or the FQDN form ('www.example.com'); the configured form is preserved in state.",
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
			"comment": schema.StringAttribute{
				MarkdownDescription: "Comment for the record, without leading or trailing whitespace. When unset, the provider never sends a comment, so an existing comment is left alone (removing the attribute later does not clear it); set it to an empty string to clear it. " +
					"On the PowerDNS API backend the comment belongs to the whole RRset (all records with the same name and type share it), so an empty string clears the comment for the whole RRset, and two records in one RRset configured with different comments produce a perpetual diff: give them the same value. " +
					"Poweradmin 4.6.0 or later is required to read the comment back; older versions ignore it, so the provider warns and keeps the configured value in state.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					noSurroundingWhitespace{},
				},
				PlanModifiers: []planmodifier.String{
					recordCommentPlanModifier{},
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
	if !data.Comment.IsNull() && !data.Comment.IsUnknown() {
		comment := data.Comment.ValueString()
		createReq.Comment = &comment
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

	// The record was written with the configured name, so keep it on lookup failure
	zoneName, zErr := r.zoneNameForNormalization(ctx, &data, record)
	if zErr != nil {
		resp.Diagnostics.AddWarning(
			"Record Name Not Normalized",
			fmt.Sprintf("Could not resolve zone name for zone ID %d: %s. Keeping configured name %q.", zoneID, zErr, data.Name.ValueString()),
		)
	}
	r.settleCreatedComment(ctx, &data, record, &resp.Diagnostics)
	data.applyRecord(record, zoneName)
	warnIfCommentNotReturned(createReq.Comment, record, &resp.Diagnostics)

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

	// Record IDs are opaque strings: numeric on SQL backends, encoded on the
	// PowerDNS API backend.
	recordID := RecordID(data.ID.ValueString())
	if recordID == "" {
		resp.Diagnostics.AddError("Invalid Record ID", "Record ID in state is empty")
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
			fmt.Sprintf("Could not read record ID %s in zone %d: %s", recordID, zoneID, err.Error()),
		)
		return
	}

	// Read must not guess: failing the refresh beats silently rewriting the name
	zoneName, zErr := r.zoneNameForNormalization(ctx, &data, record)
	if zErr != nil {
		resp.Diagnostics.AddError(
			"Error Reading Record",
			fmt.Sprintf("Could not resolve zone name for zone ID %d: %s", zoneID, zErr),
		)
		return
	}
	data.applyRecord(record, zoneName)

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

	// The plan ID is unknown when the update changes an encoded API backend ID,
	// so address the record by its prior ID from state.
	var priorID types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &priorID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	recordID := RecordID(priorID.ValueString())
	if recordID == "" {
		resp.Diagnostics.AddError("Invalid Record ID", "Record ID in state is empty")
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

	// Comment - only send when configured, so an unset comment never clears one set elsewhere
	var configComment types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("comment"), &configComment)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !configComment.IsNull() && !configComment.IsUnknown() {
		comment := configComment.ValueString()
		updateReq.Comment = &comment
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
			fmt.Sprintf("Could not update record ID %s in zone %d: %s", recordID, zoneID, err.Error()),
		)
		return
	}

	// The record was written with the configured name, so keep it on lookup failure
	zoneName, zErr := r.zoneNameForNormalization(ctx, &data, record)
	if zErr != nil {
		resp.Diagnostics.AddWarning(
			"Record Name Not Normalized",
			fmt.Sprintf("Could not resolve zone name for zone ID %d: %s. Keeping configured name %q.", zoneID, zErr, data.Name.ValueString()),
		)
	}
	data.applyRecord(record, zoneName)
	warnIfCommentNotReturned(updateReq.Comment, record, &resp.Diagnostics)

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

	// Record IDs are opaque strings: numeric on SQL backends, encoded on the
	// PowerDNS API backend.
	recordID := RecordID(data.ID.ValueString())
	if recordID == "" {
		resp.Diagnostics.AddError("Invalid Record ID", "Record ID in state is empty")
		return
	}

	zoneID := data.ZoneID.ValueInt64()

	tflog.Debug(ctx, "Deleting record", map[string]interface{}{
		"zone_id":   zoneID,
		"record_id": recordID,
	})

	// Delete the record via API
	err := r.client.DeleteRecord(ctx, zoneID, recordID)
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
			fmt.Sprintf("Could not delete record ID %s in zone %d: %s", recordID, zoneID, err.Error()),
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

	zoneID, recordID, err := parseRecordImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	// Set both IDs in state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), string(recordID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), zoneID)...)
}

// zoneNameForNormalization resolves the zone name only when the configured and
// API names differ (the only case normalization needs it); lookups are memoized.
func (r *RecordResource) zoneNameForNormalization(ctx context.Context, data *RecordResourceModel, record *Record) (string, error) {
	// Empty configured name (import) needs no preservation, so skip the lookup
	if name := data.Name.ValueString(); name == "" || name == record.Name {
		return "", nil
	}
	return r.client.GetZoneName(ctx, data.ZoneID.ValueInt64())
}

// applyRecord maps an API record onto the model, preserving the configured
// name/content forms the API normalizes away. create_ptr is not persisted by
// the API, so the plan/state value is kept (false after imports/upgrades).
func (m *RecordResourceModel) applyRecord(record *Record, zoneName string) {
	m.ID = types.StringValue(string(record.ID))
	m.ZoneID = types.Int64Value(int64(record.ZoneID))
	m.Name = types.StringValue(normalizeRecordName(m.Name.ValueString(), record.Name, zoneName))
	m.Type = types.StringValue(normalizeTypeCase(m.Type.ValueString(), record.Type))
	m.Content = types.StringValue(normalizeRecordContent(m.Content.ValueString(), record.Content))
	m.TTL = types.Int64Value(int64(record.TTL))
	m.Priority = types.Int64Value(int64(record.Priority))
	m.Disabled = types.BoolValue(record.Disabled)
	if m.CreatePTR.IsNull() {
		m.CreatePTR = types.BoolValue(false)
	}
	m.applyComment(record.Comment)
}

// applyComment maps the API comment. When the server omits the field (Poweradmin
// before 4.6.0) the prior value is kept; null (no comment) is stored as "".
func (m *RecordResourceModel) applyComment(c RecordComment) {
	if c.Present {
		if c.Value == nil {
			m.Comment = types.StringValue("")
		} else {
			m.Comment = types.StringValue(*c.Value)
		}
		return
	}
	if m.Comment.IsNull() || m.Comment.IsUnknown() {
		m.Comment = types.StringValue("")
	}
}

// settleCreatedComment clears a comment the server attached to a new record
// when "" was planned: a create cannot clear an inherited RRset or legacy
// name+type comment, so a follow-up update sends only the comment. It replaces
// record's comment with the update response; on failure it keeps the planned ""
// (so apply stays consistent) and warns.
func (r *RecordResource) settleCreatedComment(ctx context.Context, data *RecordResourceModel, record *Record, diags *diag.Diagnostics) {
	if data.Comment.IsNull() || data.Comment.IsUnknown() || data.Comment.ValueString() != "" {
		return
	}
	if !record.Comment.Present || record.Comment.Value == nil || *record.Comment.Value == "" {
		return
	}
	empty := ""
	updated, err := r.client.UpdateRecord(ctx, data.ZoneID.ValueInt64(), record.ID, UpdateRecordRequest{Comment: &empty})
	if err != nil {
		diags.AddWarning(
			"Record Comment Not Cleared",
			fmt.Sprintf("The record was created but its inherited comment could not be cleared: %s. The planned empty comment is stored; the next refresh shows the real value and the next plan clears it.", err),
		)
		record.Comment = RecordComment{Present: true, Value: &empty}
		return
	}
	if updated.Comment.Present {
		record.Comment = updated.Comment
	} else {
		record.Comment = RecordComment{Present: true, Value: &empty}
	}
}

// warnIfCommentNotReturned flags a sent comment that the server never echoes,
// which is how Poweradmin before 4.6.0 behaves (it ignores the field).
func warnIfCommentNotReturned(sent *string, record *Record, diags *diag.Diagnostics) {
	if sent == nil || record.Comment.Present {
		return
	}
	diags.AddWarning(
		"Record Comment Not Returned",
		"The comment was sent but the server did not return a comment field. Poweradmin before 4.6.0 ignores record comments, so the comment was probably not stored; the configured value is kept in state.",
	)
}
