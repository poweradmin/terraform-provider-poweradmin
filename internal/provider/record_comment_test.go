// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func strp(s string) *string { return &s }

func TestRecordComment_Decoding(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		wantPresent bool
		wantValue   *string
	}{
		{"absent", `{"id":1,"name":"www","type":"A"}`, false, nil},
		{"null", `{"id":1,"name":"www","type":"A","comment":null}`, true, nil},
		{"empty", `{"id":1,"name":"www","type":"A","comment":""}`, true, strp("")},
		{"value", `{"id":1,"name":"www","type":"A","comment":"hello"}`, true, strp("hello")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rec Record
			if err := json.Unmarshal([]byte(tc.body), &rec); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if rec.Comment.Present != tc.wantPresent {
				t.Errorf("Present = %v, want %v", rec.Comment.Present, tc.wantPresent)
			}
			if (rec.Comment.Value == nil) != (tc.wantValue == nil) || (tc.wantValue != nil && *rec.Comment.Value != *tc.wantValue) {
				t.Errorf("Value = %v, want %v", rec.Comment.Value, tc.wantValue)
			}
		})
	}
}

func TestRecordComment_ReadFromClient(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, RecordResponse{Record: Record{ID: "10", ZoneID: 1, Name: "www.example.com", Type: "A", Content: "192.0.2.1", TTL: 3600, Comment: RecordComment{Present: true, Value: strp("note")}}})
	})
	rec, err := client.GetRecord(context.Background(), 1, "10")
	if err != nil {
		t.Fatal(err)
	}
	if !rec.Comment.Present || rec.Comment.Value == nil || *rec.Comment.Value != "note" {
		t.Fatalf("comment not decoded: %+v", rec.Comment)
	}
}

func TestApplyComment(t *testing.T) {
	cases := []struct {
		name  string
		prior types.String
		api   RecordComment
		want  types.String
	}{
		{"server value wins", types.StringValue("old"), RecordComment{Present: true, Value: strp("new")}, types.StringValue("new")},
		{"null becomes empty", types.StringValue("old"), RecordComment{Present: true}, types.StringValue("")},
		{"absent keeps state", types.StringValue("kept"), RecordComment{}, types.StringValue("kept")},
		{"absent with unknown plan", types.StringUnknown(), RecordComment{}, types.StringValue("")},
		{"absent after import", types.StringNull(), RecordComment{}, types.StringValue("")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := RecordResourceModel{Comment: tc.prior}
			m.applyComment(tc.api)
			if !m.Comment.Equal(tc.want) {
				t.Errorf("Comment = %s, want %s", m.Comment, tc.want)
			}
		})
	}
}

// recordCommentModel builds a full record model with the given comment.
func recordCommentModel(comment types.String) *RecordResourceModel {
	return &RecordResourceModel{
		ID: types.StringValue("10"), ZoneID: types.Int64Value(1), Name: types.StringValue("www.example.com"),
		Type: types.StringValue("A"), Content: types.StringValue("192.0.2.1"), TTL: types.Int64Value(3600),
		Priority: types.Int64Value(0), Disabled: types.BoolValue(false), CreatePTR: types.BoolValue(false),
		Comment: comment,
	}
}

