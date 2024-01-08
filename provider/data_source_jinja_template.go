package jinja

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/nikolalohinski/terraform-provider-jinja/lib"
)

var default_delimiters = map[string]interface{}{
	"block_start":    "{%",
	"block_end":      "%}",
	"variable_start": "{{",
	"variable_end":   "}}",
	"comment_start":  "{#",
	"comment_end":    "#}",
}

var context_types = []string{"yaml", "json", "toml"}

func strictUndefinedSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeBool,
		Description: "Toggle to fail rendering on missing attribute/item",
		Optional:    true,
	}
}

func leftStripBlocksSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeBool,
		Description: "If this is set to `true` leading spaces and tabs are stripped from the start of a line to a block",
		Optional:    true,
	}
}

func trimBlocksSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeBool,
		Description: "If this is set to `true` the first newline after a block is removed (block, not variable tag!)",
		Optional:    true,
	}
}

func delimitersSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "Custom delimiters for the jinja engine",
		Optional:    true,
		Computed:    true,
		// Must provide default at runtime because TypeList and TypSet ignore DefaultFunc
		// See https://github.com/hashicorp/terraform-plugin-sdk/issues/142
		// DefaultFunc: func() (interface{}, error) { return []interface{}{default_delimiters}, nil },
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"block_start": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  default_delimiters["block_start"].(string),
				},
				"block_end": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  default_delimiters["block_end"].(string),
				},
				"variable_start": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  default_delimiters["variable_start"].(string),
				},
				"variable_end": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  default_delimiters["variable_end"].(string),
				},
				"comment_start": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  default_delimiters["comment_start"].(string),
				},
				"comment_end": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  default_delimiters["comment_end"].(string),
				},
			},
		},
	}
}

func dataSourceJinjaTemplate() *schema.Resource {
	return &schema.Resource{
		Read:        read,
		Description: "The jinja_template data source renders a jinja template with a given template with possible JSON schema validation of the context",
		Timeouts: &schema.ResourceTimeout{
			Read: schema.DefaultTimeout(5 * time.Second),
		},
		Schema: map[string]*schema.Schema{
			"header": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "Header to add at the top of the template before rendering. Deprecated in favor of the `source` block",
				Deprecated:    "Deprecated as the `source.template` field can be used alongside string manipulation within terraform to achieve the same behavior",
				ConflictsWith: []string{"source"},
			},
			"footer": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "Footer to add at the bottom of the template before rendering. Deprecated in favor of the `source` block",
				Deprecated:    "Deprecated as the `source.template` field can be used alongside string manipulation within terraform to achieve the same behavior",
				ConflictsWith: []string{"source"},
			},
			"template": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "Inlined or path to the jinja template to render. If the template is passed inlined, any filesystem calls such as using the `include` statement or the `fileset` filter won't work as expected. Deprecated in favor of the `source` block",
				Deprecated:    "Deprecated in favor of the `source` block",
				ConflictsWith: []string{"source"},
			},
			"source": {
				Type:          schema.TypeList,
				Optional:      true,
				Description:   "Source template to use for rendering",
				ConflictsWith: []string{"template", "header", "footer"},
				MaxItems:      1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"template": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Template to render. If required to load an external file, then the `file(...)` function can be used to retrieve the file's content",
						},
						"directory": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Path to the directory to use as the root starting point. If the template is an external file, then the `dirname(...)` function can be used to get the path to the template's directory. Otherwise, just using `path.module` is usually a good idea",
						},
					},
				},
			},
			"validation": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Map of JSON schemas to validate against the context. Schemas are tested sequentially in lexicographic order of this map's keys",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"context": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Context to use while rendering the template. If multiple are passed, they are merged in order with overriding",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(context_types, true),
							Description:  fmt.Sprintf("Type of parsing (one of: %s) to perform on the given string", strings.Join(context_types, ", ")),
						},
						"data": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "A string holding the serialized context",
						},
					},
				},
			},
			"delimiters":        delimitersSchema(),
			"strict_undefined":  strictUndefinedSchema(),
			"trim_blocks":       trimBlocksSchema(),
			"left_strip_blocks": leftStripBlocksSchema(),
			"result": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Rendered template with the given context",
			},
			"merged_context": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "JSON encoded representation of the merged context that has been applied to the template",
			},
		},
	}
}

