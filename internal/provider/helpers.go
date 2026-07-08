// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// parseImportIDPair parses a "parent_id/child_id" import ID, rejecting
// malformed input, trailing garbage, and negative IDs.
func parseImportIDPair(id, format string) (int64, int64, error) {
	first, second, found := strings.Cut(id, "/")
	parent, errA := strconv.ParseUint(first, 10, 63)
	child, errB := strconv.ParseUint(second, 10, 63)
	if !found || errA != nil || errB != nil {
		return 0, 0, fmt.Errorf("import ID must be in format '%s', got: %s", format, id)
	}
	return int64(parent), int64(child), nil
}

// validateLookupChoice requires exactly one of id/name in by-id-or-name data
// sources; returns false when it added an error.
func validateLookupChoice(hasID, hasName bool, what string, diags *diag.Diagnostics) bool {
	switch {
	case !hasID && !hasName:
		diags.AddError(
			"Missing Required Attribute",
			fmt.Sprintf("Either 'id' or 'name' must be specified to look up a %s", what),
		)
		return false
	case hasID && hasName:
		diags.AddError(
			"Ambiguous Lookup",
			fmt.Sprintf("Only one of 'id' or 'name' may be specified to look up a %s, not both", what),
		)
		return false
	}
	return true
}
