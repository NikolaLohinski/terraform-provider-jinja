package jinja

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestFilterIfElse(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	true  = {{ "foo" in "foo bar" | ifelse("yes", "no") }}
	false = {{ "baz" in "foo bar" | ifelse("yes", {"value": "no"}) | get("value") }}
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
						true  = yes
						false = no`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterGet(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{%- set key = "field" -%}
	{{- dictionary | get(key) -}}
	{{- dictionary | get("not strict, should just print empty") -}}
	{{- dictionary | get("with default", default=" default") -}}
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
						  field: content
						EOF
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := "content default"
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterGetStrict(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{{- dictionary | get("nope", strict=True) -}}
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
						dictionary: {}
						EOF
					}
				}`),
				ExpectError: regexp.MustCompile("Error: .* item 'nope' not found in: {}"),
			},
		},
	})
}
func TestFilterValues(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{{- numbers | values | sort | join(" > ")  -}}
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
						numbers:
						  one: 1
						  two: 2
						EOF
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := "1 > 2"
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterKeys(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{{- letters | keys | sort | join(" > ")  -}}
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
						letters:
						  a: hey
						  b: bee
						EOF
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := "a > b"
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterTry(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{%- if (foo.bar | try) is undefined %}
	Now you see me without errors!
	{%- endif %}
	{%- if (no | try) %}
	But here you don't!
	{%- endif %}
	{%- if (nested.rendered | try) is defined %}
	You should see this: "{{ nested.rendered }}"
	{%- endif %}
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
						no: false
						nested:
						  rendered: "value"
						EOF
					}
					strict_undefined = true
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`

						Now you see me without errors!
						You should see this: "value"`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterToJSON(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{{ data | keys | tojson }}
	{{ data | tojson }}
	{{ data.array | tojson }}
	{{ data.second | tojson(2) }}
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
						data:
						  first: "one"
						  second:
						    other: "two"
						  third:
						  - field: in array
						  array:
						  - one
						  - two
						EOF
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						["array","first","second","third"]
						{"array":["one","two"],"first":"one","second":{"other":"two"},"third":[{"field":"in array"}]}
						["one","two"]
						{
						  "other": "two"
						}`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterFromJSON(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{%- set var = inlined | fromjson -%}
	{{ var.string }}
	{{ var.list | tojson }}
	{{ var.nested.field }}
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
						inlined: |
						  {
						    "string": "one",
						    "list": ["item"],
						    "nested": {
						      "field": 123
						    }
						  }
						EOF
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						one
						["item"]
						123`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterConcat(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{%- set one = [] | concat(["one"]) -%}
	{%- set multiple = one | concat(["two"], ["three"]) -%}
	{{ one | tojson }}
	{{ multiple | tojson }}
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
						["one"]
						["one","two","three"]`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterSplit(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{%- set path = "one/two/three" | split("/") -%}
	{{ path | tojson }}
	{{ path | last }}
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
						["one","two","three"]
						three`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterAdd(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{%- set object = {"existing": "value", "overridden": 123} | add("other", true) | add("overridden", "new") -%}
	{%- set array = ["one"] | add("two") | add("three") -%}
	{{ object | tojson }}
	{{ array | tojson }}
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
						{"existing":"value","other":true,"overridden":"new"}
						["one","two","three"]`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterFail(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), `{{ "test failing" | fail }}`)
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
				}`),
				ExpectError: regexp.MustCompile(heredoc.Doc(`
				Error: .* test failing
				`)),
			},
		},
	})
}

func TestFilterInsert(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{%- set object = {"existing": "value", "overridden": 123} | insert("other", true) | insert("overridden", "new") -%}
	{{ object | tojson }}
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
						{"existing":"value","other":true,"overridden":"new"}`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterAppend(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{%- set array = ["one"] | append("two") | append("three") -%}
	{{ array | tojson }}
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
						["one","two","three"]`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterFlatten(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{{ [["one"], ["two", "three"]] | flatten | tojson }}
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
						["one","two","three"]`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterUnset(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{{ {"existing": "value", "disappear": 123} | unset("disappear") | tojson }}
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
						{"existing":"value"}`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestFilterFileset(t *testing.T) {
	nestedDir := path.Join(os.TempDir(), "nestedDir")
	must(os.MkdirAll(nestedDir, os.ModePerm))

	firstNested, _, _, remove_first_nested := mustCreateFile("first", "", nestedDir)
	defer remove_first_nested()

	secondNested, _, _, remove_second_nested := mustCreateFile("second", "", nestedDir)
	defer remove_second_nested()

	template, _, dir, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	{%- set paths = "./nestedDir/*" | fileset -%}
	{{- paths | tojson -}}
	`), path.Join(nestedDir, ".."))
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
						expected := `["` + dir + `/nestedDir/` + firstNested + `","` + dir + `/nestedDir/` + secondNested + `"]`
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}
