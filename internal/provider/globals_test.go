package provider_test

import (
	"fmt"
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
			itShouldFailToRender(terraformCode, "invalid call to function 'abspath': failed to validate argument 'path': True is not a string")
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
			itShouldFailToRender(terraformCode, "invalid call to function 'uuid': received 1 unexpected positional argument")
		})
	})
	Context("env", func() {
		BeforeEach(func() {
			Must(os.Setenv("FOO", "bar"))

			*template = `{{- env("FOO") -}}`
		})
		AfterEach(func() {
			Must(os.Unsetenv("FOO"))
		})
		itShouldSetTheExpectedResult(terraformCode, "bar")
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- env(true) -}}`
			})
			itShouldFailToRender(terraformCode, "invalid call to function 'env': failed to validate argument 'name': True is not a string")
		})
		Context("when the variable does not exist", func() {
			BeforeEach(func() {
				*template = `{{- env("BAR") -}}`
			})
			itShouldFailToRender(terraformCode, "failed to get 'BAR' environment variable without default")
			Context("and a default is provided", func() {
				BeforeEach(func() {
					*template = `{{- env("BAR", "foo") -}}`
				})
				itShouldSetTheExpectedResult(terraformCode, "foo")
			})
		})
	})
	Context("file", Ordered, func() {
		BeforeAll(func() {
			*directory = os.TempDir()

			Must(os.MkdirAll(path.Join(*directory, "file"), 0700))

			f := MustReturn(os.Create(path.Join(*directory, "file", "file.txt")))
			f.WriteString("test")
			Must(f.Close())

			*template = `{{- file("./file/file.txt") -}}`
		})
		AfterAll(func() {
			os.RemoveAll(*directory)
		})
		itShouldSetTheExpectedResult(terraformCode, "test")
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- file(true) -}}`
			})
			itShouldFailToRender(terraformCode, "invalid call to function 'file': failed to validate argument 'path': True is not a string")
		})
	})
	Context("fileset", Ordered, func() {
		BeforeAll(func() {
			*directory = os.TempDir()

			Must(os.MkdirAll(path.Join(*directory, "fileset", "folder"), 0700))

			MustReturn(os.Create(path.Join(*directory, "fileset", "root.txt"))).Close()
			MustReturn(os.Create(path.Join(*directory, "fileset", "folder", "nested.txt"))).Close()

			*template = `{{- fileset("./fileset/**/*.txt") -}}`
		})
		AfterAll(func() {
			os.RemoveAll(*directory)
		})
		itShouldSetTheExpectedResult(terraformCode, fmt.Sprintf("['%s', '%s']", os.TempDir()+"/fileset/root.txt", os.TempDir()+"/fileset/folder/nested.txt"))
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- fileset(True) -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
	})
	Context("dirname", func() {
		BeforeEach(func() {
			*template = `{{- dirname("test/folder/base") -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, `test/folder`)
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- dirname(True) -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
	})
})
