package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RRSetResource{}
var _ resource.ResourceWithImportState = &RRSetResource{}

func NewRRSetResource() resource.Resource {
	return &RRSetResource{}
}

// RRSetResource defines the resource implementation.
type RRSetResource struct {
	client *Client
}

// RRSetResourceModel describes the resource data model.
type RRSetResourceModel struct {
	ID      types.String         `tfsdk:"id"`
	ZoneID  types.Int64          `tfsdk:"zone_id"`
	Name    types.String         `tfsdk:"name"`
	Type    types.String         `tfsdk:"type"`
	TTL     types.Int64          `tfsdk:"ttl"`
	Records []RRSetRecordModel   `tfsdk:"records"`
}

// RRSetRecordModel describes a single record in the RRSet
type RRSetRecordModel struct {
	Content  types.String `tfsdk:"content"`
	Disabled types.Bool   `tfsdk:"disabled"`
	Priority types.Int64  `tfsdk:"priority"`
}

func (r *RRSetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rrset"
}

func (r *RRSetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This describes the provider and how it's configured.
		MarkdownDescription: "Manages a DNS Resource Record Set (RRSet). An RRSet is a collection of records with the same name and type, managed as a single unit. This matches PowerDNS behavior and is the DNS-correct way to handle multiple records.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RRSet identifier (format: zone_id/name/type)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.Int64Attribute{
				MarkdownDescription: "Zone ID where the RRSet will be created",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Record name (use @ for zone apex, or subdomain like 'www')",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Record type (A, AAAA, CNAME, MX, TXT, etc.)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "Time to live (TTL) in seconds. Defaults to 3600.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3600),
			},
			"records": schema.SetNestedAttribute{
				MarkdownDescription: "Set of record contents. All records in the RRSet share the same name, type, and TTL. Order is not significant.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"content": schema.StringAttribute{
							MarkdownDescription: "Record content (IP address, hostname, text, etc.)",
							Required:            true,
						},
						"disabled": schema.BoolAttribute{
							MarkdownDescription: "Whether this record is disabled. Default: false",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
						"priority": schema.Int64Attribute{
							MarkdownDescription: "Priority for MX, SRV and other priority-bearing records. Default: 0",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(0),
						},
					},
				},
			},
		},
	}
}

