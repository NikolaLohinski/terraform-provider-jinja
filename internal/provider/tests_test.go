package provider_test

import (
	"github.com/MakeNowJust/heredoc"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Context("tests", func() {
	var (
		template      = new(string)
		context       = new(string)
		terraformCode = new(string)
	)
	BeforeEach(func() {
		*template = ""
		*context = ""
	})
	JustBeforeEach(func() {
		*terraformCode = heredoc.Doc(`
			data "jinja_template" "test" {
				source {
					template = <<-EOF
					` + *template + `
					EOF
					directory = path.module
				}
				context {
					type = "json"
					data = jsonencode({
						` + *context + `
					})
				}
			}
		`)
	})
	Context("empty", func() {
		BeforeEach(func() {
			*template = `{{- input is empty -}}`
		})
		Context("when the input is an array", func() {
			BeforeEach(func() {
				*context = `input = ["foo"]`
			})
			itShouldSetTheExpectedResult(terraformCode, "False")
			Context("and it is empty", func() {
				BeforeEach(func() {
					*context = `input = []`
				})
				itShouldSetTheExpectedResult(terraformCode, "True")
			})
		})
		Context("when the input is a string", func() {
			BeforeEach(func() {
				*context = `input = "foo"`
			})
			itShouldSetTheExpectedResult(terraformCode, "False")
			Context("and it is empty", func() {
				BeforeEach(func() {
					*context = `input = ""`
				})
				itShouldSetTheExpectedResult(terraformCode, "True")
			})
		})
		Context("when the input is a dict", func() {
			BeforeEach(func() {
				*context = `input = { key = "value" }`
			})
			itShouldSetTheExpectedResult(terraformCode, "False")
			Context("and it is empty", func() {
				BeforeEach(func() {
					*context = `input = {}`
				})
				itShouldSetTheExpectedResult(terraformCode, "True")
			})
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- ("thrown" | fail) is empty -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
		Context("when the input is invalid", func() {
			BeforeEach(func() {
				*context = `input = true`
			})
			itShouldFailToRender(terraformCode, "invalid call to test 'empty': True is neither a list, a dict nor a string")
		})
	})
})
