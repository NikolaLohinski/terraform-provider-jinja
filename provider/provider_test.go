package jinja_test

import (
	"strconv"

	"github.com/MakeNowJust/heredoc"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Context("provider \"jinja\" { ... }", func() {
	var (
		terraformCode = new(string)
	)
	BeforeEach(func() {
		*terraformCode = heredoc.Doc(`
			provider "jinja" {}
		`)
	})

	Context("when using `left_strip_blocks`", func() {
		var (
			leftStripBlocks = new(bool)
		)
		JustBeforeEach(func() {
			*terraformCode = heredoc.Doc(`
			 	provider "jinja" {
					left_strip_blocks = ` + strconv.FormatBool(*leftStripBlocks) + `
				}
				data "jinja_template" "test" {
					source {
						template  = "\t  {% set _ = 'foo' %}test"
						directory = path.module
					}
				}
			`)
		})
		Context("when `left_strip_blocks = false`", func() {
			BeforeEach(func() {
				*leftStripBlocks = false
			})
			itShouldSetTheExpectedResult(terraformCode, "\t  test")
		})
		Context("when `left_strip_blocks = true`", func() {
			BeforeEach(func() {
				*leftStripBlocks = true
			})
			itShouldSetTheExpectedResult(terraformCode, "test")
		})
	})

	Context("when using `trim_blocks`", func() {
		var (
			trimBlocks = new(bool)
		)
		JustBeforeEach(func() {
			*terraformCode = heredoc.Doc(`
				provider "jinja" {
					trim_blocks = ` + strconv.FormatBool(*trimBlocks) + `
				}
				data "jinja_template" "test" {
					source {
						template  = <<-EOF
							{% if "foo" in "foo bar" %}
							test
							{%- endif -%}
						EOF
						directory = path.module
					}
				}
			`)
		})
		Context("when `trim_blocks = false`", func() {
			BeforeEach(func() {
				*trimBlocks = false
			})
			itShouldSetTheExpectedResult(terraformCode, "\ntest")
		})
		Context("when `trim_blocks = true`", func() {
			BeforeEach(func() {
				*trimBlocks = true
			})
			itShouldSetTheExpectedResult(terraformCode, "test")
		})
	})

	Context("when using `strict_undefined`", func() {
		var (
			strictUndefined = new(bool)
		)
		JustBeforeEach(func() {
			*terraformCode = heredoc.Doc(`
				provider "jinja" {
					strict_undefined = ` + strconv.FormatBool(*strictUndefined) + `
				}
				data "jinja_template" "test" {
					context {
						type = "json"
						data = jsonencode({ dict = { yes = true }})
					}
					source {
						template  = "Nothing: {{ dict.nope }}"
						directory = path.module
					}
				}
			`)
		})
		Context("when `strict_undefined = false`", func() {
			BeforeEach(func() {
				*strictUndefined = false
			})
			itShouldSetTheExpectedResult(terraformCode, "Nothing: ")
		})
		Context("when `strict_undefined = true`", func() {
			BeforeEach(func() {
				*strictUndefined = true
			})
			itShouldFailToRender(terraformCode, ".* Unable to evaluate dict.nope: attribute 'nope' not found")
		})
	})
	Context("when setting different `delimiters`", func() {
		BeforeEach(func() {
			*terraformCode = heredoc.Doc(`
				provider "jinja" {
 					delimiters {
 						block_start = "|##"
 						block_end = "##|"
 						variable_start = "<<"
 						variable_end = ">>"
 						comment_start = "[#"
 						comment_end = "#]"
 					}
				}
				data "jinja_template" "test" {
					source {
						template  = <<-EOF
							|##- if "foo" in "foo bar" ##|
							I am cornered
							|##- endif ##|
							<< "but pointy" >>
							[# "and can be invisible!" #]
						EOF
						directory = path.module
					}
				}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
			
			I am cornered
			but pointy

		`))
	})
})
