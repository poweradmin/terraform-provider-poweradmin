// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"testing"
)

func TestListZoneTemplates(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/zone-templates" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, []ZoneTemplate{
			{ID: 1, Name: "Default", Description: "Default template", Owner: 1, IsGlobal: false, ZonesLinked: 2},
			{ID: 2, Name: "Global", Description: "Global template", Owner: 0, IsGlobal: true, ZonesLinked: 5},
		})
	})

	templates, err := client.ListZoneTemplates(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
	if templates[1].IsGlobal != true {
		t.Errorf("expected second template to be global")
	}
}

func TestGetZoneTemplate(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/zone-templates/3" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		respondJSON(t, w, ZoneTemplate{
			ID:          3,
			Name:        "Hosting",
			Description: "Standard hosting template",
			Owner:       1,
			IsGlobal:    false,
			Records: []ZoneTemplateRecord{
				{ID: 10, Name: "[ZONE]", Type: "SOA", Content: "[NS1] [HOSTMASTER] [SERIAL] 28800 7200 604800 86400", TTL: 86400},
				{ID: 11, Name: "[ZONE]", Type: "NS", Content: "[NS1]", TTL: 86400},
			},
		})
	})

	template, err := client.GetZoneTemplate(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if template.Name != "Hosting" {
		t.Errorf("expected name 'Hosting', got '%s'", template.Name)
	}
	if len(template.Records) != 2 {
		t.Errorf("expected 2 records, got %d", len(template.Records))
	}
	if template.Records[0].Type != "SOA" {
		t.Errorf("expected first record SOA, got '%s'", template.Records[0].Type)
	}
}

func TestCreateZoneTemplate(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/zone-templates" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, createResponseID{ID: 42})
	})

	id, err := client.CreateZoneTemplate(context.Background(), CreateZoneTemplateRequest{
		Name:        "New",
		Description: "New template",
		IsGlobal:    false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("expected template ID 42, got %d", id)
	}
}

func TestUpdateZoneTemplate(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v2/zone-templates/7" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, nil)
	})

	err := client.UpdateZoneTemplate(context.Background(), 7, UpdateZoneTemplateRequest{
		Name:        "Renamed",
		Description: "Updated",
		IsGlobal:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteZoneTemplate(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/zone-templates/9" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := client.DeleteZoneTemplate(context.Background(), 9); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindZoneTemplateByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, []ZoneTemplate{
			{ID: 1, Name: "Alpha"},
			{ID: 2, Name: "Beta"},
		})
	})

	template, err := client.FindZoneTemplateByName(context.Background(), "Beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if template.ID != 2 {
		t.Errorf("expected template ID 2, got %d", template.ID)
	}
}

func TestFindZoneTemplateByName_NotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, []ZoneTemplate{{ID: 1, Name: "Alpha"}})
	})

	_, err := client.FindZoneTemplateByName(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestListZoneTemplateRecords(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/zone-templates/3/records" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		respondJSON(t, w, []ZoneTemplateRecord{
			{ID: 10, Name: "[ZONE]", Type: "NS", Content: "[NS1]", TTL: 86400},
			{ID: 11, Name: "www.[ZONE]", Type: "A", Content: "192.0.2.1", TTL: 3600},
		})
	})

	records, err := client.ListZoneTemplateRecords(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

func TestGetZoneTemplateRecord(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/zone-templates/3/records/10" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		respondJSON(t, w, ZoneTemplateRecord{
			ID: 10, Name: "[ZONE]", Type: "NS", Content: "[NS1]", TTL: 86400,
		})
	})

	record, err := client.GetZoneTemplateRecord(context.Background(), 3, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.Type != "NS" {
		t.Errorf("expected type NS, got '%s'", record.Type)
	}
}

func TestCreateZoneTemplateRecord(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/zone-templates/3/records" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, createResponseID{ID: 22})
	})

	id, err := client.CreateZoneTemplateRecord(context.Background(), 3, CreateZoneTemplateRecordRequest{
		Name: "www.[ZONE]", Type: "A", Content: "192.0.2.1", TTL: 3600,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 22 {
		t.Errorf("expected record ID 22, got %d", id)
	}
}

func TestUpdateZoneTemplateRecord(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v2/zone-templates/3/records/22" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, nil)
	})

	ttl := 7200
	err := client.UpdateZoneTemplateRecord(context.Background(), 3, 22, UpdateZoneTemplateRecordRequest{
		Name: "www.[ZONE]", Type: "A", Content: "192.0.2.2", TTL: &ttl,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteZoneTemplateRecord(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/zone-templates/3/records/22" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := client.DeleteZoneTemplateRecord(context.Background(), 3, 22); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