// commentServer records the request bodies and answers with the given record JSON.
func commentServer(t *testing.T, reply string, bodies *[]map[string]interface{}) *Client {
	return newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var m map[string]interface{}
		_ = json.Unmarshal(raw, &m)
		*bodies = append(*bodies, m)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"record":` + reply + `}}`))
	})
}

func TestRecordCreate_Comment(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		plan    types.String
		reply   string
		wantKey bool
		wantVal string
		want    string
	}{
		{"with comment", types.StringValue("hello"), `"comment":"hello"`, true, "hello", "hello"},
		{"unset sends no key", types.StringUnknown(), `"comment":null`, false, "", ""},
		{"unset on older server", types.StringUnknown(), `"x":1`, false, "", ""},
		{"empty string is sent", types.StringValue(""), `"comment":null`, true, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var bodies []map[string]interface{}
			client := commentServer(t, `{"id":10,"zone_id":1,"name":"www.example.com","type":"A","content":"192.0.2.1","ttl":3600,"disabled":false,`+tc.reply+`}`, &bodies)
			res := &RecordResource{client: client}
			schemaResp := &resource.SchemaResponse{}
			res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
			plan := tfsdk.Plan{Schema: schemaResp.Schema}
			plan.Set(ctx, recordCommentModel(tc.plan))
			resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}}
			res.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			v, ok := bodies[0]["comment"]
			if ok != tc.wantKey || (ok && v != tc.wantVal) {
				t.Errorf("request comment = %v (present %v), want present %v value %q", v, ok, tc.wantKey, tc.wantVal)
			}
			var got RecordResourceModel
			resp.State.Get(ctx, &got)
			if got.Comment.ValueString() != tc.want || got.Comment.IsNull() || got.Comment.IsUnknown() {
				t.Errorf("state comment = %s, want %q", got.Comment, tc.want)
			}
		})
	}
}

func TestRecordUpdate_Comment(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		config  types.String
		plan    types.String
		reply   string
		wantKey bool
		wantVal string
		want    string
	}{
		{"new comment", types.StringValue("second"), types.StringValue("second"), `"comment":"second"`, true, "second", "second"},
		{"unset sends no key and keeps state", types.StringNull(), types.StringValue("first"), `"comment":"first"`, false, "", "first"},
		{"empty string clears", types.StringValue(""), types.StringValue(""), `"comment":null`, true, "", ""},
		{"older server keeps state", types.StringValue("second"), types.StringValue("second"), `"x":1`, true, "second", "second"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var bodies []map[string]interface{}
			client := commentServer(t, `{"id":10,"zone_id":1,"name":"www.example.com","type":"A","content":"192.0.2.1","ttl":3600,"disabled":false,`+tc.reply+`}`, &bodies)
			res := &RecordResource{client: client}
			schemaResp := &resource.SchemaResponse{}
			res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
			state := tfsdk.State{Schema: schemaResp.Schema}
			state.Set(ctx, recordCommentModel(types.StringValue("first")))
			plan := tfsdk.Plan{Schema: schemaResp.Schema}
			plan.Set(ctx, recordCommentModel(tc.plan))
			cfgModel := recordCommentModel(tc.config)
			cfgModel.ID = types.StringNull()
			cfgPlan := tfsdk.Plan{Schema: schemaResp.Schema}
			cfgPlan.Set(ctx, cfgModel)
			config := tfsdk.Config{Schema: schemaResp.Schema, Raw: cfgPlan.Raw}
			resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			res.Update(ctx, resource.UpdateRequest{State: state, Plan: plan, Config: config}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			v, ok := bodies[0]["comment"]
			if ok != tc.wantKey || (ok && v != tc.wantVal) {
				t.Errorf("request comment = %v (present %v), want present %v value %q", v, ok, tc.wantKey, tc.wantVal)
			}
			var got RecordResourceModel
			resp.State.Get(ctx, &got)
			if got.Comment.ValueString() != tc.want {
				t.Errorf("state comment = %s, want %q", got.Comment, tc.want)
			}
		})
	}
}

func TestRecordCreate_EmptyCommentClearsInheritedComment(t *testing.T) {
	ctx := context.Background()
	record := `{"id":10,"zone_id":1,"name":"www.example.com","type":"A","content":"192.0.2.1","ttl":3600,"disabled":false,`
	cases := []struct {
		name      string
		calls     []string // reply per request, in order
		wantCalls int
	}{
		{"inherited comment is cleared", []string{record + `"comment":"lb"}`, record + `"comment":null}`}, 2},
		{"no comment, no extra update", []string{record + `"comment":null}`}, 1},
		{"empty comment, no extra update", []string{record + `"comment":""}`}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var bodies []map[string]interface{}
			var methods []string
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				var m map[string]interface{}
				_ = json.Unmarshal(raw, &m)
				bodies = append(bodies, m)
				methods = append(methods, r.Method)
				reply := tc.calls[len(tc.calls)-1]
				if len(bodies) <= len(tc.calls) {
					reply = tc.calls[len(bodies)-1]
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":true,"data":{"record":` + reply + `}}`))
			})
			res := &RecordResource{client: client}
			schemaResp := &resource.SchemaResponse{}
			res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
			plan := tfsdk.Plan{Schema: schemaResp.Schema}
			plan.Set(ctx, recordCommentModel(types.StringValue("")))
			resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}}
			res.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if len(bodies) != tc.wantCalls {
				t.Fatalf("requests = %v, want %d", methods, tc.wantCalls)
			}
			if tc.wantCalls == 2 {
				if methods[1] != http.MethodPut {
					t.Errorf("second request = %s, want PUT", methods[1])
				}
				if len(bodies[1]) != 1 || bodies[1]["comment"] != "" {
					t.Errorf("PUT body = %v, want only comment \"\"", bodies[1])
				}
			}
			var got RecordResourceModel
			resp.State.Get(ctx, &got)
			if got.Comment.IsNull() || got.Comment.ValueString() != "" {
				t.Errorf("state comment = %s, want empty string", got.Comment)
			}
			if len(resp.Diagnostics) != 0 {
				t.Errorf("unexpected diagnostics: %v", resp.Diagnostics)
			}
		})
	}
}

