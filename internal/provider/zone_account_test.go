// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAccountChanged(t *testing.T) {
	cases := []struct {
		name  string
		state types.String
		plan  types.String
		want  bool
	}{
		{"same value", types.StringValue("corp"), types.StringValue("corp"), false},
		{"different value", types.StringValue("corp"), types.StringValue("other"), true},
		{"set on a zone without account", types.StringNull(), types.StringValue("corp"), true},
		{"removed from configuration", types.StringValue("corp"), types.StringNull(), false},
		{"empty against null", types.StringNull(), types.StringValue(""), false},
		{"unknown plan", types.StringValue("corp"), types.StringUnknown(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := accountChanged(tc.state, tc.plan); got != tc.want {
				t.Errorf("accountChanged(%v, %v) = %v, want %v", tc.state, tc.plan, got, tc.want)
			}
		})
	}
}

func TestReplacesZone(t *testing.T) {
	base := ZoneResourceModel{Name: types.StringValue("a.example"), Template: types.StringNull(), SOAEditAPI: types.StringNull()}
	renamed := base
	renamed.Name = types.StringValue("b.example")
	withTemplate := base
	withTemplate.Template = types.StringValue("5")
	unknownName := base
	unknownName.Name = types.StringUnknown()

	cases := []struct {
		name                string
		state, plan, config ZoneResourceModel
		want                bool
	}{
		{"nothing changes", base, base, base, false},
		{"name changes", base, renamed, renamed, true},
		{"name unknown", base, unknownName, unknownName, true},
		{"configured template changes", base, withTemplate, withTemplate, true},
		{"template removed from configuration", withTemplate, base, base, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := replacesZone(tc.state, tc.plan, tc.config); got != tc.want {
				t.Errorf("replacesZone() = %v, want %v", got, tc.want)
			}
		})
	}
}
