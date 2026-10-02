// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &DnssecKeyResource{}
var _ resource.ResourceWithImportState = &DnssecKeyResource{}
var _ resource.ResourceWithValidateConfig = &DnssecKeyResource{}

// dnssecKeyTypes are the key types the API accepts.
var dnssecKeyTypes = []string{"ksk", "zsk", "csk"}

// dnssecAlgorithmBits maps each algorithm Poweradmin accepts to its allowed key sizes.
// PowerDNS may still refuse an algorithm its build lacks; the server reports that at apply.
var dnssecAlgorithmBits = map[string][]int64{
	"rsasha1":            {1024, 2048},
	"rsasha1-nsec3-sha1": {1024, 2048},
	"rsasha256":          {1024, 2048},
	"rsasha512":          {1024, 2048},
	"ecdsa256":           {256},
	"ecdsa384":           {384},
	"ed25519":            {256},
	"ed448":              {456},
}

func NewDnssecKeyResource() resource.Resource {
	return &DnssecKeyResource{}
}

// DnssecKeyResource defines the resource implementation.
type DnssecKeyResource struct {
	client *Client
}

// DnssecKeyResourceModel describes the resource data model.
type DnssecKeyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	KeyID       types.Int64  `tfsdk:"key_id"`
	ZoneID      types.Int64  `tfsdk:"zone_id"`
	Type        types.String `tfsdk:"type"`
	Algorithm   types.String `tfsdk:"algorithm"`
	Bits        types.Int64  `tfsdk:"bits"`
	Active      types.Bool   `tfsdk:"active"`
	KeyTag      types.Int64  `tfsdk:"keytag"`
	AlgorithmID types.Int64  `tfsdk:"algorithm_id"`
	DNSKEY      types.String `tfsdk:"dnskey"`
	DS          types.List   `tfsdk:"ds"`
}

func (r *DnssecKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dnssec_key"
}

func (r *DnssecKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DNSSEC key of a zone. Requires Poweradmin 4.5.0+ with the PowerDNS API configured and DNSSEC enabled on the server.\n\n" +
			"In PowerDNS a zone counts as signed as soon as it has an active key, so an active key signs the zone on its own. " +
			"Use `poweradmin_zone_dnssec` when the zone should go through Poweradmin's signing steps (pre-flight validation, SOA serial bump, rectify). " +
			"Unsigning a zone (destroying `poweradmin_zone_dnssec`) deletes every key of the zone, including keys managed by this resource.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier in the format `zone_id/key_id`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "PowerDNS ID of the key",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the zone the key belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Key type: `ksk`, `zsk` or `csk`. PowerDNS stores only whether the key is a secure entry point (`ksk` and `csk` both are) and reports a key as `csk` while no active key of the other kind shares its algorithm, so the configured value is kept while it matches the stored flag.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						dnssecKeyRoleChanged,
						"Replacing only when the key moves between zsk and ksk/csk, the only part PowerDNS stores.",
						"Replacing only when the key moves between `zsk` and `ksk`/`csk`, the only part PowerDNS stores.",
					),
				},
			},
			"algorithm": schema.StringAttribute{
				MarkdownDescription: "Algorithm: `rsasha1`, `rsasha1-nsec3-sha1`, `rsasha256`, `rsasha512`, `ecdsa256`, `ecdsa384`, `ed25519` or `ed448`. The PowerDNS build must support it. Keys with older algorithms (RSAMD5, DSA, GOST) cannot be imported.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bits": schema.Int64Attribute{
				MarkdownDescription: "Key size in bits. RSA algorithms take 1024 or 2048, `ecdsa256` and `ed25519` 256, `ecdsa384` 384 and `ed448` 456.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the key is active and signs the zone. Defaults to `true` (the API and web UI default to inactive).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"keytag": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Key tag",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"algorithm_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "DNSSEC algorithm number (RFC 8624), e.g. 13 for `ecdsa256`",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"dnskey": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "DNSKEY record content (flags, protocol, algorithm, public key)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ds": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "DS records in zone-file form (`keytag algorithm digest_type digest`) for submission to the parent zone's registrar. PowerDNS exports them for keys it currently counts as KSK or CSK, so a ZSK has none while an active KSK of its algorithm exists, and the list can change when other keys are activated or deactivated.",
			},
		},
	}
}

