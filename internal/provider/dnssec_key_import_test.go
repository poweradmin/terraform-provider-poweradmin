// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const testISCKey = "Private-key-format: v1.3\nAlgorithm: 13 (ECDSAP256SHA256)\nPrivateKey: c2VjcmV0LWtleS1ieXRlcw==\n"

// testECDSAISCKey returns a fresh ECDSA P-256 private key in the ISC/BIND format
// and the base64 public key its DNSKEY record must carry.
func testECDSAISCKey(t *testing.T) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	d, err := key.Bytes()
	if err != nil {
		t.Fatalf("encode key: %v", err)
	}
	point, err := key.PublicKey.Bytes() // 0x04 || X || Y
	if err != nil {
		t.Fatalf("encode public key: %v", err)
	}
	pub := point[1:]
	isc := "Private-key-format: v1.2\nAlgorithm: 13 (ECDSAP256SHA256)\nPrivateKey: " + base64.StdEncoding.EncodeToString(d) + "\n"
	return isc, base64.StdEncoding.EncodeToString(pub)
}

// testRSAISCKey returns a fresh RSASHA256 private key of the given size in the ISC/BIND format.
func testRSAISCKey(t *testing.T, bits int) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	key.Precompute()
	b64 := func(n *big.Int) string { return base64.StdEncoding.EncodeToString(n.Bytes()) }
	return "Private-key-format: v1.2\nAlgorithm: 8 (RSASHA256)\n" +
		"Modulus: " + b64(key.N) + "\nPublicExponent: " + b64(big.NewInt(int64(key.E))) + "\n" +
		"PrivateExponent: " + b64(key.D) + "\nPrime1: " + b64(key.Primes[0]) + "\nPrime2: " + b64(key.Primes[1]) + "\n" +
		"Exponent1: " + b64(key.Precomputed.Dp) + "\nExponent2: " + b64(key.Precomputed.Dq) + "\nCoefficient: " + b64(key.Precomputed.Qinv) + "\n"
}

func TestISCKeyAlgorithm(t *testing.T) {
	for _, tc := range []struct {
		name, key, want string
	}{
		{"ecdsa", testISCKey, "ecdsa256"},
		{"lower case and no space, like Poweradmin accepts", "private-key-format:v1.2\nalgorithm:8 (RSASHA256)\n", "rsasha256"},
		{"surrounding whitespace is trimmed", "\n  \nPrivate-key-format: v1.3\nAlgorithm: 15 (ED25519)\n\n", "ed25519"},
		{"PEM", "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEE\n-----END EC PRIVATE KEY-----\n", ""},
		{"no format line", "Algorithm: 13 (ECDSAP256SHA256)\n", ""},
		{"no algorithm line", "Private-key-format: v1.3\nPrivateKey: AAAA\n", ""},
		{"unsupported algorithm", "Private-key-format: v1.3\nAlgorithm: 3 (DSA)\n", ""},
		{"too large", "Private-key-format: v1.3\nAlgorithm: 13\n" + strings.Repeat("A", maxPrivateKeyBytes), ""},
	} {
		got, ok := iscKeyAlgorithm(tc.key)
		if got != tc.want || ok != (tc.want != "") {
			t.Errorf("%s: got (%q, %v), want %q", tc.name, got, ok, tc.want)
		}
	}
}

func TestImportDnssecKeyRequestRedactsPrivateKey(t *testing.T) {
	req := ImportDnssecKeyRequest{Type: "ksk", PrivateKey: testISCKey, Active: true}
	logged := req.redactedForLog()
	if strings.Contains(logged, "c2VjcmV0") || strings.Contains(logged, "Private-key-format") {
		t.Errorf("redacted log contains the private key: %s", logged)
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var sent map[string]interface{}
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatal(err)
	}
	if sent["privatekey"] != testISCKey || sent["type"] != "ksk" || sent["active"] != true {
		t.Errorf("unexpected wire body: %s", body)
	}
}

