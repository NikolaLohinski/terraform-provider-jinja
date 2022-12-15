package jinja

import (
	"fmt"
	"path"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestFilterIfElse(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	true  = {{ "foo" in "foo bar" | ifelse("yes", "no") }}
	false = {{ "baz" in "foo bar" | ifelse("yes", "no") }}

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
						false = no
						`)
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
						expected := "content"
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
