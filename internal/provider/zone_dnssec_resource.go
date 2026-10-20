// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ZoneDnssecResource{}
var _ resource.ResourceWithImportState = &ZoneDnssecResource{}

// dnssecDSRecordAttrTypes is the object type of a parsed DS record.
var dnssecDSRecordAttrTypes = map[string]attr.Type{
	"key_tag":     types.Int64Type,
	"algorithm":   types.Int64Type,
	"digest_type": types.Int64Type,
	"digest":      types.StringType,
}

func NewZoneDnssecResource() resource.Resource {
	return &ZoneDnssecResource{}
}

// ZoneDnssecResource signs a zone while it exists and unsigns it on destroy.
type ZoneDnssecResource struct {
	client *Client
}

// ZoneDnssecResourceModel describes the resource data model.
type ZoneDnssecResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ZoneID    types.Int64  `tfsdk:"zone_id"`
	Presigned types.Bool   `tfsdk:"presigned"`
	DSRecords types.List   `tfsdk:"ds_records"`
	DNSKEY    types.String `tfsdk:"dnskey"`
}

func (r *ZoneDnssecResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_dnssec"
}

func (r *ZoneDnssecResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Signs a zone with DNSSEC. Requires Poweradmin 4.5.0+ with the PowerDNS API configured and DNSSEC enabled on the server.\n\n" +
			"Creating the resource runs Poweradmin's signing steps: the zone must have active SOA and NS records, the SOA serial is bumped, " +
			"PowerDNS creates its default keys and the zone is rectified. **Destroying the resource unsigns the zone, which deletes every key " +
			"of the zone in PowerDNS**, including keys managed by `poweradmin_dnssec_key`. Presigned zones are refused.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The zone ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the zone to sign",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					zoneIDRequiresReplace(func() *Client { return r.client }),
				},
			},
			"presigned": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the zone is presigned (DNSSEC managed at the primary server)",
			},
			"ds_records": dnssecDSRecordsResourceAttribute(),
			"dnskey": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "DNSKEY record content of the first key that carries DS records (the KSK or CSK)",
			},
		},
	}
}

func dnssecDSRecordsResourceAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed:            true,
		MarkdownDescription: "DS records of every key of the zone, for submission to the parent zone's registrar",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"key_tag":     schema.Int64Attribute{Computed: true, MarkdownDescription: "Key tag"},
				"algorithm":   schema.Int64Attribute{Computed: true, MarkdownDescription: "DNSSEC algorithm number"},
				"digest_type": schema.Int64Attribute{Computed: true, MarkdownDescription: "Digest type (2 = SHA-256, 4 = SHA-384)"},
				"digest":      schema.StringAttribute{Computed: true, MarkdownDescription: "Digest"},
			},
		},
	}
}

// dnssecDSRecordsValue converts parsed DS records into a list of objects.
func dnssecDSRecordsValue(records []DnssecDSRecord) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	objType := types.ObjectType{AttrTypes: dnssecDSRecordAttrTypes}
	values := make([]attr.Value, 0, len(records))
	for _, ds := range records {
		obj, d := types.ObjectValue(dnssecDSRecordAttrTypes, map[string]attr.Value{
			"key_tag":     types.Int64Value(int64(ds.KeyTag)),
			"algorithm":   types.Int64Value(int64(ds.Algorithm)),
			"digest_type": types.Int64Value(int64(ds.DigestType)),
			"digest":      types.StringValue(ds.Digest),
		})
		diags.Append(d...)
		values = append(values, obj)
	}
	list, d := types.ListValue(objType, values)
	diags.Append(d...)
	return list, diags
}

