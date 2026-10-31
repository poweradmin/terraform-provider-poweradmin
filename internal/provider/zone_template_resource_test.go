// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneTemplateResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneTemplateResourceConfig("acc-template", "Acceptance template"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone_template.test", "name", "acc-template"),
					resource.TestCheckResourceAttr("poweradmin_zone_template.test", "description", "Acceptance template"),
					resource.TestCheckResourceAttr("poweradmin_zone_template.test", "is_global", "false"),
					resource.TestCheckResourceAttrSet("poweradmin_zone_template.test", "id"),
					resource.TestCheckResourceAttrSet("poweradmin_zone_template.test", "owner"),
				),
			},
			{
				ResourceName:      "poweradmin_zone_template.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccZoneTemplateResourceConfig("acc-template", "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone_template.test", "description", "Updated description"),
				),
			},
		},
	})
}

// TestAccZoneTemplateGroups needs Poweradmin 4.6.0+; the data source reads the
// shares back from the server, which an older server never stores.
func TestAccZoneTemplateGroups(t *testing.T) {
	tmpl := "poweradmin_zone_template.test"
	data := "data.poweradmin_zone_template.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneTemplateGroupsConfig("Shared template", `groups = [poweradmin_group.one.id]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(tmpl, "groups.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(tmpl, "groups.*", "poweradmin_group.one", "id"),
					resource.TestCheckResourceAttr(data, "groups.#", "1"),
				),
			},
			{
				// Leaving groups out keeps the stored shares
				Config: testAccZoneTemplateGroupsConfig("Kept shares", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(tmpl, "groups.#", "1"),
					resource.TestCheckResourceAttr(data, "groups.#", "1"),
				),
			},
			{
				Config: testAccZoneTemplateGroupsConfig("Shared template", `groups = [poweradmin_group.one.id, poweradmin_group.two.id]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(tmpl, "groups.#", "2"),
					resource.TestCheckResourceAttr(data, "groups.#", "2"),
				),
			},
			{
				ResourceName:      tmpl,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// A global template has no owner and no shares
				Config: testAccZoneTemplateGroupsConfig("Shared template", `is_global = true`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(tmpl, "owner", "0"),
					resource.TestCheckResourceAttr(tmpl, "groups.#", "0"),
					resource.TestCheckResourceAttr(data, "groups.#", "0"),
				),
			},
			{
				Config: testAccZoneTemplateGroupsConfig("Shared template", "is_global = false\n  groups    = [poweradmin_group.two.id]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(tmpl, "owner", func(v string) error {
						if v == "0" {
							return fmt.Errorf("a template that is no longer global needs an owner, got %s", v)
						}
						return nil
					}),
					resource.TestCheckResourceAttr(tmpl, "groups.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(tmpl, "groups.*", "poweradmin_group.two", "id"),
				),
			},
			{
				Config: testAccZoneTemplateGroupsConfig("Shared template", `groups = []`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(tmpl, "groups.#", "0"),
					resource.TestCheckResourceAttr(data, "groups.#", "0"),
				),
			},
		},
	})
}

func TestAccZoneTemplateRecordResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneTemplateRecordResourceConfig("acc-template-rec", "www", "A", "192.0.2.10", 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone_template_record.www", "name", "www"),
					resource.TestCheckResourceAttr("poweradmin_zone_template_record.www", "type", "A"),
					resource.TestCheckResourceAttr("poweradmin_zone_template_record.www", "content", "192.0.2.10"),
					resource.TestCheckResourceAttr("poweradmin_zone_template_record.www", "ttl", "3600"),
					resource.TestCheckResourceAttrSet("poweradmin_zone_template_record.www", "id"),
					resource.TestCheckResourceAttrSet("poweradmin_zone_template_record.www", "record_id"),
				),
			},
			{
				ResourceName:      "poweradmin_zone_template_record.www",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccZoneTemplateRecordResourceConfig("acc-template-rec", "www", "A", "192.0.2.20", 7200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone_template_record.www", "content", "192.0.2.20"),
					resource.TestCheckResourceAttr("poweradmin_zone_template_record.www", "ttl", "7200"),
				),
			},
		},
	})
}

func testAccZoneTemplateResourceConfig(name, description string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone_template" "test" {
  name        = %[1]q
  description = %[2]q
}
`, name, description)
}