func read(d *schema.ResourceData, meta interface{}) error {
	ctx, err := parseContext(d, meta)
	if err != nil {
		return fmt.Errorf("failed to parse configuration passed to provider: %s", err)
	}

	result, values, err := lib.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render context: %s", err)
	}
	merged_context, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("failed to marshal merged context to JSON: %s", err)
	}
	if err := d.Set("merged_context", string(merged_context)); err != nil {
		return fmt.Errorf("failed to output merged context: %s", err)
	}
	if err := d.Set("result", string(result)); err != nil {
		return fmt.Errorf("failed to output result: %s", err)
	}

	hash_function := sha256.New()
	hash_function.Write(result)

	d.SetId(base64.URLEncoding.EncodeToString(hash_function.Sum(nil)))

	return nil
}

func parseContext(d *schema.ResourceData, meta interface{}) (*lib.Context, error) {
	values, err := parseValues(d)
	if err != nil {
		return nil, fmt.Errorf("failed to parse context: %s", err)
	}

	schemas, err := parseSchemas(d)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schemas: %s", err)
	}

	configuration, err := parseConfiguration(d, meta)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %s", err)
	}

	source, err := parseSource(d)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %s", err)
	}

	return &lib.Context{
		Source:        source,
		Schemas:       schemas,
		Values:        values,
		Configuration: configuration,
		Timeout:       d.Timeout(schema.TimeoutRead),
	}, nil
}

func parseSchemas(d *schema.ResourceData) (map[string]json.RawMessage, error) {
	stringSchemas := make(map[string]string)
	if schemaField, ok := d.GetOk("schema"); ok {
		stringSchemas[humanize.Ordinal(1)] = schemaField.(string)
	} else if schemasField, ok := d.GetOk("schemas"); ok {
		castSchemasField, castOk := schemasField.([]interface{})
		if !castOk {
			return nil, fmt.Errorf("field 'schemas' is not a list: %v", schemasField)
		}
		for index, schema := range castSchemasField {
			stringSchemas[humanize.Ordinal(index+1)] = schema.(string)
		}
	} else if validationField, ok := d.GetOk("validation"); ok {
		castValidationField, ok := validationField.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("field 'validation' is not a map of string: %v", validationField)
		}
		for name, field := range castValidationField {
			castField, ok := field.(string)
			if !ok {
				return nil, fmt.Errorf("key '%s' in field 'validation' is not a string: %v", name, field)
			}
			stringSchemas[name] = castField
		}
	}

	schemas := make(map[string]json.RawMessage)
	for name, stringSchema := range stringSchemas {
		schemas[name] = json.RawMessage(stringSchema)
	}

	return schemas, nil
}

