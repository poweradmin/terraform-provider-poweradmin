// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// zoneIDRequiresReplace decides what a changed zone_id means.
//
// Poweradmin 4.6.0 reports the canonical zone id. On installs migrated to the API backend
// that differs from the row id earlier releases reported, so a stored zone_id changes once
// while naming the same zone; that change is applied in place. A row id can also be another
// zone's canonical id, so when the old id opens a different zone than the new one the plan
// stops instead of replacing: a delete under the old id could hit that other zone. An old id
// that opens no zone at all still replaces, as a delete there reaches nothing.
func zoneIDRequiresReplace(client func() *Client) planmodifier.Int64 {
	return int64planmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.Int64Request, resp *int64planmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = true
			if req.StateValue.IsNull() || req.StateValue.IsUnknown() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
				return
			}
			c := client()
			if c == nil {
				return
			}
			oldID, newID := req.StateValue.ValueInt64(), req.PlanValue.ValueInt64()
			oldName, err := c.GetZoneName(ctx, oldID)
			if err != nil {
				if !IsNotFoundError(err) {
					resp.Diagnostics.AddAttributeError(req.Path, "Cannot Check zone_id Change",
						fmt.Sprintf("Could not look up zone %d to compare it with zone %d: %s", oldID, newID, err))
				}
				return
			}
			newName, err := c.GetZoneName(ctx, newID)
			if err != nil {
				resp.Diagnostics.AddAttributeError(req.Path, "Cannot Check zone_id Change",
					fmt.Sprintf("Could not look up zone %d to compare it with zone %d: %s", newID, oldID, err))
				return
			}
			if oldName == newName {
				resp.RequiresReplace = false
				return
			}
			resp.Diagnostics.AddAttributeError(req.Path, "Ambiguous zone_id Change",
				fmt.Sprintf("zone_id changes from %d (now zone %q) to %d (zone %q). Either the resource moved to another zone, "+
					"or %d is a zone id from before Poweradmin 4.6.0 that now opens a different zone. Replacing it could delete "+
					"objects in %q, so nothing is planned. Run `terraform state rm` on this resource, then import it or let "+
					"Terraform create it again.", oldID, oldName, newID, newName, oldID, oldName))
		},
		"Changing zone_id to another zone stops the plan; a new id for the same zone is applied in place.",
		"Changing `zone_id` to another zone stops the plan; a new id for the same zone is applied in place.",
	)
}