func applyZoneDnssecStatus(data *ZoneDnssecResourceModel, zoneID int64, status *ZoneDnssecStatus) diag.Diagnostics {
	data.ID = types.StringValue(strconv.FormatInt(zoneID, 10))
	data.ZoneID = types.Int64Value(zoneID)
	data.Presigned = types.BoolValue(status.Presigned)
	if status.DNSKEY != nil {
		data.DNSKEY = types.StringValue(*status.DNSKEY)
	} else {
		data.DNSKEY = types.StringNull()
	}
	list, diags := dnssecDSRecordsValue(status.DSRecords)
	data.DSRecords = list
	return diags
}

func (r *ZoneDnssecResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZoneDnssecResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ZoneDnssecResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()

	tflog.Debug(ctx, "Signing zone", map[string]interface{}{"zone_id": zoneID})

	status, err := r.client.SetZoneDnssec(ctx, int(zoneID), true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Signing Zone",
			fmt.Sprintf("Could not sign zone %d: %s", zoneID, err.Error()),
		)
		return
	}
	if !status.Enabled {
		resp.Diagnostics.AddError(
			"Error Signing Zone",
			fmt.Sprintf("Poweradmin accepted the request but zone %d is not reported as signed", zoneID),
		)
		return
	}

	resp.Diagnostics.Append(applyZoneDnssecStatus(&data, zoneID, status)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Signed zone", map[string]interface{}{"zone_id": zoneID})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneDnssecResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ZoneDnssecResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()

	tflog.Debug(ctx, "Reading zone DNSSEC status", map[string]interface{}{"zone_id": zoneID})

	status, err := r.client.GetZoneDnssec(ctx, int(zoneID))
	if err != nil {
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Zone DNSSEC Status",
			fmt.Sprintf("Could not read DNSSEC status of zone %d: %s", zoneID, err.Error()),
		)
		return
	}

	// Unsigned outside Terraform: drop it so the next apply signs again
	if !status.Enabled {
		tflog.Info(ctx, "Zone is no longer signed, removing from state", map[string]interface{}{"zone_id": zoneID})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(applyZoneDnssecStatus(&data, zoneID, status)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneDnssecResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Only a new zone_id for the same zone reaches Update (see zoneIDRequiresReplace)
	var data ZoneDnssecResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()
	status, err := r.client.GetZoneDnssec(ctx, int(zoneID))
	if err == nil && !status.Enabled {
		// Unsigned since the plan: sign again, as Create does
		status, err = r.client.SetZoneDnssec(ctx, int(zoneID), true)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Zone DNSSEC",
			fmt.Sprintf("Could not read or sign DNSSEC of zone %d: %s", zoneID, err.Error()),
		)
		return
	}
	if !status.Enabled {
		resp.Diagnostics.AddError(
			"Error Updating Zone DNSSEC",
			fmt.Sprintf("Zone %d is still not signed after enabling DNSSEC.", zoneID),
		)
		return
	}

	resp.Diagnostics.Append(applyZoneDnssecStatus(&data, zoneID, status)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneDnssecResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ZoneDnssecResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()

	tflog.Debug(ctx, "Unsigning zone", map[string]interface{}{"zone_id": zoneID})

	status, err := r.client.SetZoneDnssec(ctx, int(zoneID), false)
	if err != nil {
		if IsNotFoundError(err) {
			tflog.Info(ctx, "Zone already removed, ignoring error", map[string]interface{}{"zone_id": zoneID})
			return
		}
		resp.Diagnostics.AddError(
			"Error Unsigning Zone",
			fmt.Sprintf("Could not unsign zone %d: %s", zoneID, err.Error()),
		)
		return
	}
	if status.Enabled {
		resp.Diagnostics.AddError(
			"Error Unsigning Zone",
			fmt.Sprintf("Poweradmin accepted the request but zone %d is still reported as signed", zoneID),
		)
		return
	}

	tflog.Debug(ctx, "Unsigned zone", map[string]interface{}{"zone_id": zoneID})
}

func (r *ZoneDnssecResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	zoneID, err := strconv.ParseUint(req.ID, 10, 63)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Zone DNSSEC",
			fmt.Sprintf("import ID must be a zone ID, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), int64(zoneID))...)
}
