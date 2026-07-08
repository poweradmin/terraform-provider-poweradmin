// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZoneResource{}
var _ resource.ResourceWithImportState = &ZoneResource{}
var _ resource.ResourceWithValidateConfig = &ZoneResource{}

func NewZoneResource() resource.Resource {
	return &ZoneResource{}
}

// ZoneResource defines the resource implementation.
type ZoneResource struct {
	client *Client
}

// ZoneResourceModel describes the resource data model.
type ZoneResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	Masters     types.String `tfsdk:"masters"`
	Account     types.String `tfsdk:"account"`
	Description types.String `tfsdk:"description"`
	Template    types.String `tfsdk:"template"`
}

func (r *ZoneResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone"
}

func (r *ZoneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DNS zone in Poweradmin. Supports MASTER, SLAVE, and NATIVE zone types.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for the zone",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The zone name (e.g., example.com)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Zone type: MASTER, SLAVE, or NATIVE. Defaults to MASTER.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"masters": schema.StringAttribute{
				MarkdownDescription: "Master server(s) for SLAVE zones. Supports multiple formats:\n" +
					"  - Plain IP: `192.0.2.1`\n" +
					"  - Multiple IPs: `192.0.2.1,192.0.2.2`\n" +
					"  - IP with port: `192.0.2.1:5300`\n" +
					"  - Multiple with ports: `192.0.2.1:5300,192.0.2.2:5300`\n" +
					"  - IPv6 with port (requires brackets): `[2001:db8::1]:5300`\n\n" +
					"  Only valid for SLAVE zones; setting it on other zone types is an error.",
				Optional: true,
			},
			"account": schema.StringAttribute{
				MarkdownDescription: "Account name for the zone",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the zone",
				Optional:            true,
			},
			"template": schema.StringAttribute{
				MarkdownDescription: "Template to use when creating the zone (only applies during creation). Setting or changing it forces zone replacement; removing it from configuration does not.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
		},
	}
}

