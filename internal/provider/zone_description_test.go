// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestZoneUpdateSendsDescriptionOnlyWhenConfigured checks the PUT body: a description
// left out of the config must not be blanked, while a configured "" still clears it.
func TestZoneUpdateSendsDescriptionOnlyWhenConfigured(t *testing.T) {
	ctx := context.Background()
	var schemaResp fwresource.SchemaResponse
	(&ZoneResource{}).Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("the schema is not an object")
	}
	null := tftypes.NewValue(tftypes.String, nil)
	object := func(description, masters tftypes.Value) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":           tftypes.NewValue(tftypes.String, "5"),
			"name":         tftypes.NewValue(tftypes.String, "example.com"),
			"type":         tftypes.NewValue(tftypes.String, "SLAVE"),
			"masters":      masters,
			"account":      null,
			"description":  description,
			"template":     null,
			"soa_edit_api": null,
		})
	}
	stored := tftypes.NewValue(tftypes.String, "set in the UI")
	newMasters := tftypes.NewValue(tftypes.String, "192.0.2.2")

	cases := []struct {
		name       string
		config     tftypes.Value
		plan       tftypes.Value
		serverDesc string
		wantSent   *string
	}{
		{"left out", null, stored, "set in the UI", nil},
		{"configured", tftypes.NewValue(tftypes.String, "new"), tftypes.NewValue(tftypes.String, "new"), "new", strPtr("new")},
		{"configured empty", tftypes.NewValue(tftypes.String, ""), tftypes.NewValue(tftypes.String, ""), "", strPtr("")},
	}
	for _, tc := range cases {
		var putBody map[string]interface{}
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Fatalf("%s: unexpected %s %s", tc.name, r.Method, r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&putBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			respondJSON(t, w, map[string]interface{}{"zone": Zone{ID: 5, Name: "example.com", Type: "SLAVE", Masters: "192.0.2.2", Description: tc.serverDesc}})
		})

		req := fwresource.UpdateRequest{
			Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: object(tc.config, newMasters)},
			Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: object(tc.plan, newMasters)},
			State:  tfsdk.State{Schema: schemaResp.Schema, Raw: object(stored, tftypes.NewValue(tftypes.String, "192.0.2.1"))},
		}
		resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: req.State.Raw}}
		(&ZoneResource{client: client}).Update(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s: update failed: %v", tc.name, resp.Diagnostics)
		}

		got, sent := putBody["description"]
		switch {
		case tc.wantSent == nil && sent:
			t.Errorf("%s: description sent as %v, want it left out (body %v)", tc.name, got, putBody)
		case tc.wantSent != nil && (!sent || got != *tc.wantSent):
			t.Errorf("%s: description = %v (sent %v), want %q", tc.name, got, sent, *tc.wantSent)
		}
		if tc.name == "left out" {
			var desc *string
			resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("description"), &desc)...)
			if desc == nil || *desc != "set in the UI" {
				t.Errorf("left out: state description = %v, want the stored value kept", desc)
			}
		}
	}
}

// TestZoneDescriptionPlansStoredValueWhenLeftOut guards the plan side of the same fix.
func TestZoneDescriptionPlansStoredValueWhenLeftOut(t *testing.T) {
	ctx := context.Background()
	var schemaResp fwresource.SchemaResponse
	(&ZoneResource{}).Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	attr, ok := schemaResp.Schema.Attributes["description"].(schema.StringAttribute)
	if !ok || !attr.Optional || !attr.Computed {
		t.Fatal("description must be Optional and Computed")
	}
	req := planmodifier.StringRequest{
		ConfigValue: types.StringNull(),
		PlanValue:   types.StringUnknown(),
		StateValue:  types.StringValue("set in the UI"),
		State:       tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})},
	}
	resp := planmodifier.StringResponse{PlanValue: req.PlanValue}
	for _, m := range attr.PlanModifiers {
		m.PlanModifyString(ctx, req, &resp)
	}
	if !resp.PlanValue.Equal(types.StringValue("set in the UI")) {
		t.Errorf("plan = %v, want the stored description", resp.PlanValue)
	}
}
