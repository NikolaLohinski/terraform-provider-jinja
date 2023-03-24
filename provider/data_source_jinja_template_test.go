package jinja

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Args:
// - prefix [default is "tmp-"]
// - content [default is ""]
// - directory [default is current working directory]
// Returns:
// - name of the file
// - content of the file
// - path to containing folder
// - callable to delete the file.
func mustCreateFile(args ...string) (string, string, string, func()) {
	if len(args) > 3 || len(args) == 0 {
		panic("mustCreateFile takes up to 3 arguments: prefix, content, directory")
	}
	var prefix, content, directory string
	switch len(args) {
	case 3:
		directory = args[2]
		fallthrough
	case 2:
		content = args[1]
		fallthrough
	case 1:
		prefix = args[0]
	case 0:
		prefix = "tmp-"
	}

	file, err := ioutil.TempFile(directory, prefix)
	if err != nil {
		panic(err)
	}
	_, err = file.WriteString(content)
	if err != nil {
		panic(err)
	}

	return path.Base(file.Name()), content, path.Dir(file.Name()), func() { os.Remove(file.Name()) }
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func TestJinjaTemplateSimple(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{% if "foo" in "foo bar" %}
	show within loop!
	{% endif %}
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`

						show within loop!

						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithInclude(t *testing.T) {
	nested, expected, dir, remove_nested := mustCreateFile("nested", heredoc.Doc(`
	I am nested !
	`))
	defer remove_nested()

	template, _, _, remove_template := mustCreateFile(t.Name(), `{% include "`+nested+`" %}`, dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateOtherDelimiters(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	|##- if "foo" in "foo bar" ##|
	I am cornered
	|##- endif ##|
	<< "but pointy" >>
	[#- "and can be invisible!" -#]
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					delimiters {
						block_start = "|##"
						block_end = "##|"
						variable_start = "<<"
						variable_end = ">>"
						comment_start = "[#"
						comment_end = "#]"
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`

						I am cornered
						but pointy

						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithContextJSON(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	This is a very nested {{ top.middle.bottom.field }}
	And this is an integer: {{ integer }}
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "json"
						data = jsonencode({
							top = {
								middle = {
									bottom = {
										field = "surprise!"
									}
								}
							}
							integer = 123
						})
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						This is a very nested surprise!
						And this is an integer: 123
						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "merged_context", func(got string) error {
						expected := heredoc.Doc(`{"integer":123,"top":{"middle":{"bottom":{"field":"surprise!"}}}}`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithContextYAML(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	The service name is {{ name }}
	{%- if flags %}
	The flags in the file are:
		{%- for flag in flags %}
	- {{ flag }}
		{%- endfor %}
	{% endif %}
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							name = "NATO"
							flags = [
								"fr",
								"us",
							]
						})
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						The service name is NATO
						The flags in the file are:
						- fr
						- us

						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithMultipleContext(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	The service name is {{ name }} in {{ country }}
	{%- if flags %}
	The flags in the file are:
		{%- for flag in flags %}
	- {{ flag }}
		{%- endfor %}
	{% endif %}
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							name = "NATO"
							country = "the US"
							flags = [
								"fr",
								"us",
							]
						})
					}
					context {
						type = "json"
						data = jsonencode({
							name = "overridden"
							flags = []
						})
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						The service name is overridden in the US
						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateOtherDelimitersAtProviderLevel(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	[%- if "foo" in "foo bar" %]
	I am cornered
	[%- endif %]
	<< "but pointy" >>
	|#- "and can be invisible!" -#|
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				provider "jinja" {
					delimiters {
						variable_start = "<<"
						variable_end = ">>"
						comment_start = "|#"
						comment_end = "#|"
					}
				}
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					delimiters {
						block_start = "[%"
						block_end = "%]"
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`

						I am cornered
						but pointy

						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithPathContext(t *testing.T) {
	ctx, _, dir, remove_context := mustCreateFile("nested", heredoc.Doc(`
	name: remote-context
	`))
	defer remove_context()

	template, _, _, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	The name field is: "{{ name }}"
	`), dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = "` + path.Join(dir, ctx) + `"
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						The name field is: "remote-context"
						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithSchema(t *testing.T) {
	schema, _, dir, remove_schema := mustCreateFile("nested", heredoc.Doc(`
	{
		"type": "object",
		"required": [
			"name",
			"other"
		],
		"properties": {
			"name": {
				"type": "string"
			},
			"other": {
				"type": "object",
				"required": ["foo"],
				"properties": {
					"foo": {
						"type": "string"
					}
				}
			}
	
		}
	}
	`))
	defer remove_schema()

	template, _, _, remove_template := mustCreateFile(t.Name(), `The name field is: "{{ name }}"`, dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							name = "schema"
							other = {
								"foo" = "bar"
							}
						})
					}
					schema = "` + path.Join(dir, schema) + `"
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := `The name field is: "schema"`
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithValidation(t *testing.T) {
	schema, _, dir, remove_schema := mustCreateFile("nested", heredoc.Doc(`
	{
		"type": "object",
		"required": [
			"name",
			"other"
		],
		"properties": {
			"name": {
				"type": "string"
			},
			"other": {
				"type": "object",
				"required": ["foo"],
				"properties": {
					"foo": {
						"type": "string"
					}
				}
			}
	
		}
	}
	`))
	defer remove_schema()

	template, _, _, remove_template := mustCreateFile(t.Name(), `The name field is: "{{ name }}"`, dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							name = "schema"
							other = {
								"foo" = "bar"
							}
						})
					}
					validation = {
						test = "` + path.Join(dir, schema) + `"
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := `The name field is: "schema"`
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithInlineSchema(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), `The name field is: "{{ name }}"`)
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							name = "schema"
						})
					}
					schema = <<-EOF
					{
						"type": "object",
						"required": [
							"name"
						],
						"properties": {
							"name": {
								"type": "string"
							}
						}
					}
					EOF
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := `The name field is: "schema"`
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithValidationThatFails(t *testing.T) {
	schema, _, dir, remove_schema := mustCreateFile("nested", heredoc.Doc(`
	{
		"type": "object",
		"required": [
			"name"
		],
		"properties": {
			"name": {
				"type": "string"
			}
		}
	}
	`))
	defer remove_schema()

	template, _, _, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	The name field is: "{{ name }}"
	`), dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({})
					}
					validation = {
						test = "` + path.Join(dir, schema) + `"
					}
				}`),
				ExpectError: regexp.MustCompile("failed to pass 'test' JSON schema validation: jsonschema: '' does not validate with .*: missing properties: 'name'"),
			},
		},
	})
}
func TestJinjaTemplateWithSchemaThatFails(t *testing.T) {
	schema, _, dir, remove_schema := mustCreateFile("nested", heredoc.Doc(`
	{
		"type": "object",
		"required": [
			"name"
		],
		"properties": {
			"name": {
				"type": "string"
			}
		}
	}
	`))
	defer remove_schema()

	template, _, _, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	The name field is: "{{ name }}"
	`), dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({})
					}
					schema = "` + path.Join(dir, schema) + `"
				}`),
				ExpectError: regexp.MustCompile("failed to pass '1st' JSON schema validation: jsonschema: '' does not validate with .*: missing properties: 'name'"),
			},
		},
	})
}

