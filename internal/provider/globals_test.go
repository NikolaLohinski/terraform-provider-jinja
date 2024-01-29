package provider_test

import (
	"os"
	"path"

	"github.com/MakeNowJust/heredoc"

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
})
