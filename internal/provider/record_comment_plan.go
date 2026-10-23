// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// recordCommentPlanModifier keeps the prior comment in the plan, but not a null
// one: state written before the attribute existed has a null comment, and
// Update stores "" for it, which a known null in the plan would contradict.
type recordCommentPlanModifier struct{}

var _ planmodifier.String = recordCommentPlanModifier{}

func (m recordCommentPlanModifier) Description(_ context.Context) string {
	return "Keeps the prior comment in the plan unless the prior state has none recorded."
}

func (m recordCommentPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m recordCommentPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// An unknown config value (another resource's attribute) must stay unknown
	if req.ConfigValue.IsUnknown() {
		return
	}
	// Create, destroy, or a comment the config already sets: nothing to keep
	if req.StateValue.IsNull() || req.Plan.Raw.IsNull() || !req.PlanValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}

// noSurroundingWhitespace rejects a comment Poweradmin would trim, which would
// otherwise make the stored value differ from the configured one forever.
type noSurroundingWhitespace struct{}

var _ validator.String = noSurroundingWhitespace{}

func (v noSurroundingWhitespace) Description(_ context.Context) string {
	return "must not start or end with whitespace"
}

func (v noSurroundingWhitespace) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v noSurroundingWhitespace) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if val != strings.TrimSpace(val) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Comment",
			"The comment must not start or end with whitespace: Poweradmin trims it, so the stored value would never match the configuration.")
	}
}
