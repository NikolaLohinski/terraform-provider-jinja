package provider

import (
	"context"

	"github.com/nikolalohinski/terraform-provider-jinja/v2/lib"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure the provider satisfies various provider interfaces.
var _ provider.Provider = &jinjaProvider{}

// jinjaProvider defines the provider implementation.
type jinjaProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type jinjaProviderModel struct {
	StrictUndefined types.Bool   `tfsdk:"strict_undefined"`
	TrimBlocks      types.Bool   `tfsdk:"trim_blocks"`
	LeftStripBlocks types.Bool   `tfsdk:"left_strip_blocks"`
	Delimiters      types.Object `tfsdk:"delimiters"`
}

type jinjaDelimitersModel struct {
	BlockStart    types.String `tfsdk:"block_start"`
	BlockEnd      types.String `tfsdk:"block_end"`
	VariableStart types.String `tfsdk:"variable_start"`
	VariableEnd   types.String `tfsdk:"variable_end"`
	CommentStart  types.String `tfsdk:"comment_start"`
	CommentEnd    types.String `tfsdk:"comment_end"`
}

func (p *jinjaProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "jinja"
	resp.Version = p.version
}

func (p *jinjaProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"delimiters": schema.SingleNestedBlock{
				MarkdownDescription: "Custom delimiters for the Jinja engine for all templates",
				Attributes: map[string]schema.Attribute{
					"block_start": schema.StringAttribute{
						Optional: true,
					},
					"block_end": schema.StringAttribute{
						Optional: true,
					},
					"variable_start": schema.StringAttribute{
						Optional: true,
					},
					"variable_end": schema.StringAttribute{
						Optional: true,
					},
					"comment_start": schema.StringAttribute{
						Optional: true,
					},
					"comment_end": schema.StringAttribute{
						Optional: true,
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"strict_undefined": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to `true` to fail on missing items and attribute for all templates",
			},
			"trim_blocks": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to `true` the first newline after a block is removed for all templates",
			},
			"left_strip_blocks": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to `true` leading spaces and tabs are stripped from the start of a line to a block for all templates",
			},
		},
	}
}

func (p *jinjaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data jinjaProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	configuration := lib.Configuration{
		StrictUndefined: data.StrictUndefined.ValueBool(),
		LeftStripBlocks: data.LeftStripBlocks.ValueBool(),
		TrimBlocks:      data.TrimBlocks.ValueBool(),
		Delimiters: lib.Delimiters{
			VariableStart: "{{",
			VariableEnd:   "}}",
			BlockStart:    "{%",
			BlockEnd:      "%}",
			CommentStart:  "{#",
			CommentEnd:    "#}",
		},
	}
	if !data.Delimiters.IsNull() && !data.Delimiters.IsUnknown() {
		var delimiters jinjaDelimitersModel
		resp.Diagnostics.Append(data.Delimiters.As(ctx, &delimiters, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !delimiters.VariableStart.IsNull() && !delimiters.VariableEnd.IsUnknown() {
			configuration.Delimiters.VariableStart = delimiters.VariableStart.ValueString()
		}
		if !delimiters.VariableEnd.IsNull() && !delimiters.VariableEnd.IsUnknown() {
			configuration.Delimiters.VariableEnd = delimiters.VariableEnd.ValueString()
		}
		if !delimiters.BlockStart.IsNull() && !delimiters.VariableEnd.IsUnknown() {
			configuration.Delimiters.BlockStart = delimiters.BlockStart.ValueString()
		}
		if !delimiters.BlockEnd.IsNull() && !delimiters.VariableEnd.IsUnknown() {
			configuration.Delimiters.BlockEnd = delimiters.BlockEnd.ValueString()
		}
		if !delimiters.CommentStart.IsNull() && !delimiters.VariableEnd.IsUnknown() {
			configuration.Delimiters.CommentStart = delimiters.CommentStart.ValueString()
		}
		if !delimiters.CommentEnd.IsNull() && !delimiters.VariableEnd.IsUnknown() {
			configuration.Delimiters.CommentEnd = delimiters.CommentEnd.ValueString()
		}
	}

	resp.DataSourceData = configuration
}

func (p *jinjaProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *jinjaProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewTemplateDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &jinjaProvider{
			version: version,
		}
	}
}
