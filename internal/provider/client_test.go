// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// newTestClient creates a Client backed by a test HTTP server.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		APIVersion: "v2",
		APIKey:     "test-key",
	}
}

// respondJSON writes a standard Poweradmin API JSON response.
func respondJSON(t *testing.T, w http.ResponseWriter, data interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := APIResponse{Success: true}
	if data != nil {
		raw, _ := json.Marshal(data)
		resp.Data = raw
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

// respondError writes a standard Poweradmin API error response.
func respondError(t *testing.T, w http.ResponseWriter, statusCode int, message string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := APIResponse{
		Success: false,
		Error:   &APIError{Code: statusCode, Message: message},
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

// Redirects must fail loudly: followed 301/302/303 would rewrite writes into
// GETs that read as success while creating/deleting nothing.
func TestNewClient_RefusesRedirects(t *testing.T) {
	backendHit := false
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		backendHit = true
		respondJSON(t, w, map[string]interface{}{"records": []Record{}})
	}))
	t.Cleanup(backend.Close)

	front := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, backend.URL+r.URL.RequestURI(), http.StatusMovedPermanently)
	}))
	t.Cleanup(front.Close)

	client, err := NewClient(&PoweradminProviderModel{
		ApiUrl: types.StringValue(front.URL),
		ApiKey: types.StringValue("test-key"),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	ctx := context.Background()

	calls := map[string]func() error{
		"Get":    func() error { var r RecordListResponse; return client.Get(ctx, "zones/1/records", &r) },
		"Post":   func() error { return client.Post(ctx, "zones/1/records", map[string]string{}, nil) },
		"Put":    func() error { return client.Put(ctx, "zones/1/records/2", map[string]string{}, nil) },
		"Delete": func() error { return client.Delete(ctx, "zones/1/records/2") },
	}
	for name, call := range calls {
		err := call()
		var apiErr *apiHTTPError
		if err == nil || !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusMovedPermanently {
			t.Errorf("%s through redirect: want HTTP 301 apiHTTPError, got %v", name, err)
		}
	}
	if backendHit {
		t.Error("redirect target was contacted; redirects must not be followed")
	}
}

// --- Zone tests ---

func TestGetZone(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/zones/1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, ZoneResponse{
			Zone: Zone{ID: 1, Name: "example.com", Type: "MASTER"},
		})
	})

	zone, err := client.GetZone(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if zone.Name != "example.com" {
		t.Errorf("expected zone name 'example.com', got '%s'", zone.Name)
	}
	if zone.Type != "MASTER" {
		t.Errorf("expected zone type 'MASTER', got '%s'", zone.Type)
	}
}

func TestListZones(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, ZoneListResponse{
			Zones: []Zone{
				{ID: 1, Name: "example.com", Type: "MASTER"},
				{ID: 2, Name: "example.org", Type: "NATIVE"},
			},
		})
	})

	zones, err := client.ListZones(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(zones) != 2 {
		t.Errorf("expected 2 zones, got %d", len(zones))
	}
}

func TestCreateZone(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		respondJSON(t, w, CreateZoneResponse{ZoneID: 42})
	})

	id, err := client.CreateZone(context.Background(), CreateZoneRequest{
		Name: "new.example.com",
		Type: "MASTER",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("expected zone ID 42, got %d", id)
	}
}

func TestUpdateZone(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v2/zones/1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, ZoneResponse{
			Zone: Zone{ID: 1, Name: "example.com", Type: "NATIVE"},
		})
	})

	newType := "NATIVE"
	zone, err := client.UpdateZone(context.Background(), 1, UpdateZoneRequest{Type: &newType})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if zone.Type != "NATIVE" {
		t.Errorf("expected type 'NATIVE', got '%s'", zone.Type)
	}
}

func TestDeleteZone(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/zones/1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := client.DeleteZone(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindZoneByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, ZoneListResponse{
			Zones: []Zone{
				{ID: 1, Name: "example.com", Type: "MASTER"},
				{ID: 2, Name: "example.org", Type: "NATIVE"},
			},
		})
	})

	zone, err := client.FindZoneByName(context.Background(), "example.org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if zone.ID != 2 {
		t.Errorf("expected zone ID 2, got %d", zone.ID)
	}
}

func TestFindZoneByName_NotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, ZoneListResponse{
			Zones: []Zone{{ID: 1, Name: "example.com", Type: "MASTER"}},
		})
	})

	_, err := client.FindZoneByName(context.Background(), "notfound.com")
	if err == nil {
		t.Fatal("expected error for missing zone")
	}
}

// --- Record tests ---

