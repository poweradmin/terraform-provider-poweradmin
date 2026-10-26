// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNormalizeRecordName(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		fromAPI    string
		zoneName   string
		want       string
	}{
		{"identical relative", "www", "www", "example.com", "www"},
		{"identical apex", "@", "@", "example.com", "@"},
		{"fqdn preserved", "www.example.com", "www", "example.com", "www.example.com"},
		{"fqdn with trailing dot preserved", "www.example.com.", "www", "example.com", "www.example.com."},
		{"fqdn case-insensitive match", "WWW.Example.COM", "www", "example.com", "WWW.Example.COM"},
		{"multi-label fqdn preserved", "sub.www.example.com", "sub.www", "example.com", "sub.www.example.com"},
		{"zone fqdn preserved for apex", "example.com", "@", "example.com", "example.com"},
		{"zone fqdn with dot preserved for apex", "example.com.", "@", "example.com", "example.com."},
		{"external rename surfaces", "www", "mail", "example.com", "mail"},
		{"external rename to apex surfaces", "www", "@", "example.com", "@"},
		{"relative multi-label drift surfaces", "sub.www", "sub", "example.com", "sub"},
		{"foreign fqdn not preserved for apex", "other.org", "@", "example.com", "@"},
		{"similar prefix is not a match", "wwwx.example.com", "www", "example.com", "www"},
		{"unknown zone name trusts configured", "www.example.com", "www", "", "www.example.com"},
		{"unknown zone name keeps identical value", "www", "www", "", "www"},
		{"unknown zone name with empty configured takes api value", "", "www", "", "www"},
		{"relative case-insensitive match", "WWW", "www", "example.com", "WWW"},
		{"unicode relative vs punycode", "bücher", "xn--bcher-kva", "example.com", "bücher"},
		{"unicode fqdn vs punycode relative in punycode zone", "www.bücher.example", "www", "xn--bcher-kva.example", "www.bücher.example"},
		{"unicode fqdn with dot vs punycode relative", "Bücher.Example.COM.", "xn--bcher-kva", "example.com", "Bücher.Example.COM."},
		{"unicode zone fqdn preserved for apex", "bücher.example.", "@", "xn--bcher-kva.example", "bücher.example."},
		{"service label in fqdn preserved", "_dmarc.bücher.example", "_dmarc", "xn--bcher-kva.example", "_dmarc.bücher.example"},
		{"wildcard in fqdn preserved", "*.example.com", "*", "example.com", "*.example.com"},
		{"unicode rename surfaces", "bücher", "xn--mnchen-3ya", "example.com", "xn--mnchen-3ya"},
		{"empty configured takes api value", "", "www", "example.com", "www"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeRecordName(tt.configured, tt.fromAPI, tt.zoneName); got != tt.want {
				t.Errorf("normalizeRecordName(%q, %q, %q) = %q, want %q", tt.configured, tt.fromAPI, tt.zoneName, got, tt.want)
			}
		})
	}
}

func TestNormalizeTypeCase(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		fromAPI    string
		want       string
	}{
		{"lowercase preserved", "master", "MASTER", "master"},
		{"exact match", "MASTER", "MASTER", "MASTER"},
		{"mixed case preserved", "CnAmE", "CNAME", "CnAmE"},
		{"real change surfaces", "MASTER", "SLAVE", "SLAVE"},
		{"empty configured takes api value", "", "A", "A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTypeCase(tt.configured, tt.fromAPI); got != tt.want {
				t.Errorf("normalizeTypeCase(%q, %q) = %q, want %q", tt.configured, tt.fromAPI, got, tt.want)
			}
		})
	}
}

func TestNormalizeEmptyString(t *testing.T) {
	tests := []struct {
		name       string
		configured types.String
		fromAPI    string
		want       types.String
	}{
		{"api value wins", types.StringValue("old"), "new", types.StringValue("new")},
		{"configured empty kept for empty response", types.StringValue(""), "", types.StringValue("")},
		{"null stays null", types.StringNull(), "", types.StringNull()},
		{"dropped value surfaces as null", types.StringValue("gone"), "", types.StringNull()},
		{"unknown resolves to null", types.StringUnknown(), "", types.StringNull()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeEmptyString(tt.configured, tt.fromAPI); !got.Equal(tt.want) {
				t.Errorf("normalizeEmptyString(%v, %q) = %v, want %v", tt.configured, tt.fromAPI, got, tt.want)
			}
		})
	}
}

