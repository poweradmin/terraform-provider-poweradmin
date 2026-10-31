// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import "testing"

func TestParseImportIDPair(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		parent  int64
		child   int64
		wantErr bool
	}{
		{"valid pair", "123/456", 123, 456, false},
		{"trailing garbage rejected", "123/456xyz", 0, 0, true},
		{"negative id rejected", "123/-456", 0, 0, true},
		{"missing separator", "123", 0, 0, true},
		{"empty parts", "/", 0, 0, true},
		{"extra separator rejected", "1/2/3", 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, child, err := parseImportIDPair(tt.id, "a/b")
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseImportIDPair(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
			if parent != tt.parent || child != tt.child {
				t.Errorf("parseImportIDPair(%q) = %d, %d; want %d, %d", tt.id, parent, child, tt.parent, tt.child)
			}
		})
	}
}

func TestParseZoneMetadataImportID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		zone    int64
		kind    string
		wantErr bool
	}{
		{"valid", "10/ALLOW-AXFR-FROM", 10, "ALLOW-AXFR-FROM", false},
		{"kind kept as given", "10/also-notify", 10, "also-notify", false},
		{"missing kind", "10/", 0, "", true},
		{"missing separator", "10", 0, "", true},
		{"non-numeric zone", "abc/SOA-EDIT", 0, "", true},
		{"negative zone", "-1/SOA-EDIT", 0, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone, kind, err := parseZoneMetadataImportID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseZoneMetadataImportID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
			if zone != tt.zone || kind != tt.kind {
				t.Errorf("parseZoneMetadataImportID(%q) = %d, %q; want %d, %q", tt.id, zone, kind, tt.zone, tt.kind)
			}
		})
	}
}
