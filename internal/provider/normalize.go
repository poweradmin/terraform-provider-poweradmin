// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/net/idna"
	"golang.org/x/text/unicode/norm"
)

// Helpers that keep state matching the plan without masking real drift:
// preserving configured spellings the API normalizes (case, zone suffixes,
// trailing dots, TXT quotes) and mapping empty API responses back to the
// configured null/empty form. Used by the zone, record, rrset, and template resources.

// normalizeTypeCase preserves the configured type spelling when the API
// returns the same type in different case (the server uppercases types).
func normalizeTypeCase(configured, fromAPI string) string {
	if strings.EqualFold(configured, fromAPI) {
		return configured
	}
	return fromAPI
}

// normalizeEmptyString maps an empty API response to the configured value's
// own empty form: "" stays "", null stays null, and a dropped non-empty
// configured value becomes null so the discrepancy surfaces.
func normalizeEmptyString(configured types.String, fromAPI string) types.String {
	if fromAPI != "" {
		return types.StringValue(fromAPI)
	}
	if !configured.IsNull() && !configured.IsUnknown() && configured.ValueString() == "" {
		return configured
	}
	return types.StringNull()
}

// normalizeTXTQuotes preserves the configured TXT content when the API returns
// it wrapped in the quotes that the server's txt_auto_quote setting adds.
func normalizeTXTQuotes(configured, fromAPI, recordType string) string {
	if strings.EqualFold(recordType, "TXT") && configured != "" && fromAPI == `"`+configured+`"` {
		return configured
	}
	return fromAPI
}

// normalizeRecordContent preserves the configured content value when the API
// strips trailing dots from FQDN content, or returns a hostname target in
// another spelling (case, Unicode vs punycode) for the hostname-target types.
func normalizeRecordContent(configured, fromAPI, recordType string) string {
	if recordContentEqual(configured, fromAPI, recordType) {
		return configured
	}
	return fromAPI
}

// recordContentEqual reports whether configured content is the same value the
// API returned. Hostname fields are compared canonically for the types and
// fields the server converts to punycode (DnsIdnService::convertContentToPunycode);
// other fields and other types (TXT, CAA, ...) must match up to a trailing dot.
func recordContentEqual(configured, fromAPI, recordType string) bool {
	if configured == "" {
		return false
	}
	if configured == fromAPI || strings.TrimSuffix(configured, ".") == fromAPI {
		return true
	}
	t := strings.ToUpper(recordType)
	switch t {
	case "CNAME", "NS", "PTR", "DNAME", "ALIAS", "MR":
		return dnsNamesEqual(configured, fromAPI)
	case "MX", "KX", "AFSDB", "SRV", "NAPTR", "RP", "MINFO", "HTTPS", "SVCB", "LP":
	default:
		return false
	}
	c, a := strings.Fields(configured), strings.Fields(fromAPI)
	if len(c) == 0 || len(c) != len(a) {
		return false
	}
	for i := range c {
		if isHostnameField(t, i, len(c)) {
			if !dnsNamesEqual(c[i], a[i]) {
				return false
			}
		} else if c[i] != a[i] {
			return false
		}
	}
	return true
}

// isHostnameField reports whether field i of n holds a hostname: the last
// field (MX, KX, AFSDB with or without a leading priority, SRV, NAPTR), the
// second (HTTPS, SVCB, LP target) or every field (RP, MINFO).
func isHostnameField(recordType string, i, n int) bool {
	switch recordType {
	case "RP", "MINFO":
		return true
	case "HTTPS", "SVCB", "LP":
		return i == 1
	}
	return i == n-1
}

// dnsNameProfile applies UTS 46 mapping (NFC, case folding, fullwidth and
// ligature mapping) like the server's idn_to_ascii, without STD3 or label
// validation so "_" and "*" labels pass through.
var dnsNameProfile = idna.New(idna.MapForLookup(), idna.Transitional(false), idna.StrictDomainName(false), idna.ValidateLabels(false))

// canonicalDNSName returns the comparison form of a DNS name: UTS 46 mapped,
// punycode, lowercase, no trailing dot. Unconvertible input is compared NFC
// lowercased as-is.
func canonicalDNSName(name string) string {
	name = strings.TrimSuffix(name, ".")
	if ascii, err := dnsNameProfile.ToASCII(name); err == nil {
		return strings.ToLower(ascii)
	}
	return strings.ToLower(norm.NFC.String(name))
}

// dnsNamesEqual reports whether two names are the same DNS name, ignoring
// case, a trailing dot and Unicode vs punycode spelling.
func dnsNamesEqual(a, b string) bool {
	return canonicalDNSName(a) == canonicalDNSName(b)
}

// normalizeZoneName preserves the configured zone name when the API returns
// the same name in its stored form (punycode, lowercase, no trailing dot).
func normalizeZoneName(configured, fromAPI string) string {
	if configured != "" && dnsNamesEqual(configured, fromAPI) {
		return configured
	}
	return fromAPI
}

// normalizeRecordName preserves the configured name when it is the FQDN form
// of the relative name the API returned (zone suffix stripped, "@" for apex),
// or the same name in another spelling (case, trailing dot, Unicode vs punycode).
func normalizeRecordName(configured, fromAPI, zoneName string) string {
	if configured == fromAPI {
		return configured
	}
	zoneName = strings.TrimSuffix(zoneName, ".")
	if zoneName == "" {
		// Zone name unknown (lookup failed): trust the configured name the
		// caller just sent; Read fails before reaching this fallback.
		if configured != "" {
			return configured
		}
		return fromAPI
	}
	if configured == "" {
		return fromAPI
	}
	if fromAPI == "@" {
		if configured == "@" || dnsNamesEqual(configured, zoneName) {
			return configured
		}
		return fromAPI
	}
	if dnsNamesEqual(configured, fromAPI) || dnsNamesEqual(configured, fromAPI+"."+zoneName) {
		return configured
	}
	return fromAPI
}
