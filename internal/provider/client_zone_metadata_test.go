// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestListZoneMetadata(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/zones/7/metadata" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, ZoneMetadataListResponse{Metadata: []ZoneMetadata{
			{Kind: "ALLOW-AXFR-FROM", Values: []string{"192.0.2.10", "AUTO-NS"}},
			{Kind: "SOA-EDIT", Values: []string{"INCEPTION-INCREMENT"}},
		}})
	})

	entries, err := client.ListZoneMetadata(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 || entries[0].Kind != "ALLOW-AXFR-FROM" || len(entries[0].Values) != 2 || entries[1].Values[0] != "INCEPTION-INCREMENT" {
		t.Errorf("metadata envelope decoded wrong: %+v", entries)
	}
}

func TestListZoneMetadata_Empty(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondJSON(t, w, map[string]interface{}{"metadata": []interface{}{}})
	})

	entries, err := client.ListZoneMetadata(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no entries, got %+v", entries)
	}
}

func TestGetZoneMetadata(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/zones/7/metadata/ALLOW-AXFR-FROM" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, ZoneMetadata{Kind: "ALLOW-AXFR-FROM", Values: []string{"192.0.2.10"}})
	})

	meta, err := client.GetZoneMetadata(context.Background(), 7, "ALLOW-AXFR-FROM")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Kind != "ALLOW-AXFR-FROM" || len(meta.Values) != 1 || meta.Values[0] != "192.0.2.10" {
		t.Errorf("unexpected metadata: %+v", meta)
	}
}

// A 404 means the zone is gone or the kind has no values; any other error must
// not be read as "gone", or Terraform would drop the resource from state.
func TestGetZoneMetadata_NotFoundVersusServerError(t *testing.T) {
	for _, tc := range []struct {
		status   int
		notFound bool
	}{
		{http.StatusNotFound, true},
		{http.StatusInternalServerError, false},
	} {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			respondError(t, w, tc.status, "Metadata kind not found")
		})

		_, err := client.GetZoneMetadata(context.Background(), 7, "ALSO-NOTIFY")
		if err == nil {
			t.Fatalf("HTTP %d: expected an error", tc.status)
		}
		if IsNotFoundError(err) != tc.notFound {
			t.Errorf("HTTP %d: IsNotFoundError = %t, want %t", tc.status, IsNotFoundError(err), tc.notFound)
		}
	}
}

func TestSetZoneMetadata_UsesPut(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v2/zones/7/metadata/ALLOW-AXFR-FROM" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if strings.TrimSpace(string(body)) != `{"values":["192.0.2.10","AUTO-NS"]}` {
			t.Errorf("unexpected body: %s", body)
		}
		respondJSON(t, w, nil)
	})

	if err := client.SetZoneMetadata(context.Background(), 7, "ALLOW-AXFR-FROM", []string{"192.0.2.10", "AUTO-NS"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Refusals (read-only kind, wrong value) must reach the user with the server's wording.
func TestSetZoneMetadata_Refused(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respondError(t, w, http.StatusForbidden, "Metadata kind PRESIGNED is maintained by PowerDNS")
	})

	err := client.SetZoneMetadata(context.Background(), 7, "PRESIGNED", []string{"1"})
	if err == nil || !strings.Contains(err.Error(), "maintained by PowerDNS") {
		t.Fatalf("expected the server's refusal, got: %v", err)
	}
}

// Poweradmin's SQL backend rewrites all kinds of a zone on every write, so writes
// to one zone must never overlap.
func TestSetZoneMetadata_SerializesWritesPerZone(t *testing.T) {
	var inFlight, maxInFlight atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := inFlight.Add(1)
		for {
			m := maxInFlight.Load()
			if n <= m || maxInFlight.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		inFlight.Add(-1)
		respondJSON(t, w, nil)
	})

	var wg sync.WaitGroup
	for _, kind := range []string{"ALLOW-AXFR-FROM", "ALSO-NOTIFY", "SOA-EDIT"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.SetZoneMetadata(context.Background(), 7, kind, []string{"x"}); err != nil {
				t.Errorf("%s: unexpected error: %v", kind, err)
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := client.DeleteZoneMetadata(context.Background(), 7, "X-OTHER"); err != nil {
			t.Errorf("delete: unexpected error: %v", err)
		}
	}()
	wg.Wait()

	if got := maxInFlight.Load(); got != 1 {
		t.Errorf("writes to one zone overlapped: %d in flight at once", got)
	}
}

func TestDeleteZoneMetadata(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/zones/7/metadata/ALSO-NOTIFY" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		respondJSON(t, w, nil)
	})

	if err := client.DeleteZoneMetadata(context.Background(), 7, "ALSO-NOTIFY"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// A kind is user input in the URL path, so it is escaped rather than able to change the route.
func TestZoneMetadataPath_EscapesKind(t *testing.T) {
	if got := zoneMetadataPath(7, "X-A/B"); got != "zones/7/metadata/X-A%2FB" {
		t.Errorf("got %q", got)
	}
}
