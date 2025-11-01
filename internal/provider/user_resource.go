// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithImportState = &UserResource{}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

// UserResource defines the resource implementation.
type UserResource struct {
	client *Client
}

// UserResourceModel describes the resource data model.
type UserResourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	Fullname    types.String `tfsdk:"fullname"`
	Email       types.String `tfsdk:"email"`
	Description types.String `tfsdk:"description"`
	Active      types.Bool   `tfsdk:"active"`
	PermTempl   types.Int64  `tfsdk:"perm_templ"`
	UseLdap     types.Bool   `tfsdk:"use_ldap"`
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a user in Poweradmin. Users can be assigned permission templates and can own DNS zones.",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for the user",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Unique username for the user",
				Required:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "User password (will be hashed). Cannot be read back from the API.",
				Required:            true,
				Sensitive:           true,
			},
			"fullname": schema.StringAttribute{
				MarkdownDescription: "Full name of the user",
				Required:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email address of the user",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description or notes about the user",
				Optional:            true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the user account is active. Defaults to true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"perm_templ": schema.Int64Attribute{
				MarkdownDescription: "Permission template ID to assign to the user",
				Optional:            true,
			},
			"use_ldap": schema.BoolAttribute{
				MarkdownDescription: "Whether the user should use LDAP authentication. Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build create request
	createReq := CreateUserRequest{
		Username: data.Username.ValueString(),
		Password: data.Password.ValueString(),
		Fullname: data.Fullname.ValueString(),
		Email:    data.Email.ValueString(),
	}

	// Set optional fields
	if !data.Description.IsNull() {
		createReq.Description = data.Description.ValueString()
	}
	if !data.Active.IsNull() {
		createReq.Active = data.Active.ValueBool()
	} else {
		createReq.Active = true
	}
	if !data.PermTempl.IsNull() {
		createReq.PermTempl = int(data.PermTempl.ValueInt64())
	}
	if !data.UseLdap.IsNull() {
		createReq.UseLdap = data.UseLdap.ValueBool()
	} else {
		createReq.UseLdap = false
	}

	tflog.Debug(ctx, "Creating user", map[string]interface{}{
		"username": createReq.Username,
	})

	// Call API to create user
	user, err := r.client.CreateUser(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating User",
			fmt.Sprintf("Could not create user: %s", err.Error()),
		)
		return
	}

	// Update state with created user
	data.ID = types.Int64Value(int64(user.UserID))
	data.Username = types.StringValue(user.Username)
	data.Fullname = types.StringValue(user.Fullname)
	data.Email = types.StringValue(user.Email)
	data.Active = types.BoolValue(user.Active)

	if user.Description != "" {
		data.Description = types.StringValue(user.Description)
	}
	if user.PermTempl != 0 {
		data.PermTempl = types.Int64Value(int64(user.PermTempl))
	}
	data.UseLdap = types.BoolValue(user.UseLdap)

	// Password is write-only, keep it in state
	// data.Password is already set from plan

	tflog.Debug(ctx, "User created successfully", map[string]interface{}{
		"id": data.ID.ValueInt64(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	userID := int(data.ID.ValueInt64())

	tflog.Debug(ctx, "Reading user", map[string]interface{}{
		"id": userID,
	})

	// Call API to get user
	user, err := r.client.GetUser(ctx, userID)
	if err != nil {
		// If user not found, remove from state
		if err.Error() == "user not found" || err.Error() == "404" {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading User",
			fmt.Sprintf("Could not read user ID %d: %s", userID, err.Error()),
		)
		return
	}

	// Update state with fetched data
	data.ID = types.Int64Value(int64(user.UserID))
	data.Username = types.StringValue(user.Username)
	data.Fullname = types.StringValue(user.Fullname)
	data.Email = types.StringValue(user.Email)
	data.Active = types.BoolValue(user.Active)

	if user.Description != "" {
		data.Description = types.StringValue(user.Description)
	} else {
		data.Description = types.StringNull()
	}

	if user.PermTempl != 0 {
		data.PermTempl = types.Int64Value(int64(user.PermTempl))
	} else {
		data.PermTempl = types.Int64Null()
	}

	data.UseLdap = types.BoolValue(user.UseLdap)

	// Password cannot be read from API, keep existing value in state

	tflog.Debug(ctx, "User read successfully")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	userID := int(data.ID.ValueInt64())

	// Build update request
	updateReq := UpdateUserRequest{
		Username: data.Username.ValueString(),
		Fullname: data.Fullname.ValueString(),
		Email:    data.Email.ValueString(),
	}

	// Check if password changed
	var oldData UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &oldData)...)
	if !resp.Diagnostics.HasError() {
		if !data.Password.Equal(oldData.Password) {
			updateReq.Password = data.Password.ValueString()
		}
	}

	// Set optional fields
	if !data.Description.IsNull() {
		updateReq.Description = data.Description.ValueString()
	}
	if !data.Active.IsNull() {
		updateReq.Active = data.Active.ValueBool()
	}
	if !data.PermTempl.IsNull() {
		updateReq.PermTempl = int(data.PermTempl.ValueInt64())
	}
	if !data.UseLdap.IsNull() {
		updateReq.UseLdap = data.UseLdap.ValueBool()
	}

	tflog.Debug(ctx, "Updating user", map[string]interface{}{
		"id": userID,
	})

	// Call API to update user
	_, err := r.client.UpdateUser(ctx, userID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating User",
			fmt.Sprintf("Could not update user ID %d: %s", userID, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "User updated successfully")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	userID := int(data.ID.ValueInt64())

	tflog.Debug(ctx, "Deleting user", map[string]interface{}{
		"id": userID,
	})

	// Call API to delete user
	err := r.client.DeleteUser(ctx, userID, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting User",
			fmt.Sprintf("Could not delete user ID %d: %s", userID, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "User deleted successfully")
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Convert the ID string to int64
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing User",
			fmt.Sprintf("Could not parse user ID '%s': %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
