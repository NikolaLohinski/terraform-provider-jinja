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
