// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"testing"
)

func TestGetGroup(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/groups/1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		respondJSON(t, w, GroupResponse{Group: Group{ID: 1, Name: "admins", Description: "Admin group", PermTemplID: 6}})
	})

	group, err := client.GetGroup(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.Name != "admins" {
		t.Errorf("expected name 'admins', got '%s'", group.Name)
	}
	if group.PermTemplID != 6 {
		t.Errorf("expected PermTemplID 6, got %d", group.PermTemplID)
	}
}

func TestListGroups(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, GroupListResponse{
			Groups: []Group{
				{ID: 1, Name: "admins"},
				{ID: 2, Name: "operators"},
			},
		})
	})

	groups, err := client.ListGroups(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestCreateGroup(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		respondJSON(t, w, CreateGroupResponse{Group: struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}{ID: 5, Name: "new-group"}})
	})

	id, err := client.CreateGroup(context.Background(), CreateGroupRequest{
		Name:        "new-group",
		PermTemplID: 6,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 5 {
		t.Errorf("expected group ID 5, got %d", id)
	}
}

func TestUpdateGroup(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v2/groups/1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, GroupResponse{Group: Group{ID: 1, Name: "updated-admins", Description: "Updated"}})
	})

	group, err := client.UpdateGroup(context.Background(), 1, UpdateGroupRequest{Name: "updated-admins"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.Name != "updated-admins" {
		t.Errorf("expected name 'updated-admins', got '%s'", group.Name)
	}
}

func TestDeleteGroup(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/groups/1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := client.DeleteGroup(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindGroupByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, GroupListResponse{
			Groups: []Group{
				{ID: 1, Name: "admins"},
				{ID: 2, Name: "operators"},
			},
		})
	})

	group, err := client.FindGroupByName(context.Background(), "operators")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.ID != 2 {
		t.Errorf("expected group ID 2, got %d", group.ID)
	}
}

func TestFindGroupByName_NotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, GroupListResponse{Groups: []Group{{ID: 1, Name: "admins"}}})
	})

	_, err := client.FindGroupByName(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing group")
	}
}

func TestAddGroupMember(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/groups/1/members" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, nil)
	})

	err := client.AddGroupMember(context.Background(), 1, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveGroupMember(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/groups/1/members/5" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := client.RemoveGroupMember(context.Background(), 1, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListGroupMembers(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/groups/1/members" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		respondJSON(t, w, GroupMemberListResponse{
			Members: []GroupMember{
				{UserID: 1, Username: "admin"},
				{UserID: 2, Username: "user1"},
			},
		})
	})

	members, err := client.ListGroupMembers(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

func TestAssignZoneToGroup(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/groups/1/zones" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, nil)
	})

	err := client.AssignZoneToGroup(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnassignZoneFromGroup(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/groups/1/zones/10" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := client.UnassignZoneFromGroup(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListGroupZones(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/groups/1/zones" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		respondJSON(t, w, GroupZoneListResponse{
			Zones: []GroupZone{
				{ZoneID: 10, ZoneName: "example.com", ZoneType: "MASTER"},
			},
		})
	})

	zones, err := client.ListGroupZones(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(zones) != 1 {
		t.Errorf("expected 1 zone, got %d", len(zones))
	}
}