func TestJinjaTemplateWithFooter(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), "body")
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					footer = "footer"
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						body
						footer`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithHeaderMacro(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{{ verbose(true) }}
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					header = <<-EOF
						{%- macro verbose(trigger) -%}
						{%- if trigger -%}
						This should be visible!
						{%- endif -%}
						{%- endmacro -%}
					EOF
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						This should be visible!
						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestGonjaForLoop(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{%- for key, value in dictionary %}
	{{ key }} = {{ value }}
	{%- endfor %}
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = <<-EOF
						dictionary:
						  foo: bar
						  tic: toc
						EOF
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`

						foo = bar
						tic = toc
						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestGonjaNoneValue(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{{ None is undefined }}
	{{ nil is defined }}
	{% set var = None %}{{ var }}
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					strict_undefined = true
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						True
						False

						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithIntegerInYAMLContext(t *testing.T) {
	template, _, dir, remove_template := mustCreateFile(t.Name(), `The int field is: {{ integer }}`)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							integer = 123
						})
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := `The int field is: 123`
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithSchemaAndSchemas(t *testing.T) {
	schema, _, dir, remove_schema := mustCreateFile("nested", heredoc.Doc(`
	{
		"type": "object",
		"required": [
			"name",
			"other"
		],
		"properties": {
			"name": {
				"type": "string"
			},
			"other": {
				"type": "object",
				"required": ["foo"],
				"properties": {
					"foo": {
						"type": "string"
					}
				}
			}
		}
	}
	`))
	defer remove_schema()

	template, _, _, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	The name field is: "{{ name }}"
	`), dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							name = "schema"
							other = {
								"foo" = "bar"
							}
						})
					}
					schema = "` + path.Join(dir, schema) + `"
					schemas = ["` + path.Join(dir, schema) + `"]
				}`),
				ExpectError: regexp.MustCompile("Error: Conflicting configuration arguments"),
			},
		},
	})
}

