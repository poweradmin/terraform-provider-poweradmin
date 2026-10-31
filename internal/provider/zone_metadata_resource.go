// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
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

var _ resource.Resource = &ZoneMetadataResource{}
var _ resource.ResourceWithImportState = &ZoneMetadataResource{}
var _ resource.ResourceWithValidateConfig = &ZoneMetadataResource{}

// zoneMetadataMaxKindLength is the size of the domainmetadata.kind column.
const zoneMetadataMaxKindLength = 32

func NewZoneMetadataResource() resource.Resource {
	return &ZoneMetadataResource{}
}

// ZoneMetadataResource defines the resource implementation.
type ZoneMetadataResource struct {
	client *Client
}

// ZoneMetadataResourceModel describes the resource data model.
type ZoneMetadataResourceModel struct {
	ID     types.String `tfsdk:"id"`
	ZoneID types.Int64  `tfsdk:"zone_id"`
	Kind   types.String `tfsdk:"kind"`
	Values types.Set    `tfsdk:"values"`
}

func (r *ZoneMetadataResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_metadata"
}

func (r *ZoneMetadataResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages one PowerDNS metadata kind of a zone, such as `ALLOW-AXFR-FROM` or `ALSO-NOTIFY`, with all of its values. Requires Poweradmin 4.3.0+.\n\n" +
			"Creating the resource replaces any values the kind already has. Poweradmin checks the values at apply: it refuses kinds that " +
			"PowerDNS maintains itself, more than one value for a single-value kind, values outside a kind's vocabulary, operator-only kinds " +
			"for non-administrators and, on the PowerDNS API backend, custom kinds without the `X-` prefix. A kind that only takes effect next " +
			"to another kind needs that kind first, so add `depends_on`. Do not manage `SOA-EDIT-API` here together with `poweradmin_zone.soa_edit_api`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier in the format `zone_id/kind`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the zone the metadata belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					zoneIDRequiresReplace(func() *Client { return r.client }),
				},
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "Metadata kind in upper case, e.g. `ALLOW-AXFR-FROM`, at most 32 characters. Poweradmin stores kinds in upper case.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"values": schema.SetAttribute{
				MarkdownDescription: "Values of the kind; at least one. Poweradmin trims values, so they must not start or end with whitespace.",
				Required:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *ZoneMetadataResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig checks kind and values at plan time; Poweradmin would store
