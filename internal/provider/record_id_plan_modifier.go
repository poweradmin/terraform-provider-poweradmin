// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// recordIDPlanModifier keeps the prior record ID in the plan, except when the
// ID is an encoded API backend ID and an attribute it encodes is changing.
type recordIDPlanModifier struct{}

var _ planmodifier.String = recordIDPlanModifier{}

func (m recordIDPlanModifier) Description(_ context.Context) string {
	return "Keeps the prior record ID unless an attribute encoded in a PowerDNS API backend record ID changes."
}

func (m recordIDPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m recordIDPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Create, destroy, or an ID the config already knows: nothing to keep
	if req.StateValue.IsNull() || req.Plan.Raw.IsNull() || !req.PlanValue.IsUnknown() {
		return
	}

	// SQL backend IDs are numeric row IDs that survive any in-place update
	if !isEncodedRecordID(req.StateValue.ValueString()) {
		resp.PlanValue = req.StateValue
		return
	}

	var plan, state RecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if recordIdentityUnchanged(plan, state) {
		resp.PlanValue = req.StateValue
	}
}

// isEncodedRecordID reports whether a record ID is a PowerDNS API backend ID
// rather than a numeric SQL row ID.
func isEncodedRecordID(id string) bool {
	_, err := strconv.ParseUint(id, 10, 64)
	return err != nil
}

// recordIdentityUnchanged compares the attributes the PowerDNS API backend
// encodes into a record ID (zone, name, type, content, priority).
func recordIdentityUnchanged(plan, state RecordResourceModel) bool {
	if plan.Name.IsUnknown() || plan.Type.IsUnknown() || plan.Content.IsUnknown() || plan.Priority.IsUnknown() {
		return false
	}
	return plan.Name.Equal(state.Name) &&
		plan.Type.Equal(state.Type) &&
		plan.Content.Equal(state.Content) &&
		plan.Priority.Equal(state.Priority)
}