func TestDnssecKeyTypePrefersReportedSEP(t *testing.T) {
	zskFlags := "256 3 13 AAAA"
	yes, no := true, false
	// sep wins over the DNSKEY flags when the server reports it
	if got := dnssecKeyType("", &DnssecKey{Type: "zsk", DNSKEY: &zskFlags, SEP: &yes}); got != "csk" {
		t.Errorf("sep=true with ZSK flags: got %q, want csk", got)
	}
	if got := dnssecKeyType("ksk", &DnssecKey{Type: "csk", SEP: &no}); got != "zsk" {
		t.Errorf("sep=false: got %q, want zsk", got)
	}
	// Without sep (4.5.0) the flags decide as before
	if got := dnssecKeyType("", &DnssecKey{Type: "zsk", DNSKEY: &zskFlags}); got != "zsk" {
		t.Errorf("no sep: got %q, want zsk", got)
	}
}

func TestPrivateKeyVersionChanged(t *testing.T) {
	for _, tc := range []struct {
		name        string
		state, plan types.Int64
		want        bool
	}{
		{"first version on an existing key", types.Int64Null(), types.Int64Value(1), true},
		{"bumped", types.Int64Value(1), types.Int64Value(2), true},
		{"unchanged", types.Int64Value(1), types.Int64Value(1), false},
		{"removed", types.Int64Value(1), types.Int64Null(), false},
	} {
		resp := &int64planmodifier.RequiresReplaceIfFuncResponse{}
		privateKeyVersionChanged(context.Background(), planmodifier.Int64Request{StateValue: tc.state, PlanValue: tc.plan}, resp)
		if resp.RequiresReplace != tc.want {
			t.Errorf("%s: RequiresReplace = %v, want %v", tc.name, resp.RequiresReplace, tc.want)
		}
	}
}

// dnssecKeyObject builds a raw resource value; unset attributes are null.
func dnssecKeyObject(t *testing.T, s fwresource.SchemaResponse, attrs map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	objType, ok := s.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatal("the schema is not an object")
	}
	values := map[string]tftypes.Value{}
	for name, typ := range objType.AttributeTypes {
		if v, ok := attrs[name]; ok {
			values[name] = v
		} else {
			values[name] = tftypes.NewValue(typ, nil)
		}
	}
	return tftypes.NewValue(objType, values)
}

func runDnssecKeyImportCreate(t *testing.T, configAlgorithm tftypes.Value, handler http.HandlerFunc) fwresource.CreateResponse {
	t.Helper()
	ctx := context.Background()
	var s fwresource.SchemaResponse
	(&DnssecKeyResource{}).Schema(ctx, fwresource.SchemaRequest{}, &s)
	str := func(v string) tftypes.Value { return tftypes.NewValue(tftypes.String, v) }
	num := func(v int) tftypes.Value { return tftypes.NewValue(tftypes.Number, v) }
	unknownStr := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	unknownNum := tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)

	common := map[string]tftypes.Value{
		"zone_id": num(10), "type": str("ksk"), "active": tftypes.NewValue(tftypes.Bool, true),
		"private_key_wo_version": num(1), "algorithm": configAlgorithm,
	}
	config := map[string]tftypes.Value{"private_key_wo": str(testISCKey)}
	plan := map[string]tftypes.Value{
		"id": unknownStr, "key_id": unknownNum, "keytag": unknownNum, "algorithm_id": unknownNum,
		"dnskey": unknownStr, "ds": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, tftypes.UnknownValue),
		"bits": unknownNum,
	}
	for k, v := range common {
		config[k] = v
		plan[k] = v
	}
	if configAlgorithm.IsNull() {
		plan["algorithm"] = unknownStr
	}

	req := fwresource.CreateRequest{
		Config: tfsdk.Config{Schema: s.Schema, Raw: dnssecKeyObject(t, s, config)},
		Plan:   tfsdk.Plan{Schema: s.Schema, Raw: dnssecKeyObject(t, s, plan)},
	}
	resp := fwresource.CreateResponse{State: tfsdk.State{Schema: s.Schema, Raw: dnssecKeyObject(t, s, nil)}}
	(&DnssecKeyResource{client: newTestClient(t, handler)}).Create(ctx, req, &resp)
	return resp
}

