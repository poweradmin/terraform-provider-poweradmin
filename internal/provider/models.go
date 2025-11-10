// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

// Zone represents a DNS zone in Poweradmin.
type Zone struct {
	ID           int    `json:"id,omitempty"`
	Name         string `json:"name"`
	Type         string `json:"type"`              // MASTER, SLAVE, NATIVE
	Masters      string `json:"masters,omitempty"` // For SLAVE zones
	Account      string `json:"account,omitempty"`
	Description  string `json:"description,omitempty"`
	SOASerial    int    `json:"soa_serial,omitempty"`
	DNSSECSigned bool   `json:"dnssec_signed,omitempty"`
}

// ZoneListResponse represents the response from listing zones.
type ZoneListResponse struct {
	Zones      []Zone      `json:"zones"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// ZoneResponse represents the response for a single zone.
type ZoneResponse struct {
	Zone Zone `json:"zone"`
}

// CreateZoneResponse represents the response from creating a zone.
type CreateZoneResponse struct {
	ZoneID int `json:"zone_id"`
}

// CreateZoneRequest represents the request to create a zone.
type CreateZoneRequest struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Masters     string `json:"masters,omitempty"`
	Account     string `json:"account,omitempty"`
	Description string `json:"description,omitempty"`
	Template    string `json:"template,omitempty"`
}

// UpdateZoneRequest represents the request to update a zone.
// Using pointers allows us to distinguish between "not set" (nil) and "set to empty" ("").
type UpdateZoneRequest struct {
	Type        *string `json:"type,omitempty"`
	Masters     *string `json:"masters,omitempty"`
	Account     *string `json:"account,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Record represents a DNS record in Poweradmin.
type Record struct {
	ID        int    `json:"id,omitempty"`
	ZoneID    int    `json:"zone_id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // A, AAAA, CNAME, MX, TXT, etc.
	Content   string `json:"content"`
	TTL       int    `json:"ttl"`
	Priority  int    `json:"priority,omitempty"` // For MX, SRV records
	Disabled  bool   `json:"disabled"`
	CreatePTR bool   `json:"create_ptr,omitempty"`
}

// RecordListResponse represents the response from listing records.
type RecordListResponse struct {
	Records    []Record    `json:"records"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// RecordResponse represents the response for a single record.
type RecordResponse struct {
	Record Record `json:"record"`
}

// CreateRecordRequest represents the request to create a record.
type CreateRecordRequest struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	TTL       int    `json:"ttl"`
	Priority  int    `json:"priority,omitempty"`
	Disabled  bool   `json:"disabled,omitempty"`
	CreatePTR bool   `json:"create_ptr,omitempty"`
}

// UpdateRecordRequest represents the request to update a record.
// Using pointers for TTL and Priority allows us to distinguish between "not set" (nil) and "set to zero" (0).
type UpdateRecordRequest struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Content  string `json:"content,omitempty"`
	TTL      *int   `json:"ttl,omitempty"`
	Priority *int   `json:"priority,omitempty"`
	Disabled *bool  `json:"disabled,omitempty"`
}

// User represents a user in Poweradmin.
type User struct {
	UserID      int      `json:"user_id,omitempty"`
	Username    string   `json:"username"`
	Fullname    string   `json:"fullname"`
	Email       string   `json:"email"`
	Description string   `json:"description,omitempty"`
	Active      bool     `json:"active"`
	PermTempl   int      `json:"perm_templ,omitempty"`
	UseLdap     bool     `json:"use_ldap,omitempty"`
	IsAdmin     bool     `json:"is_admin,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	ZoneCount   int      `json:"zone_count,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

// UserListResponse represents the response from listing users.
type UserListResponse struct {
	Users      []User      `json:"users"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// UserResponse represents the response for a single user.
type UserResponse struct {
	User User `json:"user"`
}

// CreateUserResponse represents the response from creating a user.
type CreateUserResponse struct {
	UserID int `json:"user_id"`
}

// CreateUserRequest represents the request to create a user.
type CreateUserRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Fullname    string `json:"fullname"`
	Email       string `json:"email"`
	Description string `json:"description,omitempty"`
	Active      bool   `json:"active"`
	PermTempl   int    `json:"perm_templ,omitempty"`
	UseLdap     bool   `json:"use_ldap,omitempty"`
}

// UpdateUserRequest represents the request to update a user.
type UpdateUserRequest struct {
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	Fullname    string `json:"fullname,omitempty"`
	Email       string `json:"email,omitempty"`
	Description string `json:"description,omitempty"`
	Active      bool   `json:"active,omitempty"`
	PermTempl   int    `json:"perm_templ,omitempty"`
	UseLdap     bool   `json:"use_ldap,omitempty"`
}

// Permission represents a permission in Poweradmin.
type Permission struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Descr string `json:"descr"`
}

// PermissionListResponse represents the response from listing permissions.
type PermissionListResponse struct {
	Permissions []Permission `json:"permissions"`
}

// PermissionResponse represents the response for a single permission.
type PermissionResponse struct {
	Permission Permission `json:"permission"`
}

// BulkRecordOperation represents a single operation in a bulk records request.
type BulkRecordOperation struct {
	Action   string `json:"action"`   // "create", "update", "delete"
	RecordID int    `json:"record_id,omitempty"` // For update/delete operations
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Content  string `json:"content,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

// BulkRecordsRequest represents a bulk operations request.
type BulkRecordsRequest struct {
	Operations []BulkRecordOperation `json:"operations"`
}

// BulkRecordsResponse represents the response from a bulk operations request.
type BulkRecordsResponse struct {
	SuccessCount int      `json:"success_count,omitempty"`
	FailureCount int      `json:"failure_count,omitempty"`
	Errors       []string `json:"errors,omitempty"`
}
