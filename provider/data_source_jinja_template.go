package jinja

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	json "github.com/json-iterator/go"
	"github.com/noirbizarre/gonja"
	"github.com/noirbizarre/gonja/exec"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v2"
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
		Read:        render,
		Description: "The jinja_template data source renders a jinja template with a given template with possible JSON schema validation of the context",
		Schema: map[string]*schema.Schema{
			"template": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to the jinja template to render",
			},
			"schema": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Either inline or a path to a JSON schema to validate the context",
			},
			"context": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Context to use while rendering the template",
				MaxItems:    1,
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
			"delimiters": delimitersSchema(),
			"result": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Rendered template with the given context",
			},
		},
	}
}

func render(d *schema.ResourceData, meta interface{}) error {
	if err := applyDelimiters(d, meta); err != nil {
		return fmt.Errorf("failed to apply delimiters: %s", err)
	}

	template, err := parse_template(d)
	if err != nil {
		return fmt.Errorf("failed to parse template: %s", err)
	}

	context, err := parse_context(d)
	if err != nil {
		return fmt.Errorf("failed to parse context: %s", err)
	}

	if err := validate_schema(d, context); err != nil {
		return fmt.Errorf("failed to validate context against schema: %s", err)
	}

	result, err := template.Execute(context)
	if err != nil {
		return fmt.Errorf("failed to execute template: %s", err)
	}

	if err := d.Set("result", result); err != nil {
		return fmt.Errorf("failed to output result: %s", err)
	}

	hash_function := sha256.New()
	hash_function.Write([]byte(result))

	d.SetId(base64.URLEncoding.EncodeToString(hash_function.Sum(nil)))

	return nil
}

func parse_template(d *schema.ResourceData) (*exec.Template, error) {
	template := d.Get("template").(string)
	if err := gonja.DefaultLoader.SetBaseDir(path.Dir(template)); err != nil {
		return nil, fmt.Errorf("failed to set base directory to template folder: %s", err)
	}

	tpl, err := gonja.FromFile(path.Base(template))
	if err != nil {
		return nil, fmt.Errorf("error reading template: %s", err)
	}

	return tpl, nil
}

func parse_context(d *schema.ResourceData) (map[string]interface{}, error) {
	context := make(map[string]interface{})

	context_blocks, ok := d.GetOk("context")
	if ok {
		contexts, ok := context_blocks.([]interface{})
		if !ok {
			return nil, fmt.Errorf("context blocks are invalid: %s", context_blocks)
		}
		if len(contexts) != 1 {
			return nil, fmt.Errorf("context block if defined must be unique: %s", context_blocks)
		}
		kind := d.Get("context.0.type")
		data := d.Get("context.0.data")

		if _, err := os.Stat(data.(string)); err == nil {
			content, err := ioutil.ReadFile(data.(string))
			if err != nil {
				return nil, fmt.Errorf("failed to read path %s: %s", data, err)
			}
			data = string(content)
		}

		switch strings.ToLower(kind.(string)) {
		case "json":
			if err := json.Unmarshal([]byte(data.(string)), &context); err != nil {
				return nil, fmt.Errorf("failed to JSON unmarshal context: %v", data)
			}
		case "yaml":
			if err := yaml.Unmarshal([]byte(data.(string)), &context); err != nil {
				return nil, fmt.Errorf("failed to YAML unmarshal context: %v", data)
			}
		default:
			return nil, fmt.Errorf("provided context has an unsupported type: %v", kind)
		}
	}

	return context, nil
}

func validate_schema(d *schema.ResourceData, context map[string]interface{}) error {
	schemaField, ok := d.GetOk("schema")

	if ok {
		schema := schemaField.(string)
		if _, err := os.Stat(schema); err == nil {
			content, err := ioutil.ReadFile(schema)
			if err != nil {
				return fmt.Errorf("failed to read path %s: %s", schema, err)
			}
			schema = string(content)
		}
		validator, err := jsonschema.CompileString("schema.json", schema)
		if err != nil {
			return fmt.Errorf("failed to compile JSON schema %s: %s", err, schema)
		}

		bytes, err := json.Marshal(context)
		if err != nil {
			return fmt.Errorf("failed to marshal context to JSON: %v", err)
		}
		var payload interface{}
		if err := json.Unmarshal(bytes, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal context back from JSON: %v", err)
		}

		if err := validator.Validate(payload); err != nil {
			return fmt.Errorf("failed to pass JSON schema validation: %s", err)
		}
	}

	return nil
}

func applyDelimiters(d *schema.ResourceData, meta interface{}) error {
	var delimiters map[string]interface{}

	provider_delimiters := meta.(map[string]interface{})

	delimiters_block, ok := d.GetOk("delimiters.0")
	if !ok {
		delimiters = provider_delimiters
	} else {
		delimiters = delimiters_block.(map[string]interface{})
		for name, delimiter := range delimiters {
			if default_value, ok := default_delimiters[name]; ok && delimiter != default_value {
				continue
			}
			delimiters[name] = provider_delimiters[name]
		}
	}

	gonja.DefaultEnv.Config.BlockStartString = delimiters["block_start"].(string)
	gonja.DefaultEnv.Config.BlockEndString = delimiters["block_end"].(string)
	gonja.DefaultEnv.Config.VariableStartString = delimiters["variable_start"].(string)
	gonja.DefaultEnv.Config.VariableEndString = delimiters["variable_end"].(string)
	gonja.DefaultEnv.Config.CommentStartString = delimiters["comment_start"].(string)
	gonja.DefaultEnv.Config.CommentEndString = delimiters["comment_end"].(string)

	return nil
}
