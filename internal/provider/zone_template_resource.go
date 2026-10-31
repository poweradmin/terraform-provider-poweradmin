// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ZoneTemplateResource{}
var _ resource.ResourceWithImportState = &ZoneTemplateResource{}

func NewZoneTemplateResource() resource.Resource {
	return &ZoneTemplateResource{}
}

// ZoneTemplateResource defines the resource implementation.
type ZoneTemplateResource struct {
	client *Client
}

// ZoneTemplateResourceModel describes the resource data model.
type ZoneTemplateResourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsGlobal    types.Bool   `tfsdk:"is_global"`
	Owner       types.Int64  `tfsdk:"owner"`
	Groups      types.Set    `tfsdk:"groups"`
}

func (r *ZoneTemplateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_template"
}

func (r *ZoneTemplateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a zone template in Poweradmin. Zone templates are reusable sets of DNS records that can be applied when creating new zones. Requires Poweradmin 4.2.0+.",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for the zone template",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the zone template (must be unique)",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the zone template",
				Required:            true,
			},
			"is_global": schema.BoolAttribute{
				MarkdownDescription: "Whether this template is global (visible to all users). Requires ueberuser permission to set true. Defaults to `false` on first create; once set, the value is preserved across applies (toggle it explicitly to change it, since toggling reassigns ownership server-side).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"owner": schema.Int64Attribute{
				MarkdownDescription: "User ID that owns the template. `0` indicates a global template.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"groups": schema.SetAttribute{
				MarkdownDescription: "IDs of the user groups the template is shared with. Requires Poweradmin 4.6.0+; older versions ignore it with a warning. Leave it out to keep the current shares; set it to replace them, listing every group the template should keep, including ones a super admin added that you cannot remove yourself. The owner can share only with groups they belong to, a super admin with any group, and a global template cannot be shared. A group member who is not the owner must leave it out, as any configured value is refused.",
				ElementType:         types.Int64Type,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					zoneTemplateGroupsPlanModifier{},
				},
			},
		},
	}
}

