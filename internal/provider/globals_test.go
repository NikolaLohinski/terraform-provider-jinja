package provider_test

import (
	"os"
	"path"

	"github.com/MakeNowJust/heredoc"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	. "github.com/onsi/ginkgo/v2"
	"github.com/openconfig/goyang/pkg/indent"
)

var _ = Context("globals", func() {
	var (
		template      = new(string)
		directory     = new(string)
		terraformCode = new(string)
	)
	BeforeEach(func() {
		*template = ""
		*directory = "${path.module}"
	})
	JustBeforeEach(func() {
		*terraformCode = heredoc.Doc(`
			data "jinja_template" "test" {
				source {
					template = <<-EOF
					` + indent.String("\t\t", *template) + `
					EOF
					directory = "` + *directory + `"
				}
			}
		`)
	})
	Context("abspath", Ordered, func() {
		BeforeAll(func() {
			*directory = os.TempDir()

			Must(os.MkdirAll(path.Join(*directory, "abspath"), 0700))

			MustReturn(os.Create(path.Join(*directory, "abspath", "file.txt"))).Close()

			*template = `{{- abspath("./abspath/file.txt") -}}`
		})
		AfterAll(func() {
			os.RemoveAll(*directory)
		})
		itShouldSetTheExpectedResult(terraformCode, path.Join(os.TempDir(), "abspath", "file.txt"))
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- abspath(true) -}}`
			})
			itShouldFailToRender(terraformCode, "wrong signature for function 'abspath'")
		})
	})
	Context("uuid", func() {
		BeforeEach(func() {
			*template = `{{- uuid() -}}`
		})
		It("should render the expected content", func() {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testProviderFactory,
				Steps: []resource.TestStep{
					{
						Config: *terraformCode,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttrSet("data.jinja_template.test", "id"),
							resource.TestCheckResourceAttrWith("data.jinja_template.test", "result", func(got string) error {
								_, err := uuid.Parse(got)
								return err
							}),
						),
					},
				},
			})
		})
		Context("when an input is passed", func() {
			BeforeEach(func() {
				*template = `{{- uuid(true) -}}`
			})
			itShouldFailToRender(terraformCode, "wrong signature for function 'uuid'")
		})
	})
})