func TestRecordCreateUpdate_WarnWhenCommentNotReturned(t *testing.T) {
	ctx := context.Background()
	var bodies []map[string]interface{}
	client := commentServer(t, `{"id":10,"zone_id":1,"name":"www.example.com","type":"A","content":"192.0.2.1","ttl":3600,"disabled":false}`, &bodies)
	res := &RecordResource{client: client}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	plan.Set(ctx, recordCommentModel(types.StringValue("hello")))
	created := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}}
	res.Create(ctx, resource.CreateRequest{Plan: plan}, created)
	if created.Diagnostics.HasError() || created.Diagnostics.WarningsCount() != 1 {
		t.Errorf("create diagnostics = %v, want exactly one warning", created.Diagnostics)
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	state.Set(ctx, recordCommentModel(types.StringValue("hello")))
	cfgPlan := tfsdk.Plan{Schema: schemaResp.Schema}
	cfgPlan.Set(ctx, recordCommentModel(types.StringValue("hello")))
	updated := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Update(ctx, resource.UpdateRequest{State: state, Plan: plan, Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: cfgPlan.Raw}}, updated)
	if updated.Diagnostics.HasError() || updated.Diagnostics.WarningsCount() != 1 {
		t.Errorf("update diagnostics = %v, want exactly one warning", updated.Diagnostics)
	}

	// Nothing sent, nothing to warn about
	plan.Set(ctx, recordCommentModel(types.StringUnknown()))
	quiet := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}}
	res.Create(ctx, resource.CreateRequest{Plan: plan}, quiet)
	if len(quiet.Diagnostics) != 0 {
		t.Errorf("unexpected diagnostics without a comment: %v", quiet.Diagnostics)
	}
}

func TestRecordCommentPlanModifier(t *testing.T) {
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	NewRecordResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema
	objType := s.Type().TerraformType(ctx)
	plan := tfsdk.Plan{Schema: s, Raw: recordTestValue(t, objType, "42", "192.0.2.1", 7200, 0)}

	cases := []struct {
		name        string
		state       types.String
		planValue   types.String
		config      types.String
		wantUnknown bool
		want        types.String
	}{
		{"prior comment is kept", types.StringValue("note"), types.StringUnknown(), types.StringNull(), false, types.StringValue("note")},
		{"prior empty comment is kept", types.StringValue(""), types.StringUnknown(), types.StringNull(), false, types.StringValue("")},
		{"null prior stays unknown", types.StringNull(), types.StringUnknown(), types.StringNull(), true, types.StringUnknown()},
		{"configured value is untouched", types.StringValue("old"), types.StringValue("new"), types.StringValue("new"), false, types.StringValue("new")},
		{"unknown config stays unknown", types.StringValue("old"), types.StringUnknown(), types.StringUnknown(), true, types.StringUnknown()},
		{"known config with unknown plan keeps prior", types.StringValue("old"), types.StringUnknown(), types.StringNull(), false, types.StringValue("old")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{Path: path.Root("comment"), StateValue: tc.state, PlanValue: tc.planValue, ConfigValue: tc.config, Plan: plan}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			recordCommentPlanModifier{}.PlanModifyString(ctx, req, resp)
			if resp.PlanValue.IsUnknown() != tc.wantUnknown || (!tc.wantUnknown && !resp.PlanValue.Equal(tc.want)) {
				t.Errorf("plan value = %s, want %s", resp.PlanValue, tc.want)
			}
		})
	}
}