func importedKeyResponse(t *testing.T, w http.ResponseWriter, algorithm string, bits int) {
	dnskey := "257 3 13 AAAA"
	sep := true
	respondJSON(t, w, DnssecKey{ID: 7, Type: "csk", KeyTag: 4242, Algorithm: &algorithm, AlgorithmID: 13, Bits: bits, Active: true, DNSKEY: &dnskey, DS: []string{"4242 13 2 ABCD"}, SEP: &sep})
}

// TestDnssecKeyCreateImportsFromConfig checks that Create takes the write-only key
// from the config (it is null in the plan) and posts it to the import endpoint.
func TestDnssecKeyCreateImportsFromConfig(t *testing.T) {
	var body map[string]interface{}
	resp := runDnssecKeyImportCreate(t, tftypes.NewValue(tftypes.String, nil), func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/zones/10/dnssec/keys/import" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		importedKeyResponse(t, w, "ecdsa256", 256)
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("create failed: %v", resp.Diagnostics)
	}
	if body["privatekey"] != testISCKey || body["type"] != "ksk" || body["active"] != true {
		t.Errorf("unexpected import body: %v", body)
	}

	ctx := context.Background()
	var algorithm, privateKey, typ types.String
	var bits types.Int64
	resp.State.GetAttribute(ctx, path.Root("algorithm"), &algorithm)
	resp.State.GetAttribute(ctx, path.Root("bits"), &bits)
	resp.State.GetAttribute(ctx, path.Root("private_key_wo"), &privateKey)
	resp.State.GetAttribute(ctx, path.Root("type"), &typ)
	if algorithm.ValueString() != "ecdsa256" || bits.ValueInt64() != 256 {
		t.Errorf("state algorithm/bits = %v/%v, want ecdsa256/256 from the server", algorithm, bits)
	}
	if !privateKey.IsNull() {
		t.Error("the private key must not be stored in state")
	}
	if typ.ValueString() != "ksk" {
		t.Errorf("configured type must be kept while it matches sep, got %v", typ)
	}
}

// TestDnssecKeyImportMismatchDeletesKey checks the rollback when PowerDNS reports a
// different algorithm than configured.
func TestDnssecKeyImportMismatchDeletesKey(t *testing.T) {
	deleted := false
	resp := runDnssecKeyImportCreate(t, tftypes.NewValue(tftypes.String, "ecdsa384"), func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			importedKeyResponse(t, w, "ecdsa256", 256)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/zones/10/dnssec/keys/7":
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	if !resp.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(resp.Diagnostics), "deleted again") {
		t.Fatalf("want a mismatch error, got %v", resp.Diagnostics)
	}
	if !deleted {
		t.Error("the mismatching key must be deleted")
	}
	if !resp.State.Raw.IsNull() {
		var id types.String
		resp.State.GetAttribute(context.Background(), path.Root("id"), &id)
		if !id.IsNull() {
			t.Errorf("no state may be saved for a deleted key, got id %v", id)
		}
	}
}

// TestDnssecKeyImportMismatchKeepsKeyWhenDeleteFails tracks the key in state when
// the rollback delete fails, so it is not orphaned.
func TestDnssecKeyImportMismatchKeepsKeyWhenDeleteFails(t *testing.T) {
	resp := runDnssecKeyImportCreate(t, tftypes.NewValue(tftypes.String, "ecdsa384"), func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			importedKeyResponse(t, w, "ecdsa256", 256)
			return
		}
		w.WriteHeader(http.StatusBadGateway)
	})
	if !resp.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(resp.Diagnostics), "key 7 of zone 10") {
		t.Fatalf("want an error naming the key, got %v", resp.Diagnostics)
	}
	var id types.String
	resp.State.GetAttribute(context.Background(), path.Root("id"), &id)
	if id.ValueString() != "10/7" {
		t.Errorf("state id = %v, want 10/7 so the key stays tracked", id)
	}
}