// testAccZoneTemplateGroupsConfig builds a template with two groups to share it
// with; extra holds the other template attributes under test.
func testAccZoneTemplateGroupsConfig(description, extra string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_group" "one" {
  name          = "acc-template-share-one"
  description   = "Template sharing acceptance test"
  perm_templ_id = 6
}

resource "poweradmin_group" "two" {
  name          = "acc-template-share-two"
  description   = "Template sharing acceptance test"
  perm_templ_id = 6
}

resource "poweradmin_zone_template" "test" {
  name        = "acc-template-shares"
  description = %[1]q
  %[2]s
}

data "poweradmin_zone_template" "test" {
  id         = poweradmin_zone_template.test.id
  depends_on = [poweradmin_zone_template.test]
}
`, description, extra)
}

func testAccZoneTemplateRecordResourceConfig(tmplName, recName, recType, content string, ttl int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone_template" "test" {
  name        = %[1]q
  description = "Template for record acceptance test"
}

resource "poweradmin_zone_template_record" "www" {
  template_id = poweradmin_zone_template.test.id
  name        = %[2]q
  type        = %[3]q
  content     = %[4]q
  ttl         = %[5]d
}
`, tmplName, recName, recType, content, ttl)
}

func TestZoneTemplateGroupIDs(t *testing.T) {
	if groupIDs(types.SetNull(types.Int64Type)) != nil {
		t.Errorf("an unset groups attribute must not be sent")
	}
	if groupIDs(types.SetUnknown(types.Int64Type)) != nil {
		t.Errorf("an unknown groups attribute must not be sent")
	}
	ids := groupIDs(types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(3)}))
	if ids == nil || len(*ids) != 1 || (*ids)[0] != 3 {
		t.Errorf("expected [3], got %v", ids)
	}
	empty := groupIDs(types.SetValueMust(types.Int64Type, []attr.Value{}))
	if empty == nil || len(*empty) != 0 {
		t.Errorf("an empty set must be sent to remove every share, got %v", empty)
	}
}

func TestZoneTemplateApplyToModelReadsTheGroups(t *testing.T) {
	var data ZoneTemplateResourceModel
	shared := []ZoneTemplateGroup{{ID: 3, Name: "Ops"}, {ID: 4, Name: "Dev"}}
	(&ZoneTemplateResource{}).applyToModel(&ZoneTemplate{ID: 5, Name: "t", Groups: &shared}, &data)
	if len(data.Groups.Elements()) != 2 {
		t.Fatalf("expected two groups, got %v", data.Groups)
	}

	none := []ZoneTemplateGroup{}
	(&ZoneTemplateResource{}).applyToModel(&ZoneTemplate{ID: 5, Name: "t", Groups: &none}, &data)
	if data.Groups.IsNull() || len(data.Groups.Elements()) != 0 {
		t.Errorf("an empty groups list must read as an empty set, got %v", data.Groups)
	}
}

func TestZoneTemplateApplyToModelKeepsThePlanOnAServerWithoutGroups(t *testing.T) {
	planned := types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(3)})
	data := ZoneTemplateResourceModel{Groups: planned}
	(&ZoneTemplateResource{}).applyToModel(&ZoneTemplate{ID: 5, Name: "t"}, &data)
	if !data.Groups.Equal(planned) {
		t.Errorf("an older server returns no groups; the planned value must stay, got %v", data.Groups)
	}

	var diags diag.Diagnostics
	sent := []int{3}
	warnIfGroupsNotReturned(&sent, &ZoneTemplate{ID: 5}, &diags)
	if diags.WarningsCount() != 1 {
		t.Errorf("expected a warning when sent groups are not returned, got %v", diags)
	}
	empty := []ZoneTemplateGroup{}
	diags = nil
	warnIfGroupsNotReturned(&sent, &ZoneTemplate{ID: 5, Groups: &empty}, &diags)
	warnIfGroupsNotReturned(nil, &ZoneTemplate{ID: 5}, &diags)
	warnIfGroupsNotReturned(&[]int{}, &ZoneTemplate{ID: 5}, &diags)
	if diags.WarningsCount() != 0 {
		t.Errorf("no warning expected, got %v", diags)
	}
}