func (r *ZoneTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZoneTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ZoneTemplateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := CreateZoneTemplateRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		IsGlobal:    data.IsGlobal.ValueBool(),
	}
	// Only a configured value is sent; a planned one may just be the stored shares
	var configGroups types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("groups"), &configGroups)...)
	if resp.Diagnostics.HasError() {
		return
	}
	createReq.Groups = groupIDs(configGroups)

	tflog.Debug(ctx, "Creating zone template", map[string]interface{}{
		"name":      createReq.Name,
		"is_global": createReq.IsGlobal,
	})

	templateID, err := r.client.CreateZoneTemplate(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Zone Template",
			fmt.Sprintf("Could not create zone template: %s", err.Error()),
		)
		return
	}

	template, err := r.client.GetZoneTemplate(ctx, templateID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Zone Template",
			fmt.Sprintf("Could not read created zone template ID %d: %s", templateID, err.Error()),
		)
		return
	}

	r.applyToModel(template, &data)
	warnIfGroupsNotReturned(createReq.Groups, template, &resp.Diagnostics)

	tflog.Debug(ctx, "Zone template created successfully", map[string]interface{}{
		"id": data.ID.ValueInt64(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ZoneTemplateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := int(data.ID.ValueInt64())

	tflog.Debug(ctx, "Reading zone template", map[string]interface{}{
		"id": templateID,
	})

	template, err := r.client.GetZoneTemplate(ctx, templateID)
	if err != nil {
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Zone Template",
			fmt.Sprintf("Could not read zone template ID %d: %s", templateID, err.Error()),
		)
		return
	}

	r.applyToModel(template, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ZoneTemplateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := int(data.ID.ValueInt64())

	updateReq := UpdateZoneTemplateRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		IsGlobal:    data.IsGlobal.ValueBool(),
	}
	var configGroups types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("groups"), &configGroups)...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateReq.Groups = groupIDs(configGroups)

	tflog.Debug(ctx, "Updating zone template", map[string]interface{}{
		"id": templateID,
	})

	if err := r.client.UpdateZoneTemplate(ctx, templateID, updateReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Zone Template",
			fmt.Sprintf("Could not update zone template ID %d: %s", templateID, err.Error()),
		)
		return
	}

	template, err := r.client.GetZoneTemplate(ctx, templateID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Zone Template",
			fmt.Sprintf("Could not read updated zone template ID %d: %s", templateID, err.Error()),
		)
		return
	}

	r.applyToModel(template, &data)
	warnIfGroupsNotReturned(updateReq.Groups, template, &resp.Diagnostics)

	tflog.Debug(ctx, "Zone template updated successfully")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ZoneTemplateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := int(data.ID.ValueInt64())

	tflog.Debug(ctx, "Deleting zone template", map[string]interface{}{
		"id": templateID,
	})

	if err := r.client.DeleteZoneTemplate(ctx, templateID); err != nil {
		if IsNotFoundError(err) {
			tflog.Info(ctx, "Zone template already deleted, ignoring error", map[string]interface{}{
				"id": templateID,
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Zone Template",
			fmt.Sprintf("Could not delete zone template ID %d: %s", templateID, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Zone template deleted successfully")
}

func (r *ZoneTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Zone Template",
			fmt.Sprintf("Could not parse zone template ID '%s': %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func (r *ZoneTemplateResource) applyToModel(template *ZoneTemplate, data *ZoneTemplateResourceModel) {
	data.ID = types.Int64Value(int64(template.ID))
	data.Name = types.StringValue(template.Name)
	data.Description = types.StringValue(template.Description)
	data.IsGlobal = types.BoolValue(template.IsGlobal)
	data.Owner = types.Int64Value(int64(template.Owner))

	if template.Groups == nil {
		// A server before 4.6.0 returns no groups: keep a known value so the plan holds
		if data.Groups.IsNull() || data.Groups.IsUnknown() {
			data.Groups = types.SetValueMust(types.Int64Type, []attr.Value{})
		}
		return
	}
	data.Groups = templateGroupSet(*template.Groups)
}

// templateGroupSet turns the groups a template is shared with into a set of their ids.
func templateGroupSet(shares []ZoneTemplateGroup) types.Set {
	groups := make([]attr.Value, 0, len(shares))
	for _, group := range shares {
		groups = append(groups, types.Int64Value(int64(group.ID)))
	}
	return types.SetValueMust(types.Int64Type, groups)
}

// warnIfGroupsNotReturned flags groups that were sent but that the server never
// echoes, which is how Poweradmin before 4.6.0 behaves (it ignores the field).
func warnIfGroupsNotReturned(sent *[]int, template *ZoneTemplate, diags *diag.Diagnostics) {
	if sent == nil || len(*sent) == 0 || template.Groups != nil {
		return
	}
	diags.AddWarning(
		"Zone Template Groups Not Returned",
		"The groups were sent but the server did not return a groups field. Poweradmin before 4.6.0 ignores template sharing, so the groups were probably not stored; the configured value is kept in state.",
	)
}

// zoneTemplateGroupsPlanModifier keeps the stored shares in the plan when the
// configuration leaves groups out, except for a template made global, which
// Poweradmin stores without shares.
type zoneTemplateGroupsPlanModifier struct{}

var _ planmodifier.Set = zoneTemplateGroupsPlanModifier{}

func (m zoneTemplateGroupsPlanModifier) Description(_ context.Context) string {
	return "Keeps the stored shares unless groups is configured or the template becomes global."
}

func (m zoneTemplateGroupsPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m zoneTemplateGroupsPlanModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	// A configured or unknown value is the plan; destroy needs nothing
	if !req.ConfigValue.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var isGlobal types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("is_global"), &isGlobal)...)
	if isGlobal.IsUnknown() {
		resp.PlanValue = types.SetUnknown(types.Int64Type)
		return
	}
	if !isGlobal.IsNull() && isGlobal.ValueBool() {
		resp.PlanValue = types.SetValueMust(types.Int64Type, []attr.Value{})
		return
	}
	if !req.StateValue.IsNull() {
		resp.PlanValue = req.StateValue
	}
}

// groupIDs returns the configured group ids, or nil when groups is not set, so the
// request leaves the field out and Poweradmin keeps the current shares.
func groupIDs(groups types.Set) *[]int {
	if groups.IsNull() || groups.IsUnknown() {
		return nil
	}
	ids := make([]int, 0, len(groups.Elements()))
	for _, element := range groups.Elements() {
		if id, ok := element.(types.Int64); ok {
			ids = append(ids, int(id.ValueInt64()))
		}
	}
	return &ids
}