func (r *ZoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig rejects masters on an explicitly non-SLAVE zone. When type is
// omitted the actual type may still be SLAVE (kept from state), so the
// resolved-type guards in Create/Update cover that case instead.
func (r *ZoneResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ZoneResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.Masters.IsNull() || data.Masters.IsUnknown() || data.Masters.ValueString() == "" {
		return
	}
	if data.Type.IsNull() || data.Type.IsUnknown() || data.Type.ValueString() == "" {
		return
	}
	validateMastersForType(data.Masters.ValueString(), data.Type.ValueString(), &resp.Diagnostics)
}

// validateMastersForType errors when masters is set for a non-SLAVE zone;
// returns false when it added an error.
func validateMastersForType(masters, zoneType string, diags *diag.Diagnostics) bool {
	if masters == "" || strings.EqualFold(zoneType, "SLAVE") {
		return true
	}
	diags.AddAttributeError(
		path.Root("masters"),
		"Masters Requires SLAVE Zone",
		fmt.Sprintf("masters is only supported for SLAVE zones; this zone has type %s, and the server would silently ignore the value.", zoneType),
	)
	return false
}

func (r *ZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ZoneResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build create request
	createReq := CreateZoneRequest{
		Name: data.Name.ValueString(),
	}

	// Set zone type (default to MASTER if not specified)
	if !data.Type.IsNull() && data.Type.ValueString() != "" {
		createReq.Type = data.Type.ValueString()
	} else {
		createReq.Type = "MASTER"
	}

	// Set optional fields
	if !data.Masters.IsNull() {
		createReq.Masters = data.Masters.ValueString()
	}
	if !data.Account.IsNull() {
		createReq.Account = data.Account.ValueString()
	}
	if !data.Description.IsNull() {
		createReq.Description = data.Description.ValueString()
	}
	if !data.Template.IsNull() {
		createReq.Template = data.Template.ValueString()
	}

	// Guard on the resolved type (config may omit type, defaulting to MASTER)
	if !validateMastersForType(createReq.Masters, createReq.Type, &resp.Diagnostics) {
		return
	}

	tflog.Debug(ctx, "Creating zone", map[string]interface{}{
		"name": createReq.Name,
		"type": createReq.Type,
	})

	// Create the zone via API
	zoneID, err := r.client.CreateZone(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Zone",
			fmt.Sprintf("Could not create zone %s: %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	tflog.Trace(ctx, "Created zone", map[string]interface{}{
		"id": zoneID,
	})

	// Fetch the full zone data from the API
	// The create endpoint only returns the zone ID, so we need to read back the full zone
	zone, err := r.client.GetZone(ctx, zoneID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Created Zone",
			fmt.Sprintf("Zone was created with ID %d but could not read it back: %s", zoneID, err.Error()),
		)
		return
	}

	// Map response back to model
	data.ID = types.StringValue(strconv.Itoa(zone.ID))
	data.Name = types.StringValue(zone.Name)
	data.Type = types.StringValue(normalizeTypeCase(data.Type.ValueString(), zone.Type))

	// Mirror Read's mapping so a value the server dropped surfaces immediately
	// as an inconsistent-apply error instead of silent drift on the next plan
	data.Masters = normalizeEmptyString(data.Masters, zone.Masters)
	data.Account = normalizeEmptyString(data.Account, zone.Account)
	data.Description = normalizeEmptyString(data.Description, zone.Description)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ZoneResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse zone ID
	zoneID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Zone ID",
			fmt.Sprintf("Could not parse zone ID: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Reading zone", map[string]interface{}{
		"id": zoneID,
	})

	// Get the zone from API
	zone, err := r.client.GetZone(ctx, zoneID)
	if err != nil {
		// If the zone was deleted outside of Terraform, remove it from state
		if IsNotFoundError(err) {
			tflog.Info(ctx, "Zone not found, removing from state", map[string]interface{}{
				"id": zoneID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Zone",
			fmt.Sprintf("Could not read zone ID %d: %s", zoneID, err.Error()),
		)
		return
	}

	// Update model with fresh data
	data.ID = types.StringValue(strconv.Itoa(zone.ID))
	data.Name = types.StringValue(zone.Name)
	data.Type = types.StringValue(normalizeTypeCase(data.Type.ValueString(), zone.Type))

	data.Masters = normalizeEmptyString(data.Masters, zone.Masters)
	data.Account = normalizeEmptyString(data.Account, zone.Account)
	data.Description = normalizeEmptyString(data.Description, zone.Description)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ZoneResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse zone ID
	zoneID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Zone ID",
			fmt.Sprintf("Could not parse zone ID: %s", err.Error()),
		)
		return
	}

	// Build update request
	// Only send values that are known (not unknown) to avoid clearing fields unintentionally
	// For null values, send empty string to explicitly clear them
	updateReq := UpdateZoneRequest{}

	// Type is optional/computed, only send if known (not unknown from plan)
	if !data.Type.IsUnknown() {
		typeVal := data.Type.ValueString()
		updateReq.Type = &typeVal
	}

	// For optional fields, send empty string if null to clear them
	// Send the actual value if set
	// But skip if unknown (not changed in this update)
	if !data.Masters.IsUnknown() {
		mastersVal := ""
		if !data.Masters.IsNull() {
			mastersVal = data.Masters.ValueString()
		}
		updateReq.Masters = &mastersVal
	}

	if !data.Account.IsUnknown() {
		accountVal := ""
		if !data.Account.IsNull() {
			accountVal = data.Account.ValueString()
		}
		updateReq.Account = &accountVal
	}

	if !data.Description.IsUnknown() {
		descriptionVal := ""
		if !data.Description.IsNull() {
			descriptionVal = data.Description.ValueString()
		}
		updateReq.Description = &descriptionVal
	}

	// Guard on the resolved type before the API silently drops masters
	if updateReq.Masters != nil && updateReq.Type != nil {
		if !validateMastersForType(*updateReq.Masters, *updateReq.Type, &resp.Diagnostics) {
			return
		}
	}

	tflog.Debug(ctx, "Updating zone", map[string]interface{}{
		"id": zoneID,
	})

	// Update the zone via API
	zone, err := r.client.UpdateZone(ctx, zoneID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Zone",
			fmt.Sprintf("Could not update zone ID %d: %s", zoneID, err.Error()),
		)
		return
	}

	// Update model with response
	// Match the API response to what the user configured
	data.Type = types.StringValue(normalizeTypeCase(data.Type.ValueString(), zone.Type))

	data.Masters = normalizeEmptyString(data.Masters, zone.Masters)
	data.Account = normalizeEmptyString(data.Account, zone.Account)
	data.Description = normalizeEmptyString(data.Description, zone.Description)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ZoneResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse zone ID
	zoneID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Zone ID",
			fmt.Sprintf("Could not parse zone ID: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleting zone", map[string]interface{}{
		"id": zoneID,
	})

	// Delete the zone via API
	err = r.client.DeleteZone(ctx, zoneID)
	if err != nil {
		// If the zone was already deleted outside of Terraform, that's fine
		if IsNotFoundError(err) {
			tflog.Info(ctx, "Zone already deleted, ignoring error", map[string]interface{}{
				"id": zoneID,
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Zone",
			fmt.Sprintf("Could not delete zone ID %d: %s", zoneID, err.Error()),
		)
		return
	}

	tflog.Trace(ctx, "Deleted zone", map[string]interface{}{
		"id": zoneID,
	})
}

func (r *ZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support import by zone ID or zone name
	importID := req.ID

	tflog.Debug(ctx, "Importing zone", map[string]interface{}{
		"import_id": importID,
	})

	// Try to parse as integer (zone ID)
	_, err := strconv.Atoi(importID)
	if err == nil {
		// Import by ID
		resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
		return
	}

	// Import by name - need to look up the zone
	zone, err := r.client.FindZoneByName(ctx, importID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Zone",
			fmt.Sprintf("Could not find zone '%s': %s", importID, err.Error()),
		)
		return
	}

	// Set the ID in state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(zone.ID))...)
}