// groupsPlanCase runs the groups plan modifier for a configuration and a stored state.
func groupsPlanCase(t *testing.T, configGroups, configGlobal, stateGroups tftypes.Value) types.Set {
	t.Helper()
	ctx := context.Background()
	var schemaResp fwresource.SchemaResponse
	(&ZoneTemplateResource{}).Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("the schema is not an object")
	}
	setType := tftypes.Set{ElementType: tftypes.Number}
	object := func(groups, global tftypes.Value) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.Number, 5),
			"name":        tftypes.NewValue(tftypes.String, "t"),
			"description": tftypes.NewValue(tftypes.String, "d"),
			"is_global":   global,
			"owner":       tftypes.NewValue(tftypes.Number, 1),
			"groups":      groups,
		})
	}
	unknownGroups := tftypes.NewValue(setType, tftypes.UnknownValue)
	stateValue, _ := (types.SetType{ElemType: types.Int64Type}).ValueFromTerraform(ctx, stateGroups)
	stateSet, ok := stateValue.(types.Set)
	if !ok {
		t.Fatal("the state groups are not a set")
	}
	req := planmodifier.SetRequest{
		Path:        path.Root("groups"),
		Config:      tfsdk.Config{Schema: schemaResp.Schema, Raw: object(configGroups, configGlobal)},
		Plan:        tfsdk.Plan{Schema: schemaResp.Schema, Raw: object(unknownGroups, configGlobal)},
		State:       tfsdk.State{Schema: schemaResp.Schema, Raw: object(stateGroups, tftypes.NewValue(tftypes.Bool, false))},
		ConfigValue: types.SetNull(types.Int64Type),
		PlanValue:   types.SetUnknown(types.Int64Type),
		StateValue:  stateSet,
	}
	if !configGroups.IsNull() {
		configValue, _ := (types.SetType{ElemType: types.Int64Type}).ValueFromTerraform(ctx, configGroups)
		configSet, ok := configValue.(types.Set)
		if !ok {
			t.Fatal("the configured groups are not a set")
		}
		req.ConfigValue = configSet
		req.PlanValue = req.ConfigValue
	}
	resp := planmodifier.SetResponse{PlanValue: req.PlanValue}
	zoneTemplateGroupsPlanModifier{}.PlanModifySet(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("plan modifier failed: %v", resp.Diagnostics)
	}
	return resp.PlanValue
}

func TestZoneTemplateGroupsPlanModifier(t *testing.T) {
	setType := tftypes.Set{ElementType: tftypes.Number}
	stored := tftypes.NewValue(setType, []tftypes.Value{tftypes.NewValue(tftypes.Number, 3), tftypes.NewValue(tftypes.Number, 4)})
	nullGroups := tftypes.NewValue(setType, nil)
	notGlobal := tftypes.NewValue(tftypes.Bool, nil)
	global := tftypes.NewValue(tftypes.Bool, true)

	if got := groupsPlanCase(t, nullGroups, notGlobal, stored); len(got.Elements()) != 2 {
		t.Errorf("groups left out must plan the stored shares, got %v", got)
	}
	if got := groupsPlanCase(t, nullGroups, global, stored); got.IsUnknown() || len(got.Elements()) != 0 {
		t.Errorf("a template made global keeps no shares, got %v", got)
	}
	if got := groupsPlanCase(t, nullGroups, tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue), stored); !got.IsUnknown() {
		t.Errorf("an is_global known only at apply time leaves the shares unknown, got %v", got)
	}
	configured := tftypes.NewValue(setType, []tftypes.Value{tftypes.NewValue(tftypes.Number, 9)})
	if got := groupsPlanCase(t, configured, notGlobal, stored); len(got.Elements()) != 1 {
		t.Errorf("configured groups are the plan, got %v", got)
	}
}

