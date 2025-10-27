// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

// Zone represents a DNS zone in Poweradmin
type Zone struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name"`
	Type        string `json:"type"`         // MASTER, SLAVE, NATIVE
	Masters     string `json:"masters,omitempty"` // For SLAVE zones
	Account     string `json:"account,omitempty"`
	Description string `json:"description,omitempty"`
	SOASerial   int    `json:"soa_serial,omitempty"`
	DNSSECSigned bool  `json:"dnssec_signed,omitempty"`
}

// ZoneListResponse represents the response from listing zones
type ZoneListResponse struct {
	Zones      []Zone      `json:"zones"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// ZoneResponse represents the response for a single zone
type ZoneResponse struct {
	Zone Zone `json:"zone"`
}

// CreateZoneRequest represents the request to create a zone
type CreateZoneRequest struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Masters     string `json:"masters,omitempty"`
	Account     string `json:"account,omitempty"`
	Description string `json:"description,omitempty"`
	Template    string `json:"template,omitempty"`
}

// UpdateZoneRequest represents the request to update a zone
// Using pointers allows us to distinguish between "not set" (nil) and "set to empty" ("")
type UpdateZoneRequest struct {
	Type        *string `json:"type,omitempty"`
	Masters     *string `json:"masters,omitempty"`
	Account     *string `json:"account,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Record represents a DNS record in Poweradmin
type Record struct {
	ID       int    `json:"id,omitempty"`
	ZoneID   int    `json:"zone_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`     // A, AAAA, CNAME, MX, TXT, etc.
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"` // For MX, SRV records
	Disabled bool   `json:"disabled"`
}

// RecordListResponse represents the response from listing records
type RecordListResponse struct {
	Records    []Record    `json:"records"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// RecordResponse represents the response for a single record
type RecordResponse struct {
	Record Record `json:"record"`
}

// CreateRecordRequest represents the request to create a record
type CreateRecordRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

// UpdateRecordRequest represents the request to update a record
// Using pointers for TTL and Priority allows us to distinguish between "not set" (nil) and "set to zero" (0)
type UpdateRecordRequest struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Content  string `json:"content,omitempty"`
	TTL      *int   `json:"ttl,omitempty"`
	Priority *int   `json:"priority,omitempty"`
	Disabled *bool  `json:"disabled,omitempty"`
}
