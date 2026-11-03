// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// idFollowsZoneIDPlanModifier keeps the prior id unless zone_id changes; an id
// built from zone_id then changes too (see zoneIDRequiresReplace).
type idFollowsZoneIDPlanModifier struct{}

var _ planmodifier.String = idFollowsZoneIDPlanModifier{}

func (m idFollowsZoneIDPlanModifier) Description(_ context.Context) string {
	return "Keeps the prior id unless zone_id changes."
}

func (m idFollowsZoneIDPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m idFollowsZoneIDPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.Plan.Raw.IsNull() || !req.PlanValue.IsUnknown() {
		return
	}
	var planZoneID, stateZoneID types.Int64
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("zone_id"), &planZoneID)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("zone_id"), &stateZoneID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planZoneID.Equal(stateZoneID) {
		resp.PlanValue = req.StateValue
	}
}
