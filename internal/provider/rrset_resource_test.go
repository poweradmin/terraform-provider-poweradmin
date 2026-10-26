// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestNormalizeRRSetRecords(t *testing.T) {
	configured := []RRSetRecordModel{
		{Content: types.StringValue("mail1.example.com."), Disabled: types.BoolValue(false), Priority: types.Int64Value(10)},
		{Content: types.StringValue("mail2.example.com"), Disabled: types.BoolValue(false), Priority: types.Int64Value(20)},
	}
	// API strips the trailing dot and returns records in a different order
	fromAPI := []RRSetRecord{
		{Content: "mail2.example.com", Disabled: false, Priority: 20},
		{Content: "mail1.example.com", Disabled: false, Priority: 10},
	}

	got := normalizeRRSetRecords(configured, fromAPI, "MX")

	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}
	if got[0].Content.ValueString() != "mail2.example.com" {
		t.Errorf("expected exact match preserved, got %q", got[0].Content.ValueString())
	}
	if got[1].Content.ValueString() != "mail1.example.com." {
		t.Errorf("expected configured trailing dot preserved, got %q", got[1].Content.ValueString())
	}
	if got[0].Priority.ValueInt64() != 20 || got[1].Priority.ValueInt64() != 10 {
		t.Errorf("expected API priorities kept, got %d and %d", got[0].Priority.ValueInt64(), got[1].Priority.ValueInt64())
	}
}

func TestNormalizeRRSetRecords_ContentCollision(t *testing.T) {
	// Same target with different priorities: dot spelling must stay with its own element
	configured := []RRSetRecordModel{
		{Content: types.StringValue("mail.example.com."), Disabled: types.BoolValue(false), Priority: types.Int64Value(10)},
		{Content: types.StringValue("mail.example.com"), Disabled: types.BoolValue(false), Priority: types.Int64Value(20)},
	}
	fromAPI := []RRSetRecord{
		{Content: "mail.example.com", Disabled: false, Priority: 20},
		{Content: "mail.example.com", Disabled: false, Priority: 10},
	}

	got := normalizeRRSetRecords(configured, fromAPI, "MX")

	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}
	if got[0].Content.ValueString() != "mail.example.com" || got[0].Priority.ValueInt64() != 20 {
		t.Errorf("expected undotted priority-20 record, got %q p%d", got[0].Content.ValueString(), got[0].Priority.ValueInt64())
	}
	if got[1].Content.ValueString() != "mail.example.com." || got[1].Priority.ValueInt64() != 10 {
		t.Errorf("expected dotted priority-10 record, got %q p%d", got[1].Content.ValueString(), got[1].Priority.ValueInt64())
	}
}

func TestNormalizeRRSetRecords_ExternalChangeSurfaces(t *testing.T) {
	configured := []RRSetRecordModel{
		{Content: types.StringValue("old.example.com.")},
	}
	fromAPI := []RRSetRecord{
		{Content: "new.example.com"},
	}

	got := normalizeRRSetRecords(configured, fromAPI, "MX")

	if got[0].Content.ValueString() != "new.example.com" {
		t.Errorf("expected external change to surface, got %q", got[0].Content.ValueString())
	}
}

func TestNormalizeRRSetRecords_HostnameTargets(t *testing.T) {
	tests := []struct {
		name       string
		recordType string
		configured string
		fromAPI    string
		want       string
	}{
		{"mx unicode target kept", "MX", "mail.bücher.example.", "mail.xn--bcher-kva.example", "mail.bücher.example."},
		{"cname case kept", "CNAME", "WWW.Example.com", "www.example.com", "WWW.Example.com"},
		{"srv unicode target kept", "SRV", "5 5060 sip.bücher.example", "5 5060 sip.xn--bcher-kva.example", "5 5060 sip.bücher.example"},
		{"txt stays exact", "TXT", "Hello", "hello", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configured := []RRSetRecordModel{{Content: types.StringValue(tt.configured), Disabled: types.BoolValue(false), Priority: types.Int64Value(10)}}
			got := normalizeRRSetRecords(configured, []RRSetRecord{{Content: tt.fromAPI, Priority: 10}}, tt.recordType)
			if got[0].Content.ValueString() != tt.want || got[0].Priority.ValueInt64() != 10 {
				t.Errorf("got %q p%d, want %q p10", got[0].Content.ValueString(), got[0].Priority.ValueInt64(), tt.want)
			}
		})
	}
}

func TestRRSetRead_SendsConfiguredName(t *testing.T) {
	ctx := context.Background()
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		respondJSON(t, w, RRSetResponse{RRSet: RRSet{Name: "xn--bcher-kva", Type: "MX", TTL: 3600, Records: []RRSetRecord{{Content: "mail.xn--bcher-kva.example", Priority: 10}}}})
	})
	r := &RRSetResource{client: client}
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	model := RRSetResourceModel{
		ID: types.StringValue("1/Bücher./MX"), ZoneID: types.Int64Value(1), Name: types.StringValue("Bücher."), Type: types.StringValue("MX"), TTL: types.Int64Value(3600),
		Records: []RRSetRecordModel{{Content: types.StringValue("Mail.Bücher.Example."), Disabled: types.BoolValue(false), Priority: types.Int64Value(10)}},
	}
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatalf("set state: %v", diags)
	}

	resp := &fwresource.ReadResponse{State: state}
	r.Read(ctx, fwresource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	// The server converts {name} itself; a client-side conversion could pick another label.
	if want := "/api/v2/zones/1/rrsets/B%C3%BCcher./MX"; gotPath != want {
		t.Errorf("expected configured name %q in path, got %q", want, gotPath)
	}
	var got RRSetResourceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.Name.ValueString() != "Bücher." || got.Records[0].Content.ValueString() != "Mail.Bücher.Example." {
		t.Errorf("expected configured spellings kept, got name %q content %q", got.Name.ValueString(), got.Records[0].Content.ValueString())
	}
}

func TestAccRRSetResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRRSetResourceConfig("test-rrset-acc.example.com", "www", "A", 3600, "192.0.2.1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_rrset.test", "name", "www"),
					resource.TestCheckResourceAttr("poweradmin_rrset.test", "type", "A"),
					resource.TestCheckResourceAttr("poweradmin_rrset.test", "ttl", "3600"),
					resource.TestCheckResourceAttrSet("poweradmin_rrset.test", "zone_id"),
				),
			},
			{
				ResourceName:      "poweradmin_rrset.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRRSetResourceConfig("test-rrset-acc.example.com", "www", "A", 7200, "192.0.2.1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_rrset.test", "ttl", "7200"),
				),
			},
		},
	})
}

func testAccRRSetResourceConfig(zoneName, name, recordType string, ttl int, content string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name = %[1]q
  type = "MASTER"
}

resource "poweradmin_rrset" "test" {
  zone_id = poweradmin_zone.test.id
  name    = %[2]q
  type    = %[3]q
  ttl     = %[4]d

  records = [
    { content = %[5]q },
  ]
}
`, zoneName, name, recordType, ttl, content)
}