// PowerDNS API backend record IDs are encoded strings: they must decode from
// both JSON numbers and strings, and be path-escaped in record URLs.
func TestRecordID_StringBackend(t *testing.T) {
	var record Record
	if err := json.Unmarshal([]byte(`{"id":"ZXhhbXBsZS5jb20vVFhU","name":"www","type":"A"}`), &record); err != nil {
		t.Fatalf("unmarshal string ID: %v", err)
	}
	if record.ID != "ZXhhbXBsZS5jb20vVFhU" {
		t.Errorf("expected string ID preserved, got %q", record.ID)
	}
	if err := json.Unmarshal([]byte(`{"id":42,"name":"www","type":"A"}`), &record); err != nil {
		t.Fatalf("unmarshal numeric ID: %v", err)
	}
	if record.ID != "42" {
		t.Errorf("expected numeric ID as string, got %q", record.ID)
	}

	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	})
	if err := client.DeleteRecord(context.Background(), 1, "abc/def=123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "/api/v2/zones/1/records/abc%2Fdef=123"; gotPath != want {
		t.Errorf("delete path = %s, want %s", gotPath, want)
	}
}

func TestGetRecord(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/zones/1/records/10" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		respondJSON(t, w, RecordResponse{
			Record: Record{ID: "10", ZoneID: 1, Name: "www.example.com", Type: "A", Content: "192.0.2.1", TTL: 3600},
		})
	})

	record, err := client.GetRecord(context.Background(), 1, "10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.Content != "192.0.2.1" {
		t.Errorf("expected content '192.0.2.1', got '%s'", record.Content)
	}
}

func TestListRecords(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, RecordListResponse{
			Records: []Record{
				{ID: "10", Name: "www.example.com", Type: "A", Content: "192.0.2.1"},
				{ID: "11", Name: "mail.example.com", Type: "MX", Content: "mail.example.com", Priority: 10},
			},
		})
	})

	records, err := client.ListRecords(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

func TestListRecords_WithTypeFilter(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "A" {
			t.Errorf("expected type=A query param, got '%s'", r.URL.Query().Get("type"))
		}
		respondJSON(t, w, RecordListResponse{Records: []Record{{ID: "10", Name: "www.example.com", Type: "A", Content: "192.0.2.1"}}})
	})

	records, err := client.ListRecords(context.Background(), 1, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

func TestCreateRecord(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		respondJSON(t, w, RecordResponse{
			Record: Record{ID: "20", ZoneID: 1, Name: "new.example.com", Type: "A", Content: "192.0.2.2", TTL: 3600},
		})
	})

	record, err := client.CreateRecord(context.Background(), 1, CreateRecordRequest{
		Name: "new.example.com", Type: "A", Content: "192.0.2.2", TTL: 3600,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.ID != "20" {
		t.Errorf("expected record ID 20, got %s", record.ID)
	}
}

func TestDeleteRecord(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := client.DeleteRecord(context.Background(), 1, "10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- User tests ---

func TestGetUser(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/users/5" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		respondJSON(t, w, UserResponse{
			User: User{UserID: 5, Username: "admin", Fullname: "Admin User", Email: "admin@example.com", Active: true},
		})
	})

	user, err := client.GetUser(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "admin" {
		t.Errorf("expected username 'admin', got '%s'", user.Username)
	}
}

func TestListUsers(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, UserListResponse{
			Users: []User{
				{UserID: 1, Username: "admin"},
				{UserID: 2, Username: "user1"},
			},
		})
	})

	users, err := client.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestCreateUser(t *testing.T) {
	callCount := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST for create, got %s", r.Method)
			}
			respondJSON(t, w, CreateUserResponse{UserID: 10})
		} else {
			respondJSON(t, w, UserResponse{
				User: User{UserID: 10, Username: "newuser", Email: "new@example.com", Active: true},
			})
		}
	})

	user, err := client.CreateUser(context.Background(), CreateUserRequest{
		Username: "newuser", Password: "secret", Email: "new@example.com", Active: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.UserID != 10 {
		t.Errorf("expected user ID 10, got %d", user.UserID)
	}
}

func TestDeleteUser(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := client.DeleteUser(context.Background(), 5, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteUser_WithTransfer(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		var body map[string]int
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body["transfer_to_user_id"] != 1 {
			t.Errorf("expected transfer_to_user_id=1, got %d", body["transfer_to_user_id"])
		}
		w.WriteHeader(http.StatusNoContent)
	})

	transferTo := 1
	err := client.DeleteUser(context.Background(), 5, &transferTo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindUserByUsername(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, UserListResponse{
			Users: []User{
				{UserID: 1, Username: "admin"},
				{UserID: 2, Username: "user1"},
			},
		})
	})

	user, err := client.FindUserByUsername(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.UserID != 2 {
		t.Errorf("expected user ID 2, got %d", user.UserID)
	}
}

func TestFindUserByUsername_NotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, UserListResponse{
			Users: []User{
				{UserID: 1, Username: "admin"},
			},
		})
	})

	_, err := client.FindUserByUsername(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

// --- RRSet tests ---

func TestListRRSets(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, RRSetListResponse{
			RRSets: []RRSet{
				{Name: "example.com", Type: "A", TTL: 3600, Records: []RRSetRecord{{Content: "192.0.2.1"}}},
			},
		})
	})

	rrsets, err := client.ListRRSets(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rrsets) != 1 {
		t.Errorf("expected 1 rrset, got %d", len(rrsets))
	}
}

