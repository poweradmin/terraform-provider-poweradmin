// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestRecordFilterFQDN(t *testing.T) {
	tests := []struct {
		name, filter, zone, want string
	}{
		{"relative", "www", "xn--bcher-kva.example", "www.xn--bcher-kva.example"},
		{"relative case", "WWW", "xn--bcher-kva.example", "www.xn--bcher-kva.example"},
		{"unicode relative", "münchen", "xn--bcher-kva.example", "xn--mnchen-3ya.xn--bcher-kva.example"},
		{"apex", "@", "xn--bcher-kva.example", "xn--bcher-kva.example"},
		{"apex by unicode zone name", "bücher.example.", "xn--bcher-kva.example", "xn--bcher-kva.example"},
		{"unicode fqdn with dot", "www.Bücher.Example.", "xn--bcher-kva.example", "www.xn--bcher-kva.example"},
		{"zone with trailing dot", "www", "example.com.", "www.example.com"},
		{"name repeating the zone stays qualified", "www.example.com.example.com", "example.com", "www.example.com.example.com"},
		{"other zone fqdn is relative", "www.example.org", "example.com", "www.example.org.example.com"},
		{"view variant relative", "www", "example.com..internal", "www.example.com"},
		{"view variant apex", "@", "example.com..internal", "example.com"},
		{"view variant exact fqdn with dot", "www.example.com.", "example.com..internal", "www.example.com"},
		{"view variant with trailing dot", "www", "Bücher.Example..internal.", "www.xn--bcher-kva.example"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := recordFilterFQDN(tt.filter, tt.zone); got != tt.want {
				t.Errorf("recordFilterFQDN(%q, %q) = %q, want %q", tt.filter, tt.zone, got, tt.want)
			}
		})
	}
}

func TestRecordsDataSourceRead_MatchesNameInAnySpelling(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/zones/1":
			respondJSON(t, w, ZoneResponse{Zone: Zone{ID: 1, Name: "xn--bcher-kva.example"}})
		case "/api/v2/zones/1/records":
			// The listing returns fully qualified names; API backend mode adds a trailing dot.
			respondJSON(t, w, RecordListResponse{Records: []Record{
				{ID: "10", ZoneID: 1, Name: "xn--mnchen-3ya.xn--bcher-kva.example", Type: "A", Content: "192.0.2.1", TTL: 3600},
				{ID: "11", ZoneID: 1, Name: "www.xn--bcher-kva.example.", Type: "A", Content: "192.0.2.2", TTL: 3600},
				{ID: "12", ZoneID: 1, Name: "xn--bcher-kva.example", Type: "A", Content: "192.0.2.3", TTL: 3600},
				{ID: "13", ZoneID: 1, Name: "www.xn--bcher-kva.example.xn--bcher-kva.example", Type: "A", Content: "192.0.2.4", TTL: 3600},
			}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	d := &RecordsDataSource{client: client}
	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)

	cases := []struct{ filter, wantID string }{
		{"münchen", "10"},
		{"München.Bücher.Example.", "10"},
		{"xn--mnchen-3ya", "10"},
		{"www", "11"},
		{"WWW.xn--bcher-kva.example", "11"},
		{"www.bücher.example.", "11"},
		{"@", "12"},
		{"Bücher.Example.", "12"},
		{"www.bücher.example.bücher.example", "13"},
	}
	for _, tc := range cases {
		filter := tc.filter
		config := tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
		state := tfsdk.State{Schema: schemaResp.Schema, Raw: config.Raw}
		if diags := state.Set(ctx, &RecordsDataSourceModel{ZoneID: types.Int64Value(1), Type: types.StringNull(), Name: types.StringValue(filter)}); diags.HasError() {
			t.Fatalf("set: %v", diags)
		}
		config.Raw = state.Raw
		resp := &datasource.ReadResponse{State: state}
		d.Read(ctx, datasource.ReadRequest{Config: config}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("read %q: %v", filter, resp.Diagnostics)
		}
		var got RecordsDataSourceModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
		if len(got.Records) != 1 || got.Records[0].ID.ValueString() != tc.wantID {
			t.Errorf("filter %q: expected record %s only, got %+v", filter, tc.wantID, got.Records)
		}
	}
}

func TestAccRecordsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsDataSourceConfig("test-records-ds.example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.poweradmin_records.test", "zone_id"),
					resource.TestCheckResourceAttrSet("data.poweradmin_records.test", "records.#"),
					resource.TestCheckResourceAttr("data.poweradmin_records.www", "records.#", "1"),
					resource.TestCheckResourceAttrPair("data.poweradmin_records.www", "records.0.id", "poweradmin_record.test", "id"),
					resource.TestCheckResourceAttr("data.poweradmin_records.www_fqdn", "records.#", "1"),
					resource.TestCheckResourceAttrPair("data.poweradmin_records.www_fqdn", "records.0.id", "poweradmin_record.test", "id"),
					resource.TestCheckResourceAttr("data.poweradmin_records.apex", "records.#", "1"),
					resource.TestCheckResourceAttr("data.poweradmin_records.apex", "records.0.type", "SOA"),
				),
			},
		},
	})
}

func testAccRecordsDataSourceConfig(zoneName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name = %[1]q
  type = "MASTER"
}

resource "poweradmin_record" "test" {
  zone_id = poweradmin_zone.test.id
  name    = "www"
  type    = "A"
  content = "192.0.2.1"
  ttl     = 3600
}

data "poweradmin_records" "test" {
  zone_id = poweradmin_zone.test.id

  depends_on = [poweradmin_record.test]
}

data "poweradmin_records" "www" {
  zone_id = poweradmin_zone.test.id
  name    = "www"

  depends_on = [poweradmin_record.test]
}

data "poweradmin_records" "www_fqdn" {
  zone_id = poweradmin_zone.test.id
  name    = "WWW.%[1]s."

  depends_on = [poweradmin_record.test]
}

data "poweradmin_records" "apex" {
  zone_id = poweradmin_zone.test.id
  name    = "@"
  type    = "SOA"

  depends_on = [poweradmin_record.test]
}
`, zoneName)
}