func (r *RRSetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RRSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RRSetResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request
	records := make([]map[string]interface{}, len(data.Records))
	for i, rec := range data.Records {
		// Default disabled to false if not set
		disabled := false
		if !rec.Disabled.IsNull() && !rec.Disabled.IsUnknown() {
			disabled = rec.Disabled.ValueBool()
		}
		// Default priority to 0 if not set
		priority := int64(0)
		if !rec.Priority.IsNull() && !rec.Priority.IsUnknown() {
			priority = rec.Priority.ValueInt64()
		}
		records[i] = map[string]interface{}{
			"content":  rec.Content.ValueString(),
			"disabled": disabled,
			"priority": priority,
		}
	}

	rrsetData := map[string]interface{}{
		"name":    data.Name.ValueString(),
		"type":    data.Type.ValueString(),
		"ttl":     data.TTL.ValueInt64(),
		"records": records,
	}

	tflog.Debug(ctx, "Creating RRSet", map[string]interface{}{
		"zone_id": data.ZoneID.ValueInt64(),
		"name":    data.Name.ValueString(),
		"type":    data.Type.ValueString(),
	})

	// Call API to create RRSet
	err := r.client.CreateRRSet(ctx, data.ZoneID.ValueInt64(), rrsetData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create RRSet, got error: %s", err))
		return
	}

	// Read back the RRSet to get the server's actual values
	// This ensures state matches what the API stored (normalized values, defaults applied, etc.)
	rrset, err := r.client.GetRRSet(ctx, data.ZoneID.ValueInt64(), data.Name.ValueString(), data.Type.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read RRSet after create, got error: %s", err))
		return
	}

	// Generate ID
	data.ID = types.StringValue(fmt.Sprintf("%d/%s/%s", data.ZoneID.ValueInt64(), data.Name.ValueString(), data.Type.ValueString()))

	// Update model from API response
	data.TTL = types.Int64Value(rrset.TTL)

	// Update records from API response
	createdRecords := make([]RRSetRecordModel, len(rrset.Records))
	for i, rec := range rrset.Records {
		createdRecords[i] = RRSetRecordModel{
			Content:  types.StringValue(rec.Content),
			Disabled: types.BoolValue(rec.Disabled),
			Priority: types.Int64Value(rec.Priority),
		}
	}
	data.Records = createdRecords

	tflog.Trace(ctx, "Created RRSet", map[string]interface{}{
		"zone_id": data.ZoneID.ValueInt64(),
		"name":    data.Name.ValueString(),
		"type":    data.Type.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RRSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RRSetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to read RRSet
	rrset, err := r.client.GetRRSet(ctx, data.ZoneID.ValueInt64(), data.Name.ValueString(), data.Type.ValueString())
	if err != nil {
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read RRSet, got error: %s", err))
		return
	}

	// Update model from API response
	data.TTL = types.Int64Value(rrset.TTL)

	// Update records
	records := make([]RRSetRecordModel, len(rrset.Records))
	for i, rec := range rrset.Records {
		records[i] = RRSetRecordModel{
			Content:  types.StringValue(rec.Content),
			Disabled: types.BoolValue(rec.Disabled),
			Priority: types.Int64Value(rec.Priority),
		}
	}
	data.Records = records

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RRSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RRSetResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request
	records := make([]map[string]interface{}, len(data.Records))
	for i, rec := range data.Records {
		// Default disabled to false if not set
		disabled := false
		if !rec.Disabled.IsNull() && !rec.Disabled.IsUnknown() {
			disabled = rec.Disabled.ValueBool()
		}
		// Default priority to 0 if not set
		priority := int64(0)
		if !rec.Priority.IsNull() && !rec.Priority.IsUnknown() {
			priority = rec.Priority.ValueInt64()
		}
		records[i] = map[string]interface{}{
			"content":  rec.Content.ValueString(),
			"disabled": disabled,
			"priority": priority,
		}
	}

	rrsetData := map[string]interface{}{
		"name":    data.Name.ValueString(),
		"type":    data.Type.ValueString(),
		"ttl":     data.TTL.ValueInt64(),
		"records": records,
	}

	tflog.Debug(ctx, "Updating RRSet", map[string]interface{}{
		"zone_id": data.ZoneID.ValueInt64(),
		"name":    data.Name.ValueString(),
		"type":    data.Type.ValueString(),
	})

	// Call API to update RRSet
	err := r.client.UpdateRRSet(ctx, data.ZoneID.ValueInt64(), rrsetData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update RRSet, got error: %s", err))
		return
	}

	// Read back the RRSet to get the server's normalized values
	// This ensures state matches what the API actually stored (normalized TTL, record ordering, etc.)
	rrset, err := r.client.GetRRSet(ctx, data.ZoneID.ValueInt64(), data.Name.ValueString(), data.Type.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read RRSet after update, got error: %s", err))
		return
	}

	// Update model from API response
	data.TTL = types.Int64Value(rrset.TTL)

	// Update records from API response
	updatedRecords := make([]RRSetRecordModel, len(rrset.Records))
	for i, rec := range rrset.Records {
		updatedRecords[i] = RRSetRecordModel{
			Content:  types.StringValue(rec.Content),
			Disabled: types.BoolValue(rec.Disabled),
			Priority: types.Int64Value(rec.Priority),
		}
	}
	data.Records = updatedRecords

	tflog.Trace(ctx, "Updated RRSet", map[string]interface{}{
		"zone_id": data.ZoneID.ValueInt64(),
		"name":    data.Name.ValueString(),
		"type":    data.Type.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RRSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RRSetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting RRSet", map[string]interface{}{
		"zone_id": data.ZoneID.ValueInt64(),
		"name":    data.Name.ValueString(),
		"type":    data.Type.ValueString(),
	})

	// Call API to delete RRSet
	err := r.client.DeleteRRSet(ctx, data.ZoneID.ValueInt64(), data.Name.ValueString(), data.Type.ValueString())
	if err != nil {
		// If the RRSet was already deleted outside of Terraform, that's fine
		if IsNotFoundError(err) {
			tflog.Info(ctx, "RRSet already deleted, ignoring error", map[string]interface{}{
				"zone_id": data.ZoneID.ValueInt64(),
				"name":    data.Name.ValueString(),
				"type":    data.Type.ValueString(),
			})
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete RRSet, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted RRSet", map[string]interface{}{
		"zone_id": data.ZoneID.ValueInt64(),
		"name":    data.Name.ValueString(),
		"type":    data.Type.ValueString(),
	})
}

func (r *RRSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: zone_id/name/type
	// Example: terraform import poweradmin_rrset.www 123/www/A
	tflog.Debug(ctx, "Importing RRSet", map[string]interface{}{
		"import_id": req.ID,
	})

	// Parse the import ID - format: zone_id/name/type
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'zone_id/name/type', got: %s", req.ID),
		)
		return
	}

	zoneID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Zone ID",
			fmt.Sprintf("Zone ID must be a valid integer, got: %s", parts[0]),
		)
		return
	}

	name := parts[1]
	recordType := parts[2]

	// Set the parsed values in state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), zoneID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), recordType)...)
}
