// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestIDFollowsZoneIDPlanModifier(t *testing.T) {
	ctx := context.Background()
	for name, newResource := range map[string]func() resource.Resource{
		"zone_metadata":         NewZoneMetadataResource,
		"group_zone_assignment": NewGroupZoneAssignmentResource,
		"dnssec_key":            NewDnssecKeyResource,
		"rrset":                 NewRRSetResource,
		"zone_dnssec":           NewZoneDnssecResource,
	} {
		schemaResp := &resource.SchemaResponse{}
		newResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)
		s := schemaResp.Schema

		idAttr, ok := s.Attributes["id"].(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s: id is not a string attribute", name)
		}
		mods := idAttr.PlanModifiers
		if len(mods) != 1 {
			t.Fatalf("%s: expected one id plan modifier, got %d", name, len(mods))
		}
		if _, ok := mods[0].(idFollowsZoneIDPlanModifier); !ok {
			t.Fatalf("%s: id plan modifier is %T", name, mods[0])
		}

		for _, tc := range []struct {
			planZoneID  int64
			wantUnknown bool
		}{{5, false}, {7, true}} {
			state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
			plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
			if d := state.SetAttribute(ctx, path.Root("zone_id"), int64(5)); d.HasError() {
				t.Fatalf("%s: %v", name, d)
			}
			if d := plan.SetAttribute(ctx, path.Root("zone_id"), tc.planZoneID); d.HasError() {
				t.Fatalf("%s: %v", name, d)
			}
			req := planmodifier.StringRequest{
				Path:       path.Root("id"),
				StateValue: types.StringValue("5/x"),
				PlanValue:  types.StringUnknown(),
				State:      state,
				Plan:       plan,
			}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			mods[0].PlanModifyString(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("%s: unexpected diagnostics: %v", name, resp.Diagnostics)
			}
			if resp.PlanValue.IsUnknown() != tc.wantUnknown || (!tc.wantUnknown && resp.PlanValue.ValueString() != "5/x") {
				t.Errorf("%s: zone_id 5 -> %d planned id %s, want unknown=%v", name, tc.planZoneID, resp.PlanValue, tc.wantUnknown)
			}
		}
	}
}
