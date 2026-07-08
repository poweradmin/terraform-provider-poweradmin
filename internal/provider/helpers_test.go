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