func TestJinjaTemplateWithMultipleSchemas(t *testing.T) {
	schema, _, dir, remove_schema := mustCreateFile("nested", heredoc.Doc(`
	{
		"type": "object",
		"required": [
			"name"
		],
		"properties": {
			"name": {
				"type": "string"
			}
		}
	}
	`))
	defer remove_schema()

	template, _, _, remove_template := mustCreateFile(t.Name(), `The name field is: "{{ name }}"`, dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							name = "schema"
							other = {
								"foo" = "bar"
							}
						})
					}
					schemas = [
						"` + path.Join(dir, schema) + `",
						<<-EOF
						{
							"type": "object",
							"required": [
								"other"
							],
							"properties": {
								"other": {
									"type": "object",
									"required": ["foo"],
									"properties": {
										"foo": {
											"type": "string"
										}
									}
								}
							}
						}
						EOF
					]
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := `The name field is: "schema"`
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithMultipleSchemasWhenOneIsFailing(t *testing.T) {
	schema, _, dir, remove_schema := mustCreateFile("nested", heredoc.Doc(`
	{
		"type": "object",
		"required": [
			"name"
		],
		"properties": {
			"name": {
				"type": "string"
			}
		}
	}
	`))
	defer remove_schema()

	template, _, _, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	The name field is: "{{ name }}"
	`), dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							name = 123
							other = {
								"foo" = "bar"
							}
						})
					}
					schemas = [
						"` + path.Join(dir, schema) + `",
						<<-EOF
						{
							"type": "object",
							"required": [
								"other"
							],
							"properties": {
								"other": {
									"type": "object",
									"required": ["foo"],
									"properties": {
										"foo": {
											"type": "string"
										}
									}
								}
							}
						}
						EOF
					]
				}`),
				ExpectError: regexp.MustCompile("failed to pass '1st' JSON schema validation: jsonschema: '/name' does not validate with .*#/properties/name/type: expected string, but got number"),
			},
		},
	})
}

func TestJinjaTemplateWithMultipleSchemasWhenMultipleAreFailing(t *testing.T) {
	schema, _, dir, remove_schema := mustCreateFile("nested", heredoc.Doc(`
	{
		"type": "object",
		"required": [
			"name"
		],
		"properties": {
			"name": {
				"type": "string"
			}
		}
	}
	`))
	defer remove_schema()

	template, _, _, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	The name field is: "{{ name }}"
	`), dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							name = 123
							other = "wrong"
						})
					}
					schemas = [
						"` + path.Join(dir, schema) + `",
						<<-EOF
						{
							"type": "object",
							"required": [
								"other"
							],
							"properties": {
								"other": {
									"type": "object",
									"required": ["foo"],
									"properties": {
										"foo": {
											"type": "string"
										}
									}
								}
							}
						}
						EOF
					]
				}`),
				ExpectError: regexp.MustCompile(heredoc.Doc(`
				\s+failed to pass '1st' JSON schema validation: jsonschema: '/name' does not validate with .*#/properties/name/type: expected string, but got number
				\s+failed to pass '2nd' JSON schema validation: jsonschema: '/other' does not validate with .*#/properties/other/type: expected object, but got string
				`)),
			},
		},
	})
}

func TestJinjaTemplateWithStrictUndefined(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	This is a very nested {{ dict.missing }}

	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "json"
						data = jsonencode({
							top = {
								other = "value"
							}
						})
					}
					strict_undefined = true
				}`),
				ExpectError: regexp.MustCompile(heredoc.Doc(`
				Error: .* Unable to evaluate dict.missing: attribute 'missing' not found
				`)),
			},
		},
	})
}

func TestJinjaTemplateWithProviderStrictUndefined(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	This is a very nested {{ dict.missing }}

	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				provider "jinja" {
					strict_undefined = true
				}
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "json"
						data = jsonencode({
							top = {
								other = "value"
							}
						})
					}
				}`),
				ExpectError: regexp.MustCompile(heredoc.Doc(`
				Error: .* Unable to evaluate dict.missing: attribute 'missing' not found
				`)),
			},
		},
	})
}

func TestJinjaTemplateWithStrictUndefinedAtRootLevel(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	This is {{ missing }}

	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "json"
						data = jsonencode({
							top = {
								other = "value"
							}
						})
					}
					strict_undefined = true
				}`),
				ExpectError: regexp.MustCompile(heredoc.Doc(`
				Error: .* Unable to evaluate name "missing"
				`)),
			},
		},
	})
}

func TestJinjaWhenGonjaHangsForever(t *testing.T) {
	template, _, dir, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	{{ "known bug of gonja which hangs forever if there's an unclosed string }}
	`))
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
				}`),
				ExpectError: regexp.MustCompile("Error: failed to render context: rendering timed out after .*: known possible reasons for timeouts are:"),
			},
		},
	})
}

func TestJinjaWhenGonjaPanics(t *testing.T) {
	template, _, dir, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	{{ nil | panic }}
	`))
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
				}`),
				ExpectError: regexp.MustCompile("Error: failed to render context: failed to execute template: a runtime error led gonja to panic: panic filter was called"),
			},
		},
	})
}

func TestJinjaTemplateInlined(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = <<-EOF
					{%- if "foo" in "foo bar" -%}
					Look at me!
					{%- endif -%}
					EOF
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := `Look at me!`
						if expected != got {
							return fmt.Errorf("\nexpected:\n=========\n%s\n=========\ngot:\n==========\n%s\n==========", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}