func parseConfiguration(d *schema.ResourceData, meta interface{}) (lib.Configuration, error) {
	metaObject, ok := meta.(map[string]interface{})
	if !ok {
		return lib.Configuration{}, errors.New("provider configuration is invalid")
	}
	providerDelimiters, ok := metaObject["delimiters"].(map[string]interface{})
	if !ok {
		return lib.Configuration{}, errors.New("provider delimiters configuration is invalid")
	}
	var delimiters map[string]interface{}
	delimitersBlock, ok := d.GetOk("delimiters.0")
	if !ok {
		delimiters = providerDelimiters
	} else {
		delimiters = delimitersBlock.(map[string]interface{})
		for name, delimiter := range delimiters {
			if default_value, ok := default_delimiters[name]; ok && delimiter != default_value {
				continue
			}
			delimiters[name] = providerDelimiters[name]
		}
	}

	strictUndefined, ok := d.GetOkExists("strict_undefined")
	if !ok {
		strictUndefined, ok = metaObject["strict_undefined"]
		if !ok {
			strictUndefined = false
		}
	}
	leftStripBlocks, ok := d.GetOkExists("left_strip_blocks")
	if !ok {
		leftStripBlocks, ok = metaObject["left_strip_blocks"]
		if !ok {
			leftStripBlocks = false
		}
	}
	trimBlocks, ok := d.GetOkExists("trim_blocks")
	if !ok {
		trimBlocks, ok = metaObject["trim_blocks"]
		if !ok {
			trimBlocks = false
		}
	}

	return lib.Configuration{
		StrictUndefined: strictUndefined.(bool),
		LeftStripBlocks: leftStripBlocks.(bool),
		TrimBlocks:      trimBlocks.(bool),
		Delimiters: lib.Delimiters{
			BlockStart:    delimiters["block_start"].(string),
			BlockEnd:      delimiters["block_end"].(string),
			VariableStart: delimiters["variable_start"].(string),
			VariableEnd:   delimiters["variable_end"].(string),
			CommentStart:  delimiters["comment_start"].(string),
			CommentEnd:    delimiters["comment_end"].(string),
		},
	}, nil
}

func parseValues(d *schema.ResourceData) ([]lib.Values, error) {
	context_blocks, ok := d.GetOk("context")
	if ok {
		contexts, ok := context_blocks.([]interface{})
		if !ok {
			return nil, fmt.Errorf("context blocks are invalid: %s", context_blocks)
		}
		values := make([]lib.Values, len(contexts))

		for index := range contexts {
			kind := d.Get(fmt.Sprintf("context.%d.type", index)).(string)
			dataField := d.Get(fmt.Sprintf("context.%d.data", index))
			values[index] = lib.Values{
				Type: kind,
				Data: []byte(dataField.(string)),
			}
		}

		return values, nil
	}

	return nil, nil
}

func parseSource(d *schema.ResourceData) (lib.Source, error) {
	source := lib.Source{}

	templateField, okTemplate := d.GetOk("template")
	_, okSource := d.GetOk("source")

	if !okTemplate && !okSource {
		return source, fmt.Errorf("neither \"template\" nor \"source\" was set")
	}
	if okTemplate && okSource {
		return source, fmt.Errorf("can not set \"template\" and \"source\" at the same time")
	}
	// Legacy behavior with file/inline handling and footer/header fields
	if okTemplate {
		if fileContent, err := os.ReadFile(templateField.(string)); err == nil {
			source.Template = string(fileContent)

			absolutePath, err := filepath.Abs(templateField.(string))
			if err != nil {
				return source, fmt.Errorf("failed to get an absolute path out of \"%s\": %s", templateField, err)
			}
			source.Directory = filepath.Dir(absolutePath)
		} else {
			source.Directory, err = os.Getwd()
			if err != nil {
				return source, fmt.Errorf("failed to get current working directory: %s", err)
			}
			source.Template = templateField.(string)
		}
		if header, ok := d.GetOk("header"); ok {
			source.Template = header.(string) + "\n" + source.Template
		}
		if footer, ok := d.GetOk("footer"); ok {
			source.Template = source.Template + "\n" + footer.(string)
		}
		return source, nil
	}

	template, ok := d.GetOk("source.0.template")
	if !ok {
		return source, errors.New("failed to get source.template field")
	}
	source.Template = template.(string)

	directory, ok := d.GetOk("source.0.directory")
	if !ok {
		return source, errors.New("failed to get source.directory field")
	}
	var err error
	source.Directory, err = filepath.Abs(directory.(string))
	if err != nil {
		return source, fmt.Errorf("failed to get an absolute path out of \"%s\": %s", directory, err)
	}

	return source, nil
}
