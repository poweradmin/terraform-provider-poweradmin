// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const testEncodedRecordID = "eyJ6IjoiZXhhbXBsZS5jb20iLCJuIjoid3d3LmV4YW1wbGUuY29tIiwidCI6IkEiLCJjIjoiMTkyLjAuMi4xIiwicCI6MH0"

func recordTestValue(t *testing.T, s tftypes.Type, id interface{}, content string, ttl, prio int64) tftypes.Value {
	t.Helper()
	return tftypes.NewValue(s, map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, id),
		"zone_id":    tftypes.NewValue(tftypes.Number, 1),
		"name":       tftypes.NewValue(tftypes.String, "www"),
		"type":       tftypes.NewValue(tftypes.String, "A"),
		"content":    tftypes.NewValue(tftypes.String, content),
		"ttl":        tftypes.NewValue(tftypes.Number, ttl),
		"priority":   tftypes.NewValue(tftypes.Number, prio),
		"disabled":   tftypes.NewValue(tftypes.Bool, false),
		"create_ptr": tftypes.NewValue(tftypes.Bool, false),
	})
}

func TestRecordIDPlanModifier(t *testing.T) {
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	NewRecordResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema
	objType := s.Type().TerraformType(ctx)

	cases := []struct {
		name        string
		stateID     string
		planContent string
		planTTL     int64
		planPrio    int64
		wantUnknown bool
	}{
		{"sql id, content change keeps id", "42", "192.0.2.2", 3600, 0, false},
		{"sql id, ttl change keeps id", "42", "192.0.2.1", 7200, 0, false},
		{"encoded id, ttl change keeps id", testEncodedRecordID, "192.0.2.1", 7200, 0, false},
		{"encoded id, content change unknown", testEncodedRecordID, "192.0.2.2", 3600, 0, true},
		{"encoded id, priority change unknown", testEncodedRecordID, "192.0.2.1", 3600, 10, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := tfsdk.State{Schema: s, Raw: recordTestValue(t, objType, tc.stateID, "192.0.2.1", 3600, 0)}
			plan := tfsdk.Plan{Schema: s, Raw: recordTestValue(t, objType, tftypes.UnknownValue, tc.planContent, tc.planTTL, tc.planPrio)}
			req := planmodifier.StringRequest{
				Path:       path.Root("id"),
				StateValue: types.StringValue(tc.stateID),
				PlanValue:  types.StringUnknown(),
				State:      state,
				Plan:       plan,
			}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			recordIDPlanModifier{}.PlanModifyString(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if got := resp.PlanValue.IsUnknown(); got != tc.wantUnknown {
				t.Fatalf("plan id unknown = %v, want %v (plan value %s)", got, tc.wantUnknown, resp.PlanValue)
			}
			if !tc.wantUnknown && resp.PlanValue.ValueString() != tc.stateID {
				t.Fatalf("plan id = %q, want %q", resp.PlanValue.ValueString(), tc.stateID)
			}
		})
	}
}

func TestRecordIDPlanModifier_CreateLeavesUnknown(t *testing.T) {
	resp := &planmodifier.StringResponse{PlanValue: types.StringUnknown()}
	recordIDPlanModifier{}.PlanModifyString(context.Background(), planmodifier.StringRequest{
		StateValue: types.StringNull(),
		PlanValue:  types.StringUnknown(),
	}, resp)
	if !resp.PlanValue.IsUnknown() {
		t.Fatalf("expected unknown id on create, got %s", resp.PlanValue)
	}
}

func TestIsEncodedRecordID(t *testing.T) {
	for id, want := range map[string]bool{"1": false, "123456": false, testEncodedRecordID: true, "a/b": true} {
		if got := isEncodedRecordID(id); got != want {
			t.Errorf("isEncodedRecordID(%q) = %v, want %v", id, got, want)
		}
	}
}

// An in-place content update on the API backend must address the record by its
// prior ID and store the new ID the API returns.
func TestRecordResourceUpdate_EncodedIDChanges(t *testing.T) {
	ctx := context.Background()
	const newID = "bmV3LWlk"
	var gotPath, gotMethod string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		respondJSON(t, w, RecordResponse{
			Record: Record{ID: newID, ZoneID: 1, Name: "www", Type: "A", Content: "192.0.2.2", TTL: 7200},
		})
	})
	res := &RecordResource{client: client}

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema
	objType := s.Type().TerraformType(ctx)

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: s, Raw: recordTestValue(t, objType, testEncodedRecordID, "192.0.2.1", 3600, 0)},
		Plan:  tfsdk.Plan{Schema: s, Raw: recordTestValue(t, objType, tftypes.UnknownValue, "192.0.2.2", 7200, 0)},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, nil)}}
	res.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if want := "/api/v2/zones/1/records/" + testEncodedRecordID; gotMethod != http.MethodPut || gotPath != want {
		t.Errorf("request = %s %s, want PUT %s", gotMethod, gotPath, want)
	}

	var id types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if id.ValueString() != newID {
		t.Errorf("state id = %q, want %q", id.ValueString(), newID)
	}
}