// ownerPlanCase runs the owner plan modifier for a configured is_global against a
// stored template that is not global and belongs to user 1.
func ownerPlanCase(t *testing.T, configGlobal tftypes.Value) types.Int64 {
	t.Helper()
	ctx := context.Background()
	var schemaResp fwresource.SchemaResponse
	(&ZoneTemplateResource{}).Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("the schema is not an object")
	}
	object := func(global, owner tftypes.Value) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.Number, 5),
			"name":        tftypes.NewValue(tftypes.String, "t"),
			"description": tftypes.NewValue(tftypes.String, "d"),
			"is_global":   global,
			"owner":       owner,
			"groups":      tftypes.NewValue(tftypes.Set{ElementType: tftypes.Number}, nil),
		})
	}
	unknownOwner := tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)
	req := planmodifier.Int64Request{
		Path:        path.Root("owner"),
		Config:      tfsdk.Config{Schema: schemaResp.Schema, Raw: object(configGlobal, tftypes.NewValue(tftypes.Number, nil))},
		Plan:        tfsdk.Plan{Schema: schemaResp.Schema, Raw: object(configGlobal, unknownOwner)},
		State:       tfsdk.State{Schema: schemaResp.Schema, Raw: object(tftypes.NewValue(tftypes.Bool, false), tftypes.NewValue(tftypes.Number, 1))},
		ConfigValue: types.Int64Null(),
		PlanValue:   types.Int64Unknown(),
		StateValue:  types.Int64Value(1),
	}
	resp := planmodifier.Int64Response{PlanValue: req.PlanValue}
	zoneTemplateOwnerPlanModifier{}.PlanModifyInt64(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("plan modifier failed: %v", resp.Diagnostics)
	}
	return resp.PlanValue
}

func TestZoneTemplateOwnerPlanModifier(t *testing.T) {
	if got := ownerPlanCase(t, tftypes.NewValue(tftypes.Bool, nil)); !got.Equal(types.Int64Value(1)) {
		t.Errorf("is_global left out must plan the stored owner, got %v", got)
	}
	if got := ownerPlanCase(t, tftypes.NewValue(tftypes.Bool, false)); !got.Equal(types.Int64Value(1)) {
		t.Errorf("an unchanged is_global must plan the stored owner, got %v", got)
	}
	if got := ownerPlanCase(t, tftypes.NewValue(tftypes.Bool, true)); !got.IsUnknown() {
		t.Errorf("a changed is_global leaves the owner unknown, got %v", got)
	}
	if got := ownerPlanCase(t, tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue)); !got.IsUnknown() {
		t.Errorf("an is_global known only at apply time leaves the owner unknown, got %v", got)
	}
}

// TestZoneTemplateUpdateSendsGroupsOnlyWhenConfigured checks the request body: a plan
// that carries the stored shares must not send them, or a group member gets a 403.
func TestZoneTemplateUpdateSendsGroupsOnlyWhenConfigured(t *testing.T) {
	ctx := context.Background()
	var schemaResp fwresource.SchemaResponse
	(&ZoneTemplateResource{}).Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("the schema is not an object")
	}
	setType := tftypes.Set{ElementType: tftypes.Number}
	stored := tftypes.NewValue(setType, []tftypes.Value{tftypes.NewValue(tftypes.Number, 3)})
	object := func(groups tftypes.Value) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.Number, 5),
			"name":        tftypes.NewValue(tftypes.String, "t"),
			"description": tftypes.NewValue(tftypes.String, "changed"),
			"is_global":   tftypes.NewValue(tftypes.Bool, false),
			"owner":       tftypes.NewValue(tftypes.Number, 7),
			"groups":      groups,
		})
	}

	for name, configGroups := range map[string]tftypes.Value{
		"left out":   tftypes.NewValue(setType, nil),
		"configured": stored,
	} {
		var putBody map[string]interface{}
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut {
				if err := json.NewDecoder(r.Body).Decode(&putBody); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				respondJSON(t, w, nil)
				return
			}
			groups := []ZoneTemplateGroup{{ID: 3, Name: "Ops"}}
			respondJSON(t, w, map[string]interface{}{"template": ZoneTemplate{ID: 5, Name: "t", Description: "changed", Owner: 7, Groups: &groups}})
		})

		req := fwresource.UpdateRequest{
			Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: object(configGroups)},
			Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: object(stored)},
			State:  tfsdk.State{Schema: schemaResp.Schema, Raw: object(stored)},
		}
		resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: object(stored)}}
		(&ZoneTemplateResource{client: client}).Update(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s: update failed: %v", name, resp.Diagnostics)
		}

		_, sent := putBody["groups"]
		if want := name == "configured"; sent != want {
			t.Errorf("%s: groups sent = %v, want %v (body %v)", name, sent, want, putBody)
		}
	}
}
