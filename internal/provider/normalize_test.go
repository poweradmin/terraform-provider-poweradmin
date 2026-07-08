// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import "testing"

func TestNormalizeRecordName(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		fromAPI    string
		zoneName   string
		want       string
	}{
		{"identical relative", "www", "www", "example.com", "www"},
		{"identical apex", "@", "@", "example.com", "@"},
		{"fqdn preserved", "www.example.com", "www", "example.com", "www.example.com"},
		{"fqdn with trailing dot preserved", "www.example.com.", "www", "example.com", "www.example.com."},
		{"fqdn case-insensitive match", "WWW.Example.COM", "www", "example.com", "WWW.Example.COM"},
		{"multi-label fqdn preserved", "sub.www.example.com", "sub.www", "example.com", "sub.www.example.com"},
		{"zone fqdn preserved for apex", "example.com", "@", "example.com", "example.com"},
		{"zone fqdn with dot preserved for apex", "example.com.", "@", "example.com", "example.com."},
		{"external rename surfaces", "www", "mail", "example.com", "mail"},
		{"external rename to apex surfaces", "www", "@", "example.com", "@"},
		{"relative multi-label drift surfaces", "sub.www", "sub", "example.com", "sub"},
		{"foreign fqdn not preserved for apex", "other.org", "@", "example.com", "@"},
		{"similar prefix is not a match", "wwwx.example.com", "www", "example.com", "www"},
		{"unknown zone name trusts configured", "www.example.com", "www", "", "www.example.com"},
		{"unknown zone name keeps identical value", "www", "www", "", "www"},
		{"unknown zone name with empty configured takes api value", "", "www", "", "www"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeRecordName(tt.configured, tt.fromAPI, tt.zoneName); got != tt.want {
				t.Errorf("normalizeRecordName(%q, %q, %q) = %q, want %q", tt.configured, tt.fromAPI, tt.zoneName, got, tt.want)
			}
		})
	}
}

func TestNormalizeTypeCase(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		fromAPI    string
		want       string
	}{
		{"lowercase preserved", "master", "MASTER", "master"},
		{"exact match", "MASTER", "MASTER", "MASTER"},
		{"mixed case preserved", "CnAmE", "CNAME", "CnAmE"},
		{"real change surfaces", "MASTER", "SLAVE", "SLAVE"},
		{"empty configured takes api value", "", "A", "A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTypeCase(tt.configured, tt.fromAPI); got != tt.want {
				t.Errorf("normalizeTypeCase(%q, %q) = %q, want %q", tt.configured, tt.fromAPI, got, tt.want)
			}
		})
	}
}

func TestNormalizeRecordContent(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		fromAPI    string
		want       string
	}{
		{"trailing dot preserved", "mail.example.com.", "mail.example.com", "mail.example.com."},
		{"identical values", "mail.example.com", "mail.example.com", "mail.example.com"},
		{"api keeps dot", "mail.example.com.", "mail.example.com.", "mail.example.com."},
		{"external change surfaces", "mail.example.com.", "other.example.com", "other.example.com"},
		{"empty configured takes api value", "", "mail.example.com", "mail.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeRecordContent(tt.configured, tt.fromAPI); got != tt.want {
				t.Errorf("normalizeRecordContent(%q, %q) = %q, want %q", tt.configured, tt.fromAPI, got, tt.want)
			}
		})
	}
}
