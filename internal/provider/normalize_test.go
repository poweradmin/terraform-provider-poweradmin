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
		recordType string
		want       string
	}{
		{"trailing dot preserved", "mail.example.com.", "mail.example.com", "MX", "mail.example.com."},
		{"identical values", "mail.example.com", "mail.example.com", "MX", "mail.example.com"},
		{"api keeps dot", "mail.example.com.", "mail.example.com.", "MX", "mail.example.com."},
		{"external change surfaces", "mail.example.com.", "other.example.com", "MX", "other.example.com"},
		{"empty configured takes api value", "", "mail.example.com", "MX", "mail.example.com"},
		{"cname unicode target kept", "www.bücher.example", "www.xn--bcher-kva.example", "CNAME", "www.bücher.example"},
		{"cname unicode target with dot kept", "www.bücher.example.", "www.xn--bcher-kva.example", "CNAME", "www.bücher.example."},
		{"cname case kept", "WWW.Example.COM.", "www.example.com", "cname", "WWW.Example.COM."},
		{"ns unicode target kept", "ns1.bücher.example.", "ns1.xn--bcher-kva.example.", "NS", "ns1.bücher.example."},
		{"ptr unicode target kept", "host.bücher.example", "host.xn--bcher-kva.example", "PTR", "host.bücher.example"},
		{"dname unicode target kept", "bücher.example", "xn--bcher-kva.example", "DNAME", "bücher.example"},
		{"alias unicode target kept", "bücher.example.", "xn--bcher-kva.example", "ALIAS", "bücher.example."},
		{"cname unicode change surfaces", "www.bücher.example", "www.xn--mnchen-3ya.example", "CNAME", "www.xn--mnchen-3ya.example"},
		{"mx unicode target kept", "mail.bücher.example.", "mail.xn--bcher-kva.example", "MX", "mail.bücher.example."},
		{"mx with priority in content kept", "10 Mail.Bücher.Example.", "10 mail.xn--bcher-kva.example", "MX", "10 Mail.Bücher.Example."},
		{"mx priority change surfaces", "10 mail.bücher.example", "20 mail.xn--bcher-kva.example", "MX", "20 mail.xn--bcher-kva.example"},
		{"srv unicode target kept", "5 5060 sip.bücher.example.", "5 5060 sip.xn--bcher-kva.example", "SRV", "5 5060 sip.bücher.example."},
		{"srv port change surfaces", "5 5060 sip.bücher.example", "5 5061 sip.xn--bcher-kva.example", "SRV", "5 5061 sip.xn--bcher-kva.example"},
		{"srv field count change surfaces", "5 5060 sip.example.com", "5060 sip.example.com", "SRV", "5060 sip.example.com"},
		{"kx unicode target kept", "10 kx.bücher.example", "10 kx.xn--bcher-kva.example", "KX", "10 kx.bücher.example"},
		{"afsdb unicode target kept", "1 afs.bücher.example", "1 afs.xn--bcher-kva.example", "AFSDB", "1 afs.bücher.example"},
		{"txt case is exact", "Hello World", "hello world", "TXT", "hello world"},
		{"txt unicode is exact", "bücher.example", "xn--bcher-kva.example", "TXT", "xn--bcher-kva.example"},
		{"caa unicode is exact", `0 issue "bücher.example"`, `0 issue "xn--bcher-kva.example"`, "CAA", `0 issue "xn--bcher-kva.example"`},
		{"a record trailing dot rule unchanged", "192.0.2.1", "192.0.2.1", "A", "192.0.2.1"},
		{"nfd unicode target kept", "www.bu\u0308cher.example", "www.xn--bcher-kva.example", "CNAME", "www.bu\u0308cher.example"},
		{"mr unicode target kept", "mail.bücher.example.", "mail.xn--bcher-kva.example", "MR", "mail.bücher.example."},
		{"naptr unicode replacement kept", `100 10 "S" "SIP+D2U" "" _sip._udp.bücher.example.`, `100 10 "S" "SIP+D2U" "" _sip._udp.xn--bcher-kva.example.`, "NAPTR", `100 10 "S" "SIP+D2U" "" _sip._udp.bücher.example.`},
		{"naptr flags change surfaces", `100 10 "S" "SIP+D2U" "" _sip._udp.bücher.example.`, `100 10 "U" "SIP+D2U" "" _sip._udp.xn--bcher-kva.example.`, "NAPTR", `100 10 "U" "SIP+D2U" "" _sip._udp.xn--bcher-kva.example.`},
		{"rp unicode fields kept", "admin.bücher.example. info.bücher.example.", "admin.xn--bcher-kva.example. info.xn--bcher-kva.example.", "RP", "admin.bücher.example. info.bücher.example."},
		{"minfo unicode fields kept", "Req.Bücher.Example err.bücher.example", "req.xn--bcher-kva.example err.xn--bcher-kva.example", "MINFO", "Req.Bücher.Example err.bücher.example"},
		{"https unicode target kept", "1 svc.bücher.example. alpn=h2", "1 svc.xn--bcher-kva.example. alpn=h2", "HTTPS", "1 svc.bücher.example. alpn=h2"},
		{"https params stay exact", "1 svc.bücher.example. alpn=H2", "1 svc.xn--bcher-kva.example. alpn=h2", "HTTPS", "1 svc.xn--bcher-kva.example. alpn=h2"},
		{"svcb unicode target kept", "2 svc.bücher.example. port=8443", "2 svc.xn--bcher-kva.example. port=8443", "SVCB", "2 svc.bücher.example. port=8443"},
		{"svcb priority change surfaces", "2 svc.bücher.example.", "3 svc.xn--bcher-kva.example.", "SVCB", "3 svc.xn--bcher-kva.example."},
		{"lp unicode target kept", "10 l64.bücher.example.", "10 l64.xn--bcher-kva.example.", "LP", "10 l64.bücher.example."},
		{"https unquoted ech kept", "1 . alpn=h2 ech=AEX+/w==", `1 . alpn=h2 ech="AEX+/w=="`, "HTTPS", "1 . alpn=h2 ech=AEX+/w=="},
		{"svcb unquoted ech with unicode target kept", "1 svc.bücher.example. ECH=AEX= port=443", `1 svc.xn--bcher-kva.example. ECH="AEX=" port=443`, "SVCB", "1 svc.bücher.example. ECH=AEX= port=443"},
		{"https quoted ech identical", `1 . ech="AEX="`, `1 . ech="AEX="`, "HTTPS", `1 . ech="AEX="`},
		{"https different ech surfaces", "1 . ech=AEX=", `1 . ech="BBB="`, "HTTPS", `1 . ech="BBB="`},
		{"https other params unchanged", "1 . alpn=h2", `1 . alpn="h2"`, "HTTPS", `1 . alpn="h2"`},
		{"https ech-like param unchanged", "1 . xech=AEX=", `1 . xech="AEX="`, "HTTPS", `1 . xech="AEX="`},
		{"txt ech text unchanged", "ech=AEX=", `ech="AEX="`, "TXT", `ech="AEX="`},
		{"https single field stays exact", "svc.bücher.example.", "svc.xn--bcher-kva.example.", "HTTPS", "svc.xn--bcher-kva.example."},
		{"cname dotted capital i target kept", "www.İstanbul.example.", "www.xn--istanbul-o0e.example", "CNAME", "www.İstanbul.example."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeRecordContent(tt.configured, tt.fromAPI, tt.recordType); got != tt.want {
				t.Errorf("normalizeRecordContent(%q, %q, %q) = %q, want %q", tt.configured, tt.fromAPI, tt.recordType, got, tt.want)
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
		{"bu\u0308cher.example", "xn--bcher-kva.example"},
		{"İstanbul.example", "xn--istanbul-o0e.example"},
		{"\ufb01sh.example", "fish.example"},
		{"\uff21\uff22\uff23.example", "abc.example"},
		{"_dmarc.bücher.example", "_dmarc.xn--bcher-kva.example"},
		{"*.bücher.example", "*.xn--bcher-kva.example"},
		{".", ""},
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

func TestApplyRecordKeepsEquivalentTarget(t *testing.T) {
	m := RecordResourceModel{Name: types.StringValue("@"), Type: types.StringValue("MX"), Content: types.StringValue("Mail.Bücher.Example."), Priority: types.Int64Value(10)}
	m.applyRecord(&Record{ID: "1", ZoneID: 1, Name: "@", Type: "MX", Content: "mail.xn--bcher-kva.example", TTL: 3600, Priority: 10}, "xn--bcher-kva.example")
	if got := m.Content.ValueString(); got != "Mail.Bücher.Example." {
		t.Errorf("applyRecord content = %q, want configured spelling", got)
	}
	if got := m.Priority.ValueInt64(); got != 10 {
		t.Errorf("applyRecord priority = %d, want 10", got)
	}
}