func TestGetRRSet(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/zones/1/rrsets/www.example.com/A" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		respondJSON(t, w, RRSetResponse{RRSet: RRSet{Name: "www.example.com", Type: "A", TTL: 3600, Records: []RRSetRecord{{Content: "192.0.2.1"}}}})
	})

	rrset, err := client.GetRRSet(context.Background(), 1, "www.example.com", "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rrset.Name != "www.example.com" {
		t.Errorf("expected name 'www.example.com', got '%s'", rrset.Name)
	}
}

func TestDeleteRRSet(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := client.DeleteRRSet(context.Background(), 1, "www.example.com", "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Permission tests ---

func TestGetPermission(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/permissions/1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		respondJSON(t, w, PermissionResponse{
			Permission: Permission{ID: 1, Name: "zone_master_add", Descr: "Add master zones"},
		})
	})

	perm, err := client.GetPermission(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if perm.Name != "zone_master_add" {
		t.Errorf("expected name 'zone_master_add', got '%s'", perm.Name)
	}
}

func TestListPermissions(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, PermissionListResponse{
			Permissions: []Permission{
				{ID: 1, Name: "zone_master_add"},
				{ID: 2, Name: "zone_slave_add"},
			},
		})
	})

	perms, err := client.ListPermissions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(perms))
	}
}

func TestFindPermissionByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, PermissionListResponse{
			Permissions: []Permission{
				{ID: 1, Name: "zone_master_add"},
				{ID: 2, Name: "zone_slave_add"},
			},
		})
	})

	perm, err := client.FindPermissionByName(context.Background(), "zone_slave_add")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if perm.ID != 2 {
		t.Errorf("expected permission ID 2, got %d", perm.ID)
	}
}

// --- Error handling tests ---

func TestAPIError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondError(t, w, http.StatusNotFound, "Zone not found")
	})

	_, err := client.GetZone(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected IsNotFoundError to return true, got false for: %v", err)
	}
}

// Locks in the wire shape for clearing semantics: an empty description is
// sent (clears server-side), an unset perm_templ is omitted (server keeps it).
func TestUpdateUserRequestJSON(t *testing.T) {
	empty := ""
	body, err := json.Marshal(UpdateUserRequest{Username: "u", Description: &empty})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(body), `"description":""`) {
		t.Errorf("expected empty description to be sent, got %s", body)
	}
	if strings.Contains(string(body), "perm_templ") {
		t.Errorf("expected unset perm_templ to be omitted, got %s", body)
	}
}

// TestIsNotFoundError locks in typed 404 detection: only *apiHTTPError with
// status 404 matches, never error strings that merely contain "404" (URLs with
// port 4041, zone ID 404, response bodies mentioning 404, ...).
func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"api 404", &apiHTTPError{StatusCode: 404, Message: "User not found"}, true},
		{"api conflict", &apiHTTPError{StatusCode: 409, Message: "Domain already exists"}, false},
		{"api server error", &apiHTTPError{StatusCode: 500, Message: "Failed to create user"}, false},
		{"api bad request", &apiHTTPError{StatusCode: 400, Message: "Invalid input"}, false},
		{"wrapped api 404", fmt.Errorf("could not read zone: %w", &apiHTTPError{StatusCode: 404, Message: "Zone not found"}), true},
		{"string with 404 is not matched", errors.New("API error (HTTP 404): User not found"), false},
		{"network error with 404 in url", errors.New(`request failed: Get "http://dns:4041/api/v2/zones/404": connection refused`), false},
		{"server error mentioning 404", &apiHTTPError{StatusCode: 500, Message: "upstream returned 404"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.want {
				t.Errorf("IsNotFoundError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestAuthHeaders_APIKey(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer auth header, got '%s'", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("expected X-API-Key header, got '%s'", r.Header.Get("X-API-Key"))
		}
		respondJSON(t, w, ZoneListResponse{Zones: []Zone{}})
	})

	_, err := client.ListZones(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthHeaders_BasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "admin" || password != "secret" {
			t.Errorf("expected basic auth admin:secret, got %s:%s (ok=%v)", username, password, ok)
		}
		respondJSON(t, w, ZoneListResponse{Zones: []Zone{}})
	}))
	t.Cleanup(server.Close)

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		APIVersion: "v2",
		Username:   "admin",
		Password:   "secret",
	}

	_, err := client.ListZones(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