func TestNormalizeTXTQuotes(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		fromAPI    string
		recordType string
		want       string
	}{
		{"auto-quoted txt preserved", "v=spf1 -all", `"v=spf1 -all"`, "TXT", "v=spf1 -all"},
		{"already quoted txt unchanged", `"v=spf1 -all"`, `"v=spf1 -all"`, "TXT", `"v=spf1 -all"`},
		{"lowercase type matches", "v=spf1 -all", `"v=spf1 -all"`, "txt", "v=spf1 -all"},
		{"real change surfaces", "v=spf1 -all", `"v=spf1 ~all"`, "TXT", `"v=spf1 ~all"`},
		{"non-txt quote drift surfaces", "x", `"x"`, "CNAME", `"x"`},
		{"empty configured takes api value", "", `"x"`, "TXT", `"x"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTXTQuotes(tt.configured, tt.fromAPI, tt.recordType); got != tt.want {
				t.Errorf("normalizeTXTQuotes(%q, %q, %q) = %q, want %q", tt.configured, tt.fromAPI, tt.recordType, got, tt.want)
			}
		})
	}
}

func TestNormalizeRecordContent(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		fromAPI    string
		want       string
	}{
		{"trailing dot preserved", "mail.example.com.", "mail.example.com", "mail.example.com."},
		{"identical values", "mail.example.com", "mail.example.com", "mail.example.com"},
		{"api keeps dot", "mail.example.com.", "mail.example.com.", "mail.example.com."},
		{"external change surfaces", "mail.example.com.", "other.example.com", "other.example.com"},
		{"empty configured takes api value", "", "mail.example.com", "mail.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeRecordContent(tt.configured, tt.fromAPI); got != tt.want {
				t.Errorf("normalizeRecordContent(%q, %q) = %q, want %q", tt.configured, tt.fromAPI, got, tt.want)
			}
		})
	}
}

func TestCanonicalDNSName(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"example.com", "example.com"},
		{"Example.COM.", "example.com"},
		{"bücher.example", "xn--bcher-kva.example"},
		{"BÜCHER.example.", "xn--bcher-kva.example"},
		{"xn--bcher-kva.example", "xn--bcher-kva.example"},
		{"XN--BCHER-KVA.example", "xn--bcher-kva.example"},
		{"_dmarc.example.com", "_dmarc.example.com"},
		{"_sip._tcp.example.com.", "_sip._tcp.example.com"},
		{"*.example.com", "*.example.com"},
		{"@", "@"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := canonicalDNSName(tt.in); got != tt.want {
				t.Errorf("canonicalDNSName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeZoneName(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		fromAPI    string
		want       string
	}{
		{"identical", "example.com", "example.com", "example.com"},
		{"trailing dot preserved", "example.com.", "example.com", "example.com."},
		{"case preserved", "Example.COM", "example.com", "Example.COM"},
		{"unicode preserved over punycode", "bücher.example", "xn--bcher-kva.example", "bücher.example"},
		{"unicode with dot and case preserved", "Bücher.Example.", "xn--bcher-kva.example", "Bücher.Example."},
		{"punycode configured kept", "xn--bcher-kva.example", "xn--bcher-kva.example", "xn--bcher-kva.example"},
		{"import takes api value", "", "xn--bcher-kva.example", "xn--bcher-kva.example"},
		{"different zone surfaces", "example.com", "example.org", "example.org"},
		{"different unicode zone surfaces", "bücher.example", "xn--mnchen-3ya.example", "xn--mnchen-3ya.example"},
		{"subdomain is not a match", "www.example.com", "example.com", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeZoneName(tt.configured, tt.fromAPI); got != tt.want {
				t.Errorf("normalizeZoneName(%q, %q) = %q, want %q", tt.configured, tt.fromAPI, got, tt.want)
			}
		})
	}
}

func TestApplyRecordKeepsEquivalentName(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		fromAPI    string
		want       string
	}{
		{"unicode fqdn kept", "www.bücher.example.", "www", "www.bücher.example."},
		{"unicode relative kept", "Bücher", "xn--bcher-kva", "Bücher"},
		{"rename surfaces", "www", "mail", "mail"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := RecordResourceModel{Name: types.StringValue(tt.configured), Type: types.StringValue("A"), Content: types.StringValue("192.0.2.1")}
			m.applyRecord(&Record{ID: "1", ZoneID: 1, Name: tt.fromAPI, Type: "A", Content: "192.0.2.1", TTL: 3600}, "xn--bcher-kva.example")
			if got := m.Name.ValueString(); got != tt.want {
				t.Errorf("applyRecord name = %q, want %q", got, tt.want)
			}
		})
	}
}
