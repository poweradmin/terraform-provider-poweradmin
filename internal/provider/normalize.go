// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
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
// strips trailing dots from FQDN content (CNAME, MX, NS, PTR, SRV records).
func normalizeRecordContent(configured, fromAPI string) string {
	if configured != "" && strings.TrimSuffix(configured, ".") == fromAPI {
		return configured
	}
	return fromAPI
}

// normalizeRecordName preserves the configured name when it is the FQDN form
// of the relative name the API returned (zone suffix stripped, "@" for apex),
// preventing "inconsistent result after apply" errors without masking real drift.
func normalizeRecordName(configured, fromAPI, zoneName string) string {
	if configured == fromAPI {
		return configured
	}
	relative := strings.TrimSuffix(configured, ".")
	zoneName = strings.TrimSuffix(zoneName, ".")
	if zoneName == "" {
		// Zone name unknown (lookup failed): trust the configured name the
		// caller just sent; Read fails before reaching this fallback.
		if configured != "" {
			return configured
		}
		return fromAPI
	}
	if fromAPI == "@" {
		if strings.EqualFold(relative, zoneName) {
			return configured
		}
		return fromAPI
	}
	if strings.EqualFold(relative, fromAPI+"."+zoneName) {
		return configured
	}
	return fromAPI
}
