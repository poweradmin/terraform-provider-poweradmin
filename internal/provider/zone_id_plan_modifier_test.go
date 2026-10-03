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

// zoneNamesServer answers GET /zones/{id} with the given names and 404 for any other id.
func zoneNamesServer(t *testing.T, names map[string]string) *Client {
	t.Helper()
	return newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		for id, name := range names {
			if r.URL.Path == "/api/v2/zones/"+id {
				respondJSON(t, w, ZoneResponse{Zone: Zone{Name: name}})
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"success":false,"message":"Zone not found"}`))
	})
}

func planZoneIDChange(t *testing.T, client *Client, from, to types.Int64) (bool, bool) {
	t.Helper()
	present := tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{})
	req := planmodifier.Int64Request{
		Path:       path.Root("zone_id"),
		State:      tfsdk.State{Raw: present},
		Plan:       tfsdk.Plan{Raw: present},
		StateValue: from,
		PlanValue:  to,
	}
	resp := &planmodifier.Int64Response{PlanValue: to}
	zoneIDRequiresReplace(func() *Client { return client }).PlanModifyInt64(context.Background(), req, resp)
	return resp.RequiresReplace, resp.Diagnostics.HasError()
}

func TestZoneIDRequiresReplace_NewIDForTheSameZoneIsInPlace(t *testing.T) {
	// A migrated API-backend zone: Poweradmin 4.6.0 reports canonical id 4011 for row 7
	client := zoneNamesServer(t, map[string]string{"7": "example.com", "4011": "example.com"})

	if replace, failed := planZoneIDChange(t, client, types.Int64Value(7), types.Int64Value(4011)); replace || failed {
		t.Fatalf("expected an in-place change, got replace=%v error=%v", replace, failed)
	}
}

func TestZoneIDRequiresReplace_AnotherZoneStopsThePlan(t *testing.T) {
	// 14 may be a pre-4.6.0 row id that is now another zone's canonical id
	client := zoneNamesServer(t, map[string]string{"14": "group.example", "12": "reverse.example"})

	if _, failed := planZoneIDChange(t, client, types.Int64Value(14), types.Int64Value(12)); !failed {
		t.Fatal("expected the plan to stop when the old id opens another zone")
	}
}

func TestZoneIDRequiresReplace_AnOldIDThatOpensNothingIsReplaced(t *testing.T) {
	client := zoneNamesServer(t, map[string]string{"12": "reverse.example"})

	if replace, failed := planZoneIDChange(t, client, types.Int64Value(99), types.Int64Value(12)); !replace || failed {
		t.Fatalf("expected replacement, got replace=%v error=%v", replace, failed)
	}
}

func TestZoneIDRequiresReplace_DoubtNeverAppliesInPlace(t *testing.T) {
	client := zoneNamesServer(t, map[string]string{"7": "example.com"})

	if _, failed := planZoneIDChange(t, client, types.Int64Value(7), types.Int64Value(99)); !failed {
		t.Fatal("expected the plan to stop when the new id cannot be looked up")
	}
	if replace, _ := planZoneIDChange(t, client, types.Int64Value(7), types.Int64Unknown()); !replace {
		t.Fatal("expected replacement when the new id is unknown")
	}
	if replace, _ := planZoneIDChange(t, nil, types.Int64Value(7), types.Int64Value(4011)); !replace {
		t.Fatal("expected replacement without a configured client")
	}
}

func TestZoneID_PrefersTheCanonicalID(t *testing.T) {
	if got := (Zone{ID: 7, CanonicalID: 4011}).ZoneID(); got != 4011 {
		t.Fatalf("got %d, want 4011", got)
	}
	if got := (Zone{ID: 7}).ZoneID(); got != 7 {
		t.Fatalf("got %d, want 7 when canonical_id is absent", got)
	}
}

func TestGroupZoneAssignmentUpdate_MovesTheAssignmentToTheNewZoneID(t *testing.T) {
	var calls []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		respondJSON(t, w, nil)
	})
	ctx := context.Background()
	res := &GroupZoneAssignmentResource{client: client}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	state := tfsdk.State{Schema: schemaResp.Schema}
	state.Set(ctx, &GroupZoneAssignmentResourceModel{ID: types.StringValue("3/7"), GroupID: types.Int64Value(3), ZoneID: types.Int64Value(7)})
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	plan.Set(ctx, &GroupZoneAssignmentResourceModel{ID: types.StringValue("3/7"), GroupID: types.Int64Value(3), ZoneID: types.Int64Value(4011)})
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	res.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	want := []string{"POST /api/v2/groups/3/zones", "DELETE /api/v2/groups/3/zones/7"}
	if len(calls) != 2 || calls[0] != want[0] || calls[1] != want[1] {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	var got GroupZoneAssignmentResourceModel
	resp.State.Get(ctx, &got)
	if got.ID.ValueString() != "3/4011" || got.ZoneID.ValueInt64() != 4011 {
		t.Fatalf("state = %s / %d, want 3/4011", got.ID.ValueString(), got.ZoneID.ValueInt64())
	}
}

func TestGroupZoneAssignmentUpdate_ARefusedOldRemovalOnlyWarns(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"success":false,"message":"Cannot remove the last owner"}`))
			return
		}
		respondJSON(t, w, nil)
	})
	ctx := context.Background()
	res := &GroupZoneAssignmentResource{client: client}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	state := tfsdk.State{Schema: schemaResp.Schema}
	state.Set(ctx, &GroupZoneAssignmentResourceModel{ID: types.StringValue("3/7"), GroupID: types.Int64Value(3), ZoneID: types.Int64Value(7)})
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	plan.Set(ctx, &GroupZoneAssignmentResourceModel{ID: types.StringValue("3/7"), GroupID: types.Int64Value(3), ZoneID: types.Int64Value(4011)})
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	res.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, resp)

	if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("want one warning and no error, got %v", resp.Diagnostics)
	}
	var got GroupZoneAssignmentResourceModel
	resp.State.Get(ctx, &got)
	if got.ID.ValueString() != "3/4011" {
		t.Fatalf("state id = %s, want 3/4011", got.ID.ValueString())
	}
}
