package jinja

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

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

var context_types = []string{"yaml", "json"}

func strictUndefinedSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeBool,
		Description: "Toggle to fail rendering on missing attribute/item",
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
		Schema: map[string]*schema.Schema{
			"header": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Header to add at the top of the template before rendering",
			},
			"footer": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Footer to add at the bottom of the template before rendering",
			},
			"template": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Inlined or path to the jinja template to render. If the template is passed inlined, any filesystem calls such as using the `include` statement or the `fileset` filter won't work as expected.",
			},
			"schema": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "Either inline or a path to a JSON schema to validate the context",
				Deprecated:    "Deprecated in favor of the 'validation' field",
				ConflictsWith: []string{"schemas", "validation"},
			},
			"schemas": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of either inline or paths to JSON schemas to validate one by one in order against the context",
				MinItems:    1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Deprecated:    "Deprecated in favor of the 'validation' field",
				ConflictsWith: []string{"schema", "validation"},
			},
			"validation": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Map of either inline or paths to JSON schemas to validate one by one in order against the context",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"schema", "schemas"},
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
							Description:  fmt.Sprintf("Type of parsing (one of: %s) to perform on the given string or file", strings.Join(context_types, ", ")),
						},
						"data": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Either a string holding the serialized context or path to a file",
						},
					},
				},
			},
			"delimiters":       delimitersSchema(),
			"strict_undefined": strictUndefinedSchema(),
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

	// check if template location is not an existing file, then most likely it's an inlined template
	if _, err := os.Stat(ctx.Template.Location); err != nil {
		temp, err := os.CreateTemp("", "")
		if err != nil {
			return fmt.Errorf("failed to create temporary file to hold the inlined template: %s", err)
		}
		defer func() {
			if err := os.Remove(temp.Name()); err != nil {
				log.Printf("[ERROR] failed to remove temporary file")
			}
		}()
		if _, err := temp.Write([]byte(ctx.Template.Location)); err != nil {
			return fmt.Errorf("failed to write template to temporary file: %s", temp.Name())
		}
		if err := temp.Close(); err != nil {
			return fmt.Errorf("failed to close temporary file: %s", temp.Name())
		}
		log.Printf("[WARN] detected inlined template: filesystem call such as the 'include' statement won't work as expected")
		ctx.Template.Location = temp.Name()
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

	return &lib.Context{
		Template: lib.Template{
			Header:   d.Get("header").(string),
			Footer:   d.Get("footer").(string),
			Location: d.Get("template").(string),
		},
		Schemas:       schemas,
		Values:        values,
		Configuration: configuration,
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
		if _, err := os.Stat(stringSchema); err == nil {
			content, err := ioutil.ReadFile(stringSchema)
			if err != nil {
				return nil, fmt.Errorf("failed to read path %s: %s", stringSchema, err)
			}
			schemas[name] = json.RawMessage(content)
		} else {
			schemas[name] = json.RawMessage(stringSchema)
		}
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

	strictUndefined, ok := d.GetOk("strict_undefined")
	if !ok {
		strictUndefined, ok = metaObject["strict_undefined"]
		if !ok {
			strictUndefined = false
		}
	}

	return lib.Configuration{
		StrictUndefined: strictUndefined.(bool),
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
			var data []byte
			if _, err := os.Stat(dataField.(string)); err == nil {
				data, err = ioutil.ReadFile(dataField.(string))
				if err != nil {
					return nil, fmt.Errorf("failed to read path %s: %s", dataField, err)
				}
			} else {
				data = []byte(dataField.(string))
			}
			values[index] = lib.Values{
				Type: kind,
				Data: data,
			}
		}

		return values, nil
	}

	return nil, nil
}