func (r *DnssecKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig checks type, algorithm and bits at plan time; values not yet
// known (computed from other resources) are left to the server.
func (r *DnssecKeyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data DnssecKeyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		validateDnssecKeyType(data.Type.ValueString(), &resp.Diagnostics)
	}
	if data.Algorithm.IsNull() || data.Algorithm.IsUnknown() {
		return
	}
	bits := int64(-1)
	if !data.Bits.IsNull() && !data.Bits.IsUnknown() {
		bits = data.Bits.ValueInt64()
	}
	validateDnssecAlgorithmBits(data.Algorithm.ValueString(), bits, &resp.Diagnostics)
}

// validateDnssecKeyType errors when keyType is not ksk, zsk or csk.
func validateDnssecKeyType(keyType string, diags *diag.Diagnostics) {
	if slices.Contains(dnssecKeyTypes, keyType) {
		return
	}
	diags.AddAttributeError(
		path.Root("type"),
		"Invalid DNSSEC Key Type",
		fmt.Sprintf("type must be one of %s, got: %q", strings.Join(dnssecKeyTypes, ", "), keyType),
	)
}

// validateDnssecAlgorithmBits errors on an unknown algorithm or a size it does
// not take; bits < 0 means the size is not known yet and is not checked.
func validateDnssecAlgorithmBits(algorithm string, bits int64, diags *diag.Diagnostics) {
	allowed, ok := dnssecAlgorithmBits[algorithm]
	if !ok {
		names := make([]string, 0, len(dnssecAlgorithmBits))
		for name := range dnssecAlgorithmBits {
			names = append(names, name)
		}
		slices.Sort(names)
		diags.AddAttributeError(
			path.Root("algorithm"),
			"Invalid DNSSEC Algorithm",
			fmt.Sprintf("algorithm must be one of %s, got: %q", strings.Join(names, ", "), algorithm),
		)
		return
	}
	if bits < 0 || slices.Contains(allowed, bits) {
		return
	}
	sizes := make([]string, len(allowed))
	for i, b := range allowed {
		sizes[i] = fmt.Sprint(b)
	}
	diags.AddAttributeError(
		path.Root("bits"),
		"Invalid DNSSEC Key Size",
		fmt.Sprintf("%s requires %s bits, got: %d", algorithm, strings.Join(sizes, " or "), bits),
	)
}

// dnssecKeyRoleChanged forces replacement only when a key moves between zsk and
// ksk/csk; ksk and csk differ in reported role only and are reconciled in place.
func dnssecKeyRoleChanged(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.PlanValue.IsUnknown() {
		resp.RequiresReplace = true
		return
	}
	resp.RequiresReplace = (req.StateValue.ValueString() == "zsk") != (req.PlanValue.ValueString() == "zsk")
}

// dnssecKeyType reconciles the configured type with the key PowerDNS reports.
// PowerDNS derives ksk/zsk/csk from the other active keys on every read; only the
// SEP flag is stored, so the configured type stands while it matches that flag.
func dnssecKeyType(current string, key *DnssecKey) string {
	reported := strings.ToLower(key.Type)
	sep, known := dnskeyHasSEP(key.DNSKEY)
	switch {
	case !known && current != "":
		return current
	case !known:
		return reported
	case current != "" && (current == "zsk") != sep:
		return current
	case slices.Contains(dnssecKeyTypes, reported) && (reported == "zsk") != sep:
		return reported
	case sep:
		return "csk"
	default:
		return "zsk"
	}
}

// dnskeyHasSEP reads the SEP bit from the flags field of DNSKEY content.
func dnskeyHasSEP(dnskey *string) (sep bool, known bool) {
	if dnskey == nil {
		return false, false
	}
	fields := strings.Fields(*dnskey)
	if len(fields) == 0 {
		return false, false
	}
	flags, err := strconv.Atoi(fields[0])
	if err != nil {
		return false, false
	}
	return flags&1 == 1, true
}

