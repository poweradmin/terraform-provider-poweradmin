// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &DnssecKeysDataSource{}

func NewDnssecKeysDataSource() datasource.DataSource {
	return &DnssecKeysDataSource{}
}

// DnssecKeysDataSource lists the DNSSEC keys of a zone.
type DnssecKeysDataSource struct {
	client *Client
}

// DnssecKeysDataSourceModel describes the data source data model.
type DnssecKeysDataSourceModel struct {
	ID     types.String         `tfsdk:"id"`
	ZoneID types.Int64          `tfsdk:"zone_id"`
	Keys   []DnssecKeyItemModel `tfsdk:"keys"`
}

// DnssecKeyItemModel is one key in the list.
type DnssecKeyItemModel struct {
	KeyID       types.Int64  `tfsdk:"key_id"`
	Type        types.String `tfsdk:"type"`
	KeyTag      types.Int64  `tfsdk:"keytag"`
	Algorithm   types.String `tfsdk:"algorithm"`
	AlgorithmID types.Int64  `tfsdk:"algorithm_id"`
	Bits        types.Int64  `tfsdk:"bits"`
	Active      types.Bool   `tfsdk:"active"`
	DNSKEY      types.String `tfsdk:"dnskey"`
	DS          types.List   `tfsdk:"ds"`
}

func (d *DnssecKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dnssec_keys"
}

func (d *DnssecKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the DNSSEC keys of a zone, including the default keys PowerDNS created when the zone was signed. " +
			"Requires Poweradmin 4.5.0+ with the PowerDNS API configured.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The zone ID",
			},
			"zone_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "ID of the zone",
			},
			"keys": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Keys of the zone; empty when the zone is not signed",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key_id":       schema.Int64Attribute{Computed: true, MarkdownDescription: "PowerDNS ID of the key"},
						"type":         schema.StringAttribute{Computed: true, MarkdownDescription: "Role PowerDNS currently assigns: `ksk` or `zsk` when the algorithm has both an active secure-entry-point key and an active other key, `csk` otherwise"},
						"keytag":       schema.Int64Attribute{Computed: true, MarkdownDescription: "Key tag"},
						"algorithm":    schema.StringAttribute{Computed: true, MarkdownDescription: "Algorithm name, e.g. `ecdsa256`; null for an algorithm Poweradmin does not name"},
						"algorithm_id": schema.Int64Attribute{Computed: true, MarkdownDescription: "DNSSEC algorithm number"},
						"bits":         schema.Int64Attribute{Computed: true, MarkdownDescription: "Key size in bits"},
						"active":       schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the key is active"},
						"dnskey":       schema.StringAttribute{Computed: true, MarkdownDescription: "DNSKEY record content"},
						"ds": schema.ListAttribute{
							Computed:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "DS records in zone-file form; present for keys PowerDNS currently counts as KSK or CSK",
						},
					},
				},
			},
		},
	}
}

func (d *DnssecKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DnssecKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DnssecKeysDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()

	tflog.Debug(ctx, "Listing DNSSEC keys", map[string]interface{}{"zone_id": zoneID})

	keys, err := d.client.ListDnssecKeys(ctx, int(zoneID))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing DNSSEC Keys",
			fmt.Sprintf("Could not list DNSSEC keys of zone %d: %s", zoneID, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(zoneID, 10))
	data.Keys = make([]DnssecKeyItemModel, 0, len(keys))
	for _, key := range keys {
		item := DnssecKeyItemModel{
			KeyID:       types.Int64Value(int64(key.ID)),
			Type:        types.StringValue(strings.ToLower(key.Type)),
			KeyTag:      types.Int64Value(int64(key.KeyTag)),
			Algorithm:   types.StringPointerValue(key.Algorithm),
			AlgorithmID: types.Int64Value(int64(key.AlgorithmID)),
			Bits:        types.Int64Value(int64(key.Bits)),
			Active:      types.BoolValue(key.Active),
			DNSKEY:      types.StringPointerValue(key.DNSKEY),
		}
		ds := key.DS
		if ds == nil {
			ds = []string{}
		}
		list, diags := types.ListValueFrom(ctx, types.StringType, ds)
		resp.Diagnostics.Append(diags...)
		item.DS = list
		data.Keys = append(data.Keys, item)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
