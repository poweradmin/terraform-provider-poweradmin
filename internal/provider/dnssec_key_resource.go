// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
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
var _ resource.ResourceWithModifyPlan = &DnssecKeyResource{}

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

// dnssecAlgorithmIDs maps DNSSEC algorithm numbers to the names above.
var dnssecAlgorithmIDs = map[int]string{
	5: "rsasha1", 7: "rsasha1-nsec3-sha1", 8: "rsasha256", 10: "rsasha512",
	13: "ecdsa256", 14: "ecdsa384", 15: "ed25519", 16: "ed448",
}

// ISC/BIND private key lines, matched like Poweradmin's isIscPrivateKey:
// case-insensitive, at the start of a line, on the trimmed key of at most 16 KiB.
var (
	iscFormatLine    = regexp.MustCompile(`(?mi)^Private-key-format:\s*v1\.\d+\s*$`)
	iscAlgorithmLine = regexp.MustCompile(`(?mi)^Algorithm:\s*(\d+)`)
)

const maxPrivateKeyBytes = 16384

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

	PrivateKeyWO        types.String `tfsdk:"private_key_wo"`
	PrivateKeyWOVersion types.Int64  `tfsdk:"private_key_wo_version"`
}

func (r *DnssecKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dnssec_key"
}

func (r *DnssecKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DNSSEC key of a zone. Requires Poweradmin 4.5.0+ with the PowerDNS API configured and DNSSEC enabled on the server.\n\n" +
			"In PowerDNS a zone counts as signed as soon as it has an active key, so an active key signs the zone on its own. " +
			"Use `poweradmin_zone_dnssec` when the zone should go through Poweradmin's signing steps (pre-flight validation, SOA serial bump, rectify). " +
			"Unsigning a zone (destroying `poweradmin_zone_dnssec`) deletes every key of the zone, including keys managed by this resource.\n\n" +
			"Set `private_key_wo` to import an existing private key instead of generating one (Poweradmin 4.6.0+, Terraform 1.11+ or OpenTofu 1.11+). " +
			"Replacing a key deletes the old one before creating the new one; use `create_before_destroy` when it is the zone's only active key.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier in the format `zone_id/key_id`",
				PlanModifiers: []planmodifier.String{
					idFollowsZoneIDPlanModifier{},
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
					zoneIDRequiresReplace(func() *Client { return r.client }),
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
				MarkdownDescription: "Algorithm: `rsasha1`, `rsasha1-nsec3-sha1`, `rsasha256`, `rsasha512`, `ecdsa256`, `ecdsa384`, `ed25519` or `ed448`. The PowerDNS build must support it. Required unless `private_key_wo` is set; with a private key it is read from the key and, when set, must match it. Existing keys with older algorithms (RSAMD5, DSA, GOST) cannot be brought under management with `terraform import`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"bits": schema.Int64Attribute{
				MarkdownDescription: "Key size in bits. RSA algorithms take 1024 or 2048, `ecdsa256` and `ed25519` 256, `ecdsa384` 384 and `ed448` 456. Required unless `private_key_wo` is set; with a private key it is read from the key (an imported RSA key may have another size) and, when set, must match it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"private_key_wo": schema.StringAttribute{
				MarkdownDescription: "Existing private key to import, in the ISC/BIND format (`Private-key-format: v1.x`, as written by `dnssec-keygen` or `pdnsutil export-zone-key`); PEM keys are not accepted. " +
					"Write-only: it is sent once when the key is created and never stored in plan or state. Requires Poweradmin 4.6.0+ and Terraform 1.11+ or OpenTofu 1.11+. " +
					"Must be set together with `private_key_wo_version`; change the version to import a different key, which replaces this one.",
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
			},
			"private_key_wo_version": schema.Int64Attribute{
				MarkdownDescription: "Version of `private_key_wo`. Terraform cannot see changes to a write-only value, so change this number to import a new key; that replaces the key. Must be set together with `private_key_wo`.",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplaceIf(
						privateKeyVersionChanged,
						"Replacing when a new private key version is set.",
						"Replacing when a new private key version is set.",
					),
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
	bits := int64(-1)
	if !data.Bits.IsNull() && !data.Bits.IsUnknown() {
		bits = data.Bits.ValueInt64()
	}

	if !data.PrivateKeyWO.IsNull() || !data.PrivateKeyWOVersion.IsNull() {
		validateDnssecKeyImport(data, bits, &resp.Diagnostics)
		return
	}

	for _, attr := range []struct {
		name    string
		missing bool
	}{{"algorithm", data.Algorithm.IsNull()}, {"bits", data.Bits.IsNull()}} {
		if attr.missing {
			resp.Diagnostics.AddAttributeError(path.Root(attr.name), "Missing DNSSEC Key Attribute",
				fmt.Sprintf("%s is required unless private_key_wo is set", attr.name))
		}
	}
	if data.Algorithm.IsNull() || data.Algorithm.IsUnknown() {
		return
	}
	// Sizes are checked in ModifyPlan, which sees state: an imported key may keep a size
	// that generation does not offer once private_key_wo is removed from the config
	validateDnssecAlgorithmBits(data.Algorithm.ValueString(), -1, &resp.Diagnostics)
}

// ModifyPlan checks the key size against what generation accepts when a key is
// generated or its size would change; a stored size, e.g. an imported RSA-4096 key, stays valid.
func (r *DnssecKeyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var config DnssecKeyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || !config.PrivateKeyWO.IsNull() {
		return
	}
	if config.Algorithm.IsNull() || config.Algorithm.IsUnknown() || config.Bits.IsNull() || config.Bits.IsUnknown() {
		return
	}
	if !req.State.Raw.IsNull() {
		var state DnssecKeyResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if state.Algorithm.Equal(config.Algorithm) && state.Bits.Equal(config.Bits) {
			return
		}
	}
	validateDnssecAlgorithmBits(config.Algorithm.ValueString(), config.Bits.ValueInt64(), &resp.Diagnostics)
}

// validateDnssecKeyImport checks a private key import: the key and its version go
// together, and a configured algorithm or fixed key size must match the key.
func validateDnssecKeyImport(data DnssecKeyResourceModel, bits int64, diags *diag.Diagnostics) {
	if data.PrivateKeyWO.IsNull() {
		diags.AddAttributeError(path.Root("private_key_wo_version"), "Missing Private Key",
			"private_key_wo_version is set without private_key_wo")
		return
	}
	if data.PrivateKeyWOVersion.IsNull() {
		diags.AddAttributeError(path.Root("private_key_wo_version"), "Missing Private Key Version",
			"private_key_wo_version must be set together with private_key_wo, so that a new key can be imported by changing it")
	}
	if !data.Algorithm.IsNull() && !data.Algorithm.IsUnknown() {
		validateDnssecAlgorithmBits(data.Algorithm.ValueString(), -1, diags)
	}
	if data.PrivateKeyWO.IsUnknown() || diags.HasError() {
		return
	}

	keyAlgorithm, ok := iscKeyAlgorithm(data.PrivateKeyWO.ValueString())
	if !ok {
		diags.AddAttributeError(path.Root("private_key_wo"), "Invalid Private Key",
			"private_key_wo must be an ISC/BIND private key (Private-key-format: v1.x) with an Algorithm line of a supported algorithm; PEM keys are not accepted")
		return
	}
	if !data.Algorithm.IsNull() && !data.Algorithm.IsUnknown() && data.Algorithm.ValueString() != keyAlgorithm {
		diags.AddAttributeError(path.Root("algorithm"), "DNSSEC Algorithm Does Not Match Private Key",
			fmt.Sprintf("algorithm is %q but the private key uses %s", data.Algorithm.ValueString(), keyAlgorithm))
	}
	if allowed := dnssecAlgorithmBits[keyAlgorithm]; bits >= 0 && len(allowed) == 1 && allowed[0] != bits {
		diags.AddAttributeError(path.Root("bits"), "DNSSEC Key Size Does Not Match Private Key",
			fmt.Sprintf("bits is %d but %s keys are %d bits", bits, keyAlgorithm, allowed[0]))
	}
}

// iscKeyAlgorithm returns the algorithm name from an ISC/BIND private key, never echoing the key.
func iscKeyAlgorithm(privateKey string) (string, bool) {
	privateKey = strings.TrimSpace(privateKey)
	if privateKey == "" || len(privateKey) > maxPrivateKeyBytes || !iscFormatLine.MatchString(privateKey) {
		return "", false
	}
	m := iscAlgorithmLine.FindStringSubmatch(privateKey)
	if m == nil {
		return "", false
	}
	id, err := strconv.Atoi(m[1])
	if err != nil {
		return "", false
	}
	name, ok := dnssecAlgorithmIDs[id]
	return name, ok
}

// privateKeyVersionChanged replaces the key when a new private key version is
// set; removing the version alone keeps the key.
func privateKeyVersionChanged(ctx context.Context, req planmodifier.Int64Request, resp *int64planmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = !req.PlanValue.IsNull() && !req.PlanValue.Equal(req.StateValue)
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
// Only the SEP flag is stored (reported as sep on 4.6.0+, else read from the
// DNSKEY flags), so the configured type stands while it matches that flag.
func dnssecKeyType(current string, key *DnssecKey) string {
	reported := strings.ToLower(key.Type)
	sep, known := dnskeyHasSEP(key.DNSKEY)
	if key.SEP != nil {
		sep, known = *key.SEP, true
	}
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
	} else if data.Algorithm.IsUnknown() {
		data.Algorithm = types.StringNull()
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

	// Write-only values are only in the config, never in the plan
	var privateKey types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("private_key_wo"), &privateKey)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !privateKey.IsNull() {
		r.importKey(ctx, &data, privateKey.ValueString(), resp)
		return
	}

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

// importKey creates the key from a private key and checks it against a configured
// algorithm and size; a mismatching key is deleted again so it is not left behind.
func (r *DnssecKeyResource) importKey(ctx context.Context, data *DnssecKeyResourceModel, privateKey string, resp *resource.CreateResponse) {
	zoneID := data.ZoneID.ValueInt64()
	tflog.Debug(ctx, "Importing DNSSEC key", map[string]interface{}{
		"zone_id": zoneID,
		"type":    data.Type.ValueString(),
		"active":  data.Active.ValueBool(),
	})

	key, err := r.client.ImportDnssecKey(ctx, int(zoneID), ImportDnssecKeyRequest{
		Type:       data.Type.ValueString(),
		PrivateKey: privateKey,
		Active:     data.Active.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Importing DNSSEC Key",
			fmt.Sprintf("Could not import DNSSEC key for zone %d: %s", zoneID, err.Error()))
		return
	}

	if mismatch := dnssecKeyMismatch(*data, key); mismatch != "" {
		if err := r.client.DeleteDnssecKey(ctx, int(zoneID), key.ID); err != nil && !IsNotFoundError(err) {
			// Track the key in state so it is not orphaned; the next plan replaces it
			resp.Diagnostics.Append(applyDnssecKey(ctx, data, zoneID, key)...)
			resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
			resp.Diagnostics.AddError("Imported DNSSEC Key Does Not Match",
				fmt.Sprintf("%s. Deleting the imported key %d of zone %d failed too: %s", mismatch, key.ID, zoneID, err.Error()))
			return
		}
		resp.Diagnostics.AddError("Imported DNSSEC Key Does Not Match",
			fmt.Sprintf("%s; the imported key was deleted again.", mismatch))
		return
	}

	resp.Diagnostics.Append(applyDnssecKey(ctx, data, zoneID, key)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Imported DNSSEC key", map[string]interface{}{"id": data.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// dnssecKeyMismatch describes how an imported key differs from a configured
// algorithm or size, or returns "" when it matches.
func dnssecKeyMismatch(data DnssecKeyResourceModel, key *DnssecKey) string {
	if !data.Algorithm.IsNull() && !data.Algorithm.IsUnknown() {
		got := dnssecAlgorithmIDs[key.AlgorithmID]
		if key.Algorithm != nil {
			got = *key.Algorithm
		}
		if got != data.Algorithm.ValueString() {
			return fmt.Sprintf("algorithm is %q but PowerDNS reports the imported key as %q (algorithm %d)", data.Algorithm.ValueString(), got, key.AlgorithmID)
		}
	}
	if !data.Bits.IsNull() && !data.Bits.IsUnknown() && int64(key.Bits) != data.Bits.ValueInt64() {
		return fmt.Sprintf("bits is %d but PowerDNS reports the imported key as %d bits", data.Bits.ValueInt64(), key.Bits)
	}
	return ""
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