// applyDnssecKey copies the server's view of a key into the model. The
// algorithm name is kept from config when the server cannot name it.
func applyDnssecKey(ctx context.Context, data *DnssecKeyResourceModel, zoneID int64, key *DnssecKey) diag.Diagnostics {
	data.ID = types.StringValue(fmt.Sprintf("%d/%d", zoneID, key.ID))
	data.KeyID = types.Int64Value(int64(key.ID))
	data.ZoneID = types.Int64Value(zoneID)
	current := ""
	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		current = data.Type.ValueString()
	}
	data.Type = types.StringValue(dnssecKeyType(current, key))
	if key.Algorithm != nil {
		data.Algorithm = types.StringValue(*key.Algorithm)
	}
	data.Bits = types.Int64Value(int64(key.Bits))
	data.Active = types.BoolValue(key.Active)
	data.KeyTag = types.Int64Value(int64(key.KeyTag))
	data.AlgorithmID = types.Int64Value(int64(key.AlgorithmID))
	if key.DNSKEY != nil {
		data.DNSKEY = types.StringValue(*key.DNSKEY)
	} else {
		data.DNSKEY = types.StringNull()
	}
	ds := key.DS
	if ds == nil {
		ds = []string{}
	}
	list, diags := types.ListValueFrom(ctx, types.StringType, ds)
	data.DS = list
	return diags
}

func (r *DnssecKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DnssecKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()
	createReq := CreateDnssecKeyRequest{
		Type:      data.Type.ValueString(),
		Algorithm: data.Algorithm.ValueString(),
		Bits:      int(data.Bits.ValueInt64()),
		Active:    data.Active.ValueBool(),
	}

	tflog.Debug(ctx, "Creating DNSSEC key", map[string]interface{}{
		"zone_id":   zoneID,
		"type":      createReq.Type,
		"algorithm": createReq.Algorithm,
		"bits":      createReq.Bits,
		"active":    createReq.Active,
	})

	key, err := r.client.CreateDnssecKey(ctx, int(zoneID), createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating DNSSEC Key",
			fmt.Sprintf("Could not create DNSSEC key for zone %d: %s", zoneID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(applyDnssecKey(ctx, &data, zoneID, key)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Created DNSSEC key", map[string]interface{}{"id": data.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnssecKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DnssecKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()
	keyID := data.KeyID.ValueInt64()

	tflog.Debug(ctx, "Reading DNSSEC key", map[string]interface{}{
		"zone_id": zoneID,
		"key_id":  keyID,
	})

	key, err := r.client.GetDnssecKey(ctx, int(zoneID), int(keyID))
	if err != nil {
		// Only 404 means gone; a 502 (PowerDNS unreachable) must stay an error
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading DNSSEC Key",
			fmt.Sprintf("Could not read DNSSEC key %d of zone %d: %s", keyID, zoneID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(applyDnssecKey(ctx, &data, zoneID, key)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnssecKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DnssecKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only active changes on the server; a ksk/csk rename is reconciled by applyDnssecKey
	zoneID := data.ZoneID.ValueInt64()
	keyID := data.KeyID.ValueInt64()
	active := data.Active.ValueBool()

	tflog.Debug(ctx, "Updating DNSSEC key", map[string]interface{}{
		"zone_id": zoneID,
		"key_id":  keyID,
		"active":  active,
	})

	key, err := r.client.SetDnssecKeyActive(ctx, int(zoneID), int(keyID), active)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating DNSSEC Key",
			fmt.Sprintf("Could not set active=%t on DNSSEC key %d of zone %d: %s", active, keyID, zoneID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(applyDnssecKey(ctx, &data, zoneID, key)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnssecKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DnssecKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()
	keyID := data.KeyID.ValueInt64()

	tflog.Debug(ctx, "Deleting DNSSEC key", map[string]interface{}{
		"zone_id": zoneID,
		"key_id":  keyID,
	})

	err := r.client.DeleteDnssecKey(ctx, int(zoneID), int(keyID))
	if err != nil {
		if IsNotFoundError(err) {
			tflog.Info(ctx, "DNSSEC key already removed, ignoring error", map[string]interface{}{
				"zone_id": zoneID,
				"key_id":  keyID,
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting DNSSEC Key",
			fmt.Sprintf("Could not delete DNSSEC key %d of zone %d: %s", keyID, zoneID, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted DNSSEC key")
}

func (r *DnssecKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	zoneID, keyID, err := parseImportIDPair(req.ID, "zone_id/key_id")
	if err != nil {
		resp.Diagnostics.AddError("Error Importing DNSSEC Key", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), zoneID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key_id"), keyID)...)
}
