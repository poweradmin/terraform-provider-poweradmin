// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func strPtr(s string) *string { return &s }

func TestListDnssecKeys(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/zones/7/dnssec/keys" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, []DnssecKey{
			{ID: 1, Type: "csk", KeyTag: 12345, Algorithm: strPtr("ecdsa256"), AlgorithmID: 13, Bits: 256, Active: true,
				DNSKEY: strPtr("257 3 13 AAAA"), DS: []string{"12345 13 2 ABCD"}},
			{ID: 2, Type: "zsk", KeyTag: 54321, AlgorithmID: 250, Bits: 256},
		})
	})

	keys, err := client.ListDnssecKeys(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].KeyTag != 12345 || *keys[0].Algorithm != "ecdsa256" || keys[0].DS[0] != "12345 13 2 ABCD" {
		t.Errorf("first key decoded wrong: %+v", keys[0])
	}
	if keys[1].Algorithm != nil || keys[1].DNSKEY != nil {
		t.Errorf("unnamed algorithm and missing dnskey must decode as nil: %+v", keys[1])
	}
}

// A 404 means the key or zone is gone; a 502 means PowerDNS did not answer and
// must never be mistaken for "gone", or Terraform would drop and recreate keys.
func TestGetDnssecKey_NotFoundVersusUnreachable(t *testing.T) {
	for _, tc := range []struct {
		status   int
		notFound bool
	}{
		{http.StatusNotFound, true},
		{http.StatusBadGateway, false},
	} {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v2/zones/7/dnssec/keys/3" {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			respondError(t, w, tc.status, "nope")
		})

		_, err := client.GetDnssecKey(context.Background(), 7, 3)
		if err == nil {
			t.Fatalf("HTTP %d: expected an error", tc.status)
		}
		if IsNotFoundError(err) != tc.notFound {
			t.Errorf("HTTP %d: IsNotFoundError = %t, want %t", tc.status, IsNotFoundError(err), tc.notFound)
		}
	}
}

// The API creates keys inactive unless told otherwise, so active=false must be sent, not omitted.
func TestCreateDnssecKey_SendsActiveExplicitly(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/zones/7/dnssec/keys" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]interface{}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("bad body: %v", err)
		}
		if got["type"] != "ksk" || got["algorithm"] != "rsasha256" || got["bits"] != float64(2048) {
			t.Errorf("unexpected body: %s", body)
		}
		if active, ok := got["active"]; !ok || active != false {
			t.Errorf("active must be sent as false, body: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"success":true,"data":{"id":9,"type":"ksk","keytag":1,"algorithm":"rsasha256","algorithm_id":8,"bits":2048,"active":false,"dnskey":"257 3 8 AAAA","ds":[]}}`))
	})

	key, err := client.CreateDnssecKey(context.Background(), 7, CreateDnssecKeyRequest{Type: "ksk", Algorithm: "rsasha256", Bits: 2048, Active: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.ID != 9 || key.Active {
		t.Errorf("unexpected key: %+v", key)
	}
}

func TestSetDnssecKeyActive_UsesPatch(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v2/zones/7/dnssec/keys/9" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if strings.TrimSpace(string(body)) != `{"active":false}` {
			t.Errorf("unexpected body: %s", body)
		}
		respondJSON(t, w, DnssecKey{ID: 9, Type: "ksk", Active: false})
	})

	key, err := client.SetDnssecKeyActive(context.Background(), 7, 9, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.Active {
		t.Error("expected the key to come back inactive")
	}
}

func TestDeleteDnssecKey(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/zones/7/dnssec/keys/9" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, nil)
	})

	if err := client.DeleteDnssecKey(context.Background(), 7, 9); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetZoneDnssec(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/zones/7/dnssec" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, map[string]interface{}{
			"enabled":   true,
			"presigned": false,
			"ds_records": []map[string]interface{}{
				{"key_tag": 12345, "algorithm": 13, "digest_type": 2, "digest": "ABCD"},
			},
			"dnskey": "257 3 13 AAAA",
		})
	})

	status, err := client.GetZoneDnssec(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Enabled || len(status.DSRecords) != 1 || status.DSRecords[0].DigestType != 2 || *status.DNSKEY != "257 3 13 AAAA" {
		t.Errorf("status decoded wrong: %+v", status)
	}
}

func TestGetZoneDnssec_UnreachableIsNotNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondError(t, w, http.StatusBadGateway, "Failed to retrieve DNSSEC status from PowerDNS")
	})

	_, err := client.GetZoneDnssec(context.Background(), 7)
	if err == nil || IsNotFoundError(err) {
		t.Fatalf("a 502 must be an error other than not-found, got: %v", err)
	}
}