// them in another form and leave a diff forever. Unknown values are left to the server.
func (r *ZoneMetadataResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ZoneMetadataResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Kind.IsNull() && !data.Kind.IsUnknown() {
		validateZoneMetadataKind(data.Kind.ValueString(), &resp.Diagnostics)
	}
	if data.Values.IsNull() || data.Values.IsUnknown() {
		return
	}
	var values []types.String
	resp.Diagnostics.Append(data.Values.ElementsAs(ctx, &values, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateZoneMetadataValues(values, &resp.Diagnostics)
}

// validateZoneMetadataKind errors on a kind Poweradmin would store differently or refuse.
func validateZoneMetadataKind(kind string, diags *diag.Diagnostics) {
	stored := strings.ToUpper(strings.TrimSpace(kind))
	switch {
	case kind == "" || len(kind) > zoneMetadataMaxKindLength:
		diags.AddAttributeError(
			path.Root("kind"),
			"Invalid Metadata Kind",
			fmt.Sprintf("kind must be 1 to %d characters long, got: %q", zoneMetadataMaxKindLength, kind),
		)
	case kind != stored:
		diags.AddAttributeError(
			path.Root("kind"),
			"Invalid Metadata Kind",
			fmt.Sprintf("kind must be upper case without surrounding whitespace, as Poweradmin stores it: use %q, got: %q", stored, kind),
		)
	}
}

// validateZoneMetadataValues errors on an empty set and on values Poweradmin
// would trim; unknown elements are not checked.
func validateZoneMetadataValues(values []types.String, diags *diag.Diagnostics) {
	if len(values) == 0 {
		diags.AddAttributeError(
			path.Root("values"),
			"Invalid Metadata Values",
			"values must hold at least one value; remove the resource to delete the kind.",
		)
		return
	}
	for _, v := range values {
		if v.IsNull() || v.IsUnknown() {
			continue
		}
		if s := v.ValueString(); s == "" || s != strings.TrimSpace(s) {
			diags.AddAttributeError(
				path.Root("values"),
				"Invalid Metadata Value",
				fmt.Sprintf("values must not be empty or start or end with whitespace: Poweradmin trims them, so the stored value would never match the configuration. Got: %q", s),
			)
		}
	}
}

// applyZoneMetadata copies the server's view of a kind into the model.
func applyZoneMetadata(ctx context.Context, data *ZoneMetadataResourceModel, zoneID int64, meta *ZoneMetadata) diag.Diagnostics {
	data.ID = types.StringValue(fmt.Sprintf("%d/%s", zoneID, meta.Kind))
	data.ZoneID = types.Int64Value(zoneID)
	data.Kind = types.StringValue(meta.Kind)
	values := meta.Values
	if values == nil {
		values = []string{}
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, values)
	data.Values = set
	return diags
}

// setZoneMetadata writes the planned values and copies what the server stored into data.
func (r *ZoneMetadataResource) setZoneMetadata(ctx context.Context, data *ZoneMetadataResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	var values []string
	diags.Append(data.Values.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return diags
	}

	zoneID := data.ZoneID.ValueInt64()
	kind := data.Kind.ValueString()

	tflog.Debug(ctx, "Setting zone metadata", map[string]interface{}{
		"zone_id": zoneID,
		"kind":    kind,
		"values":  values,
	})

	if err := r.client.SetZoneMetadata(ctx, int(zoneID), kind, values); err != nil {
		diags.AddError(
			"Error Setting Zone Metadata",
			fmt.Sprintf("Could not set metadata kind %s of zone %d: %s", kind, zoneID, err.Error()),
		)
		return diags
	}

	// The PUT returns no data, so read back the values as stored
	meta, err := r.client.GetZoneMetadata(ctx, int(zoneID), kind)
	if err != nil {
		diags.AddError(
			"Error Reading Zone Metadata",
			fmt.Sprintf("Could not read metadata kind %s of zone %d after setting it: %s", kind, zoneID, err.Error()),
		)
		return diags
	}
	diags.Append(applyZoneMetadata(ctx, data, zoneID, meta)...)
	return diags
}

func (r *ZoneMetadataResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ZoneMetadataResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.setZoneMetadata(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Created zone metadata", map[string]interface{}{"id": data.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneMetadataResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ZoneMetadataResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()
	kind := data.Kind.ValueString()

	tflog.Debug(ctx, "Reading zone metadata", map[string]interface{}{
		"zone_id": zoneID,
		"kind":    kind,
	})

	meta, err := r.client.GetZoneMetadata(ctx, int(zoneID), kind)
	if err != nil {
		// 404: the zone is gone or the kind has no values left
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Zone Metadata",
			fmt.Sprintf("Could not read metadata kind %s of zone %d: %s", kind, zoneID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(applyZoneMetadata(ctx, &data, zoneID, meta)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneMetadataResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Only values, or a new zone_id for the same zone (see zoneIDRequiresReplace), reach Update
	var data ZoneMetadataResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.setZoneMetadata(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneMetadataResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ZoneMetadataResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()
	kind := data.Kind.ValueString()

	tflog.Debug(ctx, "Deleting zone metadata", map[string]interface{}{
		"zone_id": zoneID,
		"kind":    kind,
	})

	err := r.client.DeleteZoneMetadata(ctx, int(zoneID), kind)
	if err != nil {
		if IsNotFoundError(err) {
			tflog.Info(ctx, "Zone metadata already removed, ignoring error", map[string]interface{}{
				"zone_id": zoneID,
				"kind":    kind,
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Zone Metadata",
			fmt.Sprintf("Could not delete metadata kind %s of zone %d: %s", kind, zoneID, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted zone metadata")
}

func (r *ZoneMetadataResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	zoneID, kind, err := parseZoneMetadataImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Zone Metadata", err.Error())
		return
	}

	// Read replaces kind and id with the spelling the server stores
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), zoneID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("kind"), kind)...)
}