func TestDnssecKeyResource_ValidatesImportAtPlan(t *testing.T) {
	providerConfig := `
provider "poweradmin" {
  api_url = "http://127.0.0.1:1"
  api_key = "unused"
}
`
	key := fmt.Sprintf("%q", testISCKey)
	for _, tc := range []struct {
		spec string
		err  string
	}{
		{`type = "csk"`, `algorithm is required unless private_key_wo is set`},
		{`type = "csk"` + "\n" + `algorithm = "ecdsa256"`, `bits is required unless private_key_wo is set`},
		{`type = "csk"` + "\n" + `private_key_wo = ` + key, `private_key_wo_version must be set together with private_key_wo`},
		{`type = "csk"` + "\n" + `algorithm = "ecdsa256"` + "\n" + `bits = 256` + "\n" + `private_key_wo_version = 1`, `private_key_wo_version is set without private_key_wo`},
		{`type = "csk"` + "\n" + `private_key_wo = "-----BEGIN PRIVATE KEY-----"` + "\n" + `private_key_wo_version = 1`, `must be an ISC/BIND private key`},
		{`type = "csk"` + "\n" + `algorithm = "rsasha256"` + "\n" + `private_key_wo = ` + key + "\n" + `private_key_wo_version = 1`, `algorithm is "rsasha256" but the private key uses ecdsa256`},
		{`type = "csk"` + "\n" + `bits = 384` + "\n" + `private_key_wo = ` + key + "\n" + `private_key_wo_version = 1`, `bits is 384 but ecdsa256 keys are 256 bits`},
	} {
		resource.UnitTest(t, resource.TestCase{
			TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig + fmt.Sprintf(`
resource "poweradmin_dnssec_key" "test" {
  zone_id = 1
  %s
}
`, tc.spec),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(regexp.QuoteMeta(tc.err)),
				},
			},
		})
	}
}

// TestAccDnssecKeyResource_ImportPrivateKey imports a generated key; it needs
// Poweradmin 4.6.0+ with the PowerDNS API and Terraform/OpenTofu 1.11+.
func TestAccDnssecKeyResource_ImportPrivateKey(t *testing.T) {
	key, pub := testECDSAISCKey(t)
	config := func(version int) string {
		return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name = "test-dnssec-key-import-acc.example.com"
  type = "MASTER"
}

resource "poweradmin_dnssec_key" "test" {
  zone_id                = poweradmin_zone.test.id
  type                   = "csk"
  private_key_wo         = %q
  private_key_wo_version = %d
}
`, key, version)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.test", "algorithm", "ecdsa256"),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.test", "bits", "256"),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.test", "active", "true"),
					resource.TestCheckNoResourceAttr("poweradmin_dnssec_key.test", "private_key_wo"),
					// PowerDNS serves the public half of the key that was sent
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.test", "dnskey", "257 3 13 "+pub),
				),
			},
			// The same version plans no change
			{Config: config(1), PlanOnly: true},
		},
	})
}

// TestAccDnssecKeyResource_ImportSizeMismatchRollsBack imports a 2048-bit RSA key
// configured as 1024 bits: only PowerDNS knows the size, so the key is deleted again.
func TestAccDnssecKeyResource_ImportSizeMismatchRollsBack(t *testing.T) {
	key := testRSAISCKey(t, 2048)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name = "test-dnssec-key-mismatch-acc.example.com"
  type = "MASTER"
}

resource "poweradmin_dnssec_key" "test" {
  zone_id                = poweradmin_zone.test.id
  type                   = "csk"
  bits                   = 1024
  private_key_wo         = %q
  private_key_wo_version = 1
}

data "poweradmin_dnssec_keys" "after" {
  zone_id = poweradmin_zone.test.id
}
`, key),
				ExpectError: regexp.MustCompile(`bits is 1024 but PowerDNS reports the imported key as 2048 bits;\s+the\s+imported\s+key\s+was\s+deleted\s+again`),
			},
			{
				// The zone is left with no keys, so nothing was orphaned
				Config: testAccProviderConfig() + `
resource "poweradmin_zone" "test" {
  name = "test-dnssec-key-mismatch-acc.example.com"
  type = "MASTER"
}

data "poweradmin_dnssec_keys" "after" {
  zone_id = poweradmin_zone.test.id
}
`,
				Check: resource.TestCheckResourceAttr("data.poweradmin_dnssec_keys.after", "keys.#", "0"),
			},
		},
	})
}
