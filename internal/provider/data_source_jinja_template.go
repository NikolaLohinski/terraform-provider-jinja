package provider

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/terraform-provider-jinja/v2/lib"
)

var (
	_              datasource.DataSourceWithConfigValidators = &TemplateDataSource{}
	defaultTimeout                                           = time.Second * 30
)

func NewTemplateDataSource() datasource.DataSource {
	return &TemplateDataSource{}
}

type TemplateDataSource struct {
	Configuration lib.Configuration
}

type TemplateDataSourceModel struct {
	Source          types.List     `tfsdk:"source"`
	Context         types.List     `tfsdk:"context"`
	Validation      types.Map      `tfsdk:"validation"`
	StrictUndefined types.Bool     `tfsdk:"strict_undefined"`
	TrimBlocks      types.Bool     `tfsdk:"trim_blocks"`
	LeftStripBlocks types.Bool     `tfsdk:"left_strip_blocks"`
	Delimiters      types.Object   `tfsdk:"delimiters"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
	// Computed
	Result        types.String `tfsdk:"result"`
	MergedContext types.String `tfsdk:"merged_context"`
	ID            types.String `tfsdk:"id"`
	// Deprecated
	Header   types.String `tfsdk:"header"`
	Footer   types.String `tfsdk:"footer"`
	Template types.String `tfsdk:"template"`
}
type SourceModel struct {
	Template  types.String `tfsdk:"template"`
	Directory types.String `tfsdk:"directory"`
}
type ContextModel struct {
	Type types.String `tfsdk:"type"`
	Data types.String `tfsdk:"data"`
}

func (d *TemplateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}

func (d *TemplateDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.Conflicting(
			path.MatchRoot("source"),
			path.MatchRoot("template"),
		),
		datasourcevalidator.Conflicting(
			path.MatchRoot("source"),
			path.MatchRoot("header"),
		),
		datasourcevalidator.Conflicting(
			path.MatchRoot("source"),
			path.MatchRoot("footer"),
		),
		datasourcevalidator.AtLeastOneOf(
			path.MatchRoot("source"),
			path.MatchRoot("template"),
		),
	}
}

func (d *TemplateDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The `jinja_template` data source renders a jinja template with a given template with possible JSON schema validation of the context",
		Blocks: map[string]schema.Block{
			"source": schema.ListNestedBlock{
				MarkdownDescription: "Source template to use for rendering",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"template": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Template to render. If required to load an external file, then the `file(...)` function can be used to retrieve the file's content",
						},
						"directory": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Path to the directory to use as the root starting point. If the template is an external file, then the `dirname(...)` function can be used to get the path to the template's directory. Otherwise, just using `path.module` is usually a good idea",
						},
					},
				},
			},
			"delimiters": schema.SingleNestedBlock{
				MarkdownDescription: "Custom delimiters for the Jinja engine. Setting any nested value overrides the one set at the provider level if any",
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
			"context": schema.ListNestedBlock{
				MarkdownDescription: "Context to use while rendering the template. If multiple are passed, they are merged in order with overriding",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: fmt.Sprintf("Type of parsing (one of: `%s`) to perform on the given string", strings.Join(lib.SupportedValuesFormats, "`,`")),
							Validators: []validator.String{
								stringvalidator.OneOf(lib.SupportedValuesFormats...),
							},
						},
						"data": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "A string holding the serialized context",
						},
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"template": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Inlined or path to the jinja template to render. If the template is passed inlined, any filesystem calls such as using the `include` statement or the `fileset` filter won't work as expected. Deprecated in favor of the `source` block",
				DeprecationMessage:  "Deprecated in favor of the `source` block",
			},
			"header": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Header to add at the top of the template before rendering. Deprecated in favor of the `source` block",
				DeprecationMessage:  "Deprecated as the `source.template` field can be used alongside string manipulation within terraform to achieve the same behavior",
			},
			"footer": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Footer to add at the bottom of the template before rendering. Deprecated in favor of the `source` block",
				DeprecationMessage:  "Deprecated as the `source.template` field can be used alongside string manipulation within terraform to achieve the same behavior",
			},
			"strict_undefined": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to `true` to fail on missing items and attribute. Setting this value overrides any value set at the provider level if any",
			},
			"trim_blocks": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to `true` the first newline after a block is removed. Setting this value overrides any value set at the provider level if any",
			},
			"left_strip_blocks": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to `true` leading spaces and tabs are stripped from the start of a line to a block. Setting this value overrides any value set at the provider level if any",
			},

			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Read: true,
			}),
			"validation": schema.MapAttribute{
				Optional:            true,
				MarkdownDescription: "Map of JSON schemas to validate against the context. Schemas are tested sequentially in lexicographic order of this map's keys",
				ElementType:         types.StringType,
			},
			"result": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Rendered template with the given context",
			},
			"merged_context": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "JSON encoded representation of the merged context that has been applied to the template",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The sha256 of the `result` field",
			},
		},
	}
}

func (d *TemplateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	configuration, ok := req.ProviderData.(lib.Configuration)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected data source configure type",
			fmt.Sprintf("Expected lib.Configuration, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.Configuration = configuration
}

func (t *TemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TemplateDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	renderContext := t.parseRenderContext(ctx, data, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	result, values, err := lib.Render(renderContext)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to render",
			fmt.Sprintf("Rendering context returned an error: %s", err.Error()),
		)
		return
	}
	data.Result = types.StringValue(string(result))
	data.ID = types.StringValue(string(sha256.New().Sum(result)))

	merged_context, err := json.Marshal(values)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to build `merged_context` field",
			fmt.Sprintf("Marshalling returned values returned an error: %s", err.Error()),
		)
		return
	}
	data.MergedContext = types.StringValue(string(merged_context))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (t *TemplateDataSource) parseRenderContext(ctx context.Context, data TemplateDataSourceModel, resp *datasource.ReadResponse) *lib.Context {
	values := t.parseValues(ctx, data, resp)
	if resp.Diagnostics.HasError() {
		return nil
	}

	schemas := t.parseSchemas(ctx, data, resp)
	if resp.Diagnostics.HasError() {
		return nil
	}

	configuration := t.parseConfiguration(ctx, data, resp)
	if resp.Diagnostics.HasError() {
		return nil
	}

	source := t.parseSource(ctx, data, resp)
	if resp.Diagnostics.HasError() {
		return nil
	}

	timeout, diagnostics := data.Timeouts.Read(ctx, defaultTimeout)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return nil
	}

	return &lib.Context{
		Source:        source,
		Schemas:       schemas,
		Values:        values,
		Configuration: configuration,
		Timeout:       timeout,
	}
}

func (t *TemplateDataSource) parseSchemas(ctx context.Context, data TemplateDataSourceModel, resp *datasource.ReadResponse) map[string]json.RawMessage {
	stringSchemas := make(map[string]string)
	resp.Diagnostics.Append(data.Validation.ElementsAs(ctx, &stringSchemas, false)...)
	if resp.Diagnostics.HasError() {
		return nil
	}

	schemas := make(map[string]json.RawMessage)
	for name, stringSchema := range stringSchemas {
		schemas[name] = json.RawMessage(stringSchema)
	}

	return schemas
}

func (t *TemplateDataSource) parseConfiguration(ctx context.Context, data TemplateDataSourceModel, resp *datasource.ReadResponse) lib.Configuration {
	if !data.StrictUndefined.IsNull() && !data.StrictUndefined.IsUnknown() {
		t.Configuration.StrictUndefined = data.StrictUndefined.ValueBool()
	}
	if !data.LeftStripBlocks.IsNull() && !data.LeftStripBlocks.IsUnknown() {
		t.Configuration.LeftStripBlocks = data.LeftStripBlocks.ValueBool()
	}
	if !data.TrimBlocks.IsNull() && !data.TrimBlocks.IsUnknown() {
		t.Configuration.TrimBlocks = data.TrimBlocks.ValueBool()
	}
	if !data.Delimiters.IsNull() && !data.Delimiters.IsUnknown() {
		var delimiters jinjaDelimitersModel
		resp.Diagnostics.Append(data.Delimiters.As(ctx, &delimiters, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return t.Configuration
		}
		if !delimiters.VariableStart.IsNull() && !delimiters.VariableStart.IsUnknown() {
			t.Configuration.Delimiters.VariableStart = delimiters.VariableStart.ValueString()
		}
		if !delimiters.VariableEnd.IsNull() && !delimiters.VariableEnd.IsUnknown() {
			t.Configuration.Delimiters.VariableEnd = delimiters.VariableEnd.ValueString()
		}
		if !delimiters.BlockStart.IsNull() && !delimiters.BlockStart.IsUnknown() {
			t.Configuration.Delimiters.BlockStart = delimiters.BlockStart.ValueString()
		}
		if !delimiters.BlockEnd.IsNull() && !delimiters.BlockEnd.IsUnknown() {
			t.Configuration.Delimiters.BlockEnd = delimiters.BlockEnd.ValueString()
		}
		if !delimiters.CommentStart.IsNull() && !delimiters.CommentStart.IsUnknown() {
			t.Configuration.Delimiters.CommentStart = delimiters.CommentStart.ValueString()
		}
		if !delimiters.CommentEnd.IsNull() && !delimiters.CommentEnd.IsUnknown() {
			t.Configuration.Delimiters.CommentEnd = delimiters.CommentEnd.ValueString()
		}
	}

	return t.Configuration
}

func (t *TemplateDataSource) parseValues(ctx context.Context, data TemplateDataSourceModel, resp *datasource.ReadResponse) []lib.Values {
	if !data.Context.IsNull() && !data.Context.IsUnknown() {
		var contexts []ContextModel
		resp.Diagnostics.Append(data.Context.ElementsAs(ctx, &contexts, false)...)
		if resp.Diagnostics.HasError() {
			return nil
		}
		values := make([]lib.Values, len(contexts))
		for index, context := range contexts {
			values[index] = lib.Values{
				Type: context.Type.ValueString(),
				Data: []byte(context.Data.ValueString()),
			}
		}
		return values
	}
	return nil
}

func (t *TemplateDataSource) parseSource(ctx context.Context, data TemplateDataSourceModel, resp *datasource.ReadResponse) lib.Source {
	source := lib.Source{}
	// Legacy behavior via the template field with file/inline handling and footer/header fields
	if !data.Template.IsUnknown() && !data.Template.IsNull() {
		template := data.Template.ValueString()
		if fileContent, err := os.ReadFile(template); err == nil {
			source.Template = string(fileContent)

			absolutePath, err := filepath.Abs(template)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid path",
					fmt.Sprintf("Failed to get an absolute path out of \"%s\": %s", template, err.Error()),
				)
				return source
			}
			source.Directory = filepath.Dir(absolutePath)
		} else {
			source.Directory, err = os.Getwd()
			if err != nil {
				resp.Diagnostics.AddError(
					"Unexpected error",
					fmt.Sprintf("failed to get the current work directory: %s", err.Error()),
				)
				return source
			}
			source.Template = template
		}
		if !data.Header.IsNull() && !data.Header.IsUnknown() {
			source.Template = data.Header.ValueString() + "\n" + source.Template
		}
		if !data.Footer.IsNull() && !data.Footer.IsUnknown() {
			source.Template = source.Template + "\n" + data.Footer.ValueString()
		}
		return source
	}
	var sourceModels []SourceModel
	resp.Diagnostics.Append(data.Source.ElementsAs(ctx, &sourceModels, false)...)
	if resp.Diagnostics.HasError() {
		return source
	}
	sourceModel := sourceModels[0]

	source.Template = sourceModel.Template.ValueString()

	directory, err := filepath.Abs(sourceModel.Directory.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid path",
			fmt.Sprintf("failed to get an absolute path from the given directory: %s", err.Error()),
		)
		return source
	}
	source.Directory = directory

	return source
}