func TestNoSurroundingWhitespace(t *testing.T) {
	cases := []struct {
		value   types.String
		wantErr bool
	}{
		{types.StringValue("fine"), false},
		{types.StringValue("inner space ok"), false},
		{types.StringValue(""), false},
		{types.StringNull(), false},
		{types.StringUnknown(), false},
		{types.StringValue(" lead"), true},
		{types.StringValue("trail "), true},
		{types.StringValue("trail\n"), true},
		{types.StringValue("\ttab"), true},
	}
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		noSurroundingWhitespace{}.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("comment"), ConfigValue: tc.value}, resp)
		if resp.Diagnostics.HasError() != tc.wantErr {
			t.Errorf("value %s: error = %v, want %v", tc.value, resp.Diagnostics.HasError(), tc.wantErr)
		}
	}
}

func TestRecordsDataSource_ReadMapsComment(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"records":[` +
			`{"id":1,"zone_id":1,"name":"a.example.com","type":"A","content":"192.0.2.1","ttl":300,"disabled":false,"comment":"note"},` +
			`{"id":2,"zone_id":1,"name":"b.example.com","type":"A","content":"192.0.2.2","ttl":300,"disabled":false,"comment":null},` +
			`{"id":3,"zone_id":1,"name":"c.example.com","type":"A","content":"192.0.2.3","ttl":300,"disabled":false}]}}`))
	})
	ds := &RecordsDataSource{client: client}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema

	config := tfsdk.Config{Schema: s}
	cfgState := tfsdk.State{Schema: s}
	cfgState.Set(ctx, &RecordsDataSourceModel{ZoneID: types.Int64Value(1), Type: types.StringNull(), Name: types.StringNull()})
	config.Raw = cfgState.Raw
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics: %v", resp.Diagnostics)
	}
	var got RecordsDataSourceModel
	resp.State.Get(ctx, &got)
	if len(got.Records) != 3 {
		t.Fatalf("records = %d, want 3", len(got.Records))
	}
	if got.Records[0].Comment.ValueString() != "note" {
		t.Errorf("record 0 comment = %s, want note", got.Records[0].Comment)
	}
	if !got.Records[1].Comment.IsNull() || !got.Records[2].Comment.IsNull() {
		t.Errorf("null and absent comments should be null, got %s and %s", got.Records[1].Comment, got.Records[2].Comment)
	}
}

func TestRecordCreate_FailedClearKeepsPlannedEmptyComment(t *testing.T) {
	ctx := context.Background()
	var methods []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"record":{"id":10,"zone_id":1,"name":"www.example.com","type":"A","content":"192.0.2.1","ttl":3600,"disabled":false,"comment":"lb"}}}`))
	})
	res := &RecordResource{client: client}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	plan.Set(ctx, recordCommentModel(types.StringValue("")))
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}}
	res.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("diagnostics = %v, want exactly one warning and no error", resp.Diagnostics)
	}
	if len(methods) < 2 || methods[1] != http.MethodPut {
		t.Fatalf("requests = %v, want POST then PUT", methods)
	}
	var got RecordResourceModel
	resp.State.Get(ctx, &got)
	if got.Comment.IsNull() || got.Comment.ValueString() != "" {
		t.Errorf("state comment = %s, want the planned empty string", got.Comment)
	}
}
