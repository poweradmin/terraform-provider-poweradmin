// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure PoweradminProvider satisfies various provider interfaces.
var _ provider.Provider = &PoweradminProvider{}
var _ provider.ProviderWithFunctions = &PoweradminProvider{}
var _ provider.ProviderWithEphemeralResources = &PoweradminProvider{}

// PoweradminProvider defines the provider implementation.
type PoweradminProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// PoweradminProviderModel describes the provider data model.
type PoweradminProviderModel struct {
	ApiUrl     types.String `tfsdk:"api_url"`
	ApiKey     types.String `tfsdk:"api_key"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	Insecure   types.Bool   `tfsdk:"insecure"`
	ApiVersion types.String `tfsdk:"api_version"`
}

func (p *PoweradminProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "poweradmin"
	resp.Version = p.version
}

func (p *PoweradminProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provider for managing Poweradmin DNS zones and records. Compatible with both Terraform and OpenTofu.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				MarkdownDescription: "Poweradmin API base URL (e.g., https://dns.example.com)",
				Required:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key for authentication (X-API-Key header)",
				Optional:            true,
				Sensitive:           true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username for HTTP basic authentication (alternative to api_key)",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for HTTP basic authentication",
				Optional:            true,
				Sensitive:           true,
			},
			"insecure": schema.BoolAttribute{
				MarkdownDescription: "Skip TLS certificate verification (not recommended for production)",
				Optional:            true,
			},
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Poweradmin API version to use. Only 'v2' is supported (Poweradmin 4.1.0+). Defaults to 'v2'",
				Optional:            true,
			},
		},
	}
}

func (p *PoweradminProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data PoweradminProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validate configuration
	if data.ApiUrl.IsNull() || data.ApiUrl.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing API URL",
			"The api_url attribute is required for the Poweradmin provider",
		)
		return
	}

	// Validate authentication: require either API key or username/password
	hasApiKey := !data.ApiKey.IsNull() && data.ApiKey.ValueString() != ""
	hasBasicAuth := !data.Username.IsNull() && data.Username.ValueString() != "" &&
		!data.Password.IsNull() && data.Password.ValueString() != ""

	if !hasApiKey && !hasBasicAuth {
		resp.Diagnostics.AddError(
			"Missing Authentication",
			"Either api_key or both username and password must be provided for authentication",
		)
		return
	}

	// Validate API version if specified
	if !data.ApiVersion.IsNull() && data.ApiVersion.ValueString() != "" {
		apiVersion := data.ApiVersion.ValueString()
		if apiVersion != "v2" {
			resp.Diagnostics.AddError(
				"Invalid API Version",
				"api_version must be 'v2' (Poweradmin 4.1.0+). This is the only supported version.",
			)
			return
		}
	}

	// Create Poweradmin API client
	client, err := NewClient(&data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Poweradmin API Client",
			fmt.Sprintf("Failed to initialize API client: %s", err.Error()),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *PoweradminProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewZoneResource,
		NewRecordResource,
		NewRRSetResource,
		NewUserResource,
	}
}

func (p *PoweradminProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		// No ephemeral resources currently implemented.
		// Potential future enhancement: temporary API keys if Poweradmin REST API supports it.
	}
}

func (p *PoweradminProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewZoneDataSource,
		NewPermissionDataSource,
		NewRecordsDataSource,
		NewRRSetsDataSource,
	}
}

func (p *PoweradminProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		// No provider functions currently implemented.
		// Potential future enhancements: FQDN formatting, DNS validation helpers, etc.
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PoweradminProvider{
			version: version,
		}
	}
}