func TestSetZoneDnssec(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/v2/zones/7/dnssec" {
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			var got SetZoneDnssecRequest
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil || got.Enabled != enabled {
				t.Errorf("expected enabled=%t, got %+v (%v)", enabled, got, err)
			}
			respondJSON(t, w, map[string]interface{}{"enabled": enabled, "presigned": false, "ds_records": []interface{}{}, "dnskey": nil})
		})

		status, err := client.SetZoneDnssec(context.Background(), 7, enabled)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status.Enabled != enabled || status.DNSKEY != nil {
			t.Errorf("unexpected status: %+v", status)
		}
	}
}

func TestValidateDnssecAlgorithmBits(t *testing.T) {
	for _, tc := range []struct {
		algorithm string
		bits      int64
		wantErr   bool
	}{
		{"ecdsa256", 256, false},
		{"ecdsa384", 384, false},
		{"ed25519", 256, false},
		{"ed448", 456, false},
		{"rsasha256", 2048, false},
		{"rsasha1-nsec3-sha1", 1024, false},
		{"ecdsa256", -1, false}, // size not known yet
		{"rsasha256", 4096, true},
		{"ecdsa256", 384, true},
		{"ed448", 448, true},
		{"ECDSA256", 256, true},
		{"gost", 512, true},
	} {
		var diags diag.Diagnostics
		validateDnssecAlgorithmBits(tc.algorithm, tc.bits, &diags)
		if diags.HasError() != tc.wantErr {
			t.Errorf("%s/%d: HasError = %t, want %t (%v)", tc.algorithm, tc.bits, diags.HasError(), tc.wantErr, diags)
		}
	}
}

func TestValidateDnssecKeyType(t *testing.T) {
	for keyType, wantErr := range map[string]bool{"ksk": false, "zsk": false, "csk": false, "KSK": true, "both": true, "": true} {
		var diags diag.Diagnostics
		validateDnssecKeyType(keyType, &diags)
		if diags.HasError() != wantErr {
			t.Errorf("%q: HasError = %t, want %t", keyType, diags.HasError(), wantErr)
		}
	}
}

// PowerDNS reports a KSK as csk until an active ZSK shares its algorithm, so the
// configured type must survive as long as the stored SEP flag agrees with it.
func TestDnssecKeyType(t *testing.T) {
	ksk := strPtr("257 3 8 AAAA")
	zsk := strPtr("256 3 8 AAAA")
	for _, tc := range []struct {
		name     string
		current  string
		reported string
		dnskey   *string
		want     string
	}{
		{"ksk read back as csk", "ksk", "csk", ksk, "ksk"},
		{"zsk read back as csk", "zsk", "csk", zsk, "zsk"},
		{"csk read back as ksk", "csk", "ksk", ksk, "csk"},
		{"configured zsk but key is SEP", "zsk", "csk", ksk, "csk"},
		{"configured ksk but key is not SEP", "ksk", "zsk", zsk, "zsk"},
		{"import keeps a consistent role", "", "ksk", ksk, "ksk"},
		{"import of a lone ZSK", "", "csk", zsk, "zsk"},
		{"import of a lone KSK takes the reported role", "", "csk", ksk, "csk"},
		{"no dnskey keeps config", "ksk", "csk", nil, "ksk"},
		{"no dnskey on import", "", "csk", nil, "csk"},
	} {
		got := dnssecKeyType(tc.current, &DnssecKey{Type: tc.reported, DNSKEY: tc.dnskey})
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// Only a move between zsk and ksk/csk changes what PowerDNS stores, so only that replaces the key.
func TestDnssecKeyRoleChanged(t *testing.T) {
	for _, tc := range []struct {
		state, plan string
		unknown     bool
		want        bool
	}{
		{"csk", "ksk", false, false},
		{"ksk", "csk", false, false},
		{"zsk", "ksk", false, true},
		{"csk", "zsk", false, true},
		{"csk", "", true, true},
	} {
		plan := types.StringValue(tc.plan)
		if tc.unknown {
			plan = types.StringUnknown()
		}
		resp := &stringplanmodifier.RequiresReplaceIfFuncResponse{}
		dnssecKeyRoleChanged(context.Background(), planmodifier.StringRequest{StateValue: types.StringValue(tc.state), PlanValue: plan}, resp)
		if resp.RequiresReplace != tc.want {
			t.Errorf("%s -> %s (unknown=%t): RequiresReplace = %t, want %t", tc.state, tc.plan, tc.unknown, resp.RequiresReplace, tc.want)
		}
	}
}
