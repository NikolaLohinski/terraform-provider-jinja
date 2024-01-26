package provider_test

import (
	"os"
	"path"
	"strconv"

	"github.com/MakeNowJust/heredoc"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Context("data \"jinja_template\" \"test\" { ... }", func() {
	var (
		terraformCode = new(string)
	)
	BeforeEach(func() {
		*terraformCode = heredoc.Doc(`
			provider "jinja" {}
		`)
	})

	Context("when the template is including a file", func() {
		const (
			includedContent = "this is the included template"
		)
		var (
			file *os.File
		)
		BeforeEach(func() {
			file = nil
		})
		AfterEach(func() {
			if file != nil {
				os.RemoveAll(file.Name())
			}
		})
		Context("when the include is relative", func() {
			BeforeEach(func() {
				directory := os.TempDir()

				file = MustReturn(os.CreateTemp(directory, ""))
				_ = MustReturn(file.WriteString(includedContent))

				*terraformCode = heredoc.Doc(`
					data "jinja_template" "test" {
						source {
							template  = "{% include './` + path.Base(file.Name()) + `' %}"
							directory = "` + directory + `"
						}
					}
				`)
			})
			itShouldSetTheExpectedResult(terraformCode, includedContent)
		})
		Context("when the include is absolute", func() {
			BeforeEach(func() {
				file = MustReturn(os.CreateTemp("", ""))
				_ = MustReturn(file.WriteString(includedContent))

				*terraformCode = heredoc.Doc(`
					data "jinja_template" "test" {
						source {
							template  = "{% include '` + file.Name() + `' %}"
							directory = path.module
						}
					}
				`)
			})
			itShouldSetTheExpectedResult(terraformCode, includedContent)
		})
		Context("when the include is missing", func() {
			BeforeEach(func() {
				*terraformCode = heredoc.Doc(`
					data "jinja_template" "test" {
						source {
							template  = "{% include '/tmp/does/not/exist' %}"
							directory = path.module
						}
					}
				`)
			})
			itShouldFailToRender(terraformCode, "stat /tmp/does/not: no such file or directory")

			Context("when the include statement has `ignore missing`", func() {
				BeforeEach(func() {
					*terraformCode = heredoc.Doc(`
						data "jinja_template" "test" {
							source {
								template  = "this is just the root template{% include '/tmp/does/not/exist' ignore missing %}"
								directory = path.module
							}
						}
					`)
				})
				itShouldSetTheExpectedResult(terraformCode, "this is just the root template")
			})
		})
	})

	Context("when using `left_strip_blocks`", func() {
		var (
			leftStripBlocks = new(bool)
		)
		JustBeforeEach(func() {
			*terraformCode = heredoc.Doc(`
				data "jinja_template" "test" {
					left_strip_blocks = ` + strconv.FormatBool(*leftStripBlocks) + `
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

			Context("but `left_strip_blocks = true` at the provider level", func() {
				JustBeforeEach(func() {
					*terraformCode = heredoc.Doc(`
						provider jinja {
							left_strip_blocks = true
						}
						` + *terraformCode + `
					`)
				})
				itShouldSetTheExpectedResult(terraformCode, "\t  test")
			})
		})
		Context("when `left_strip_blocks = true`", func() {
			BeforeEach(func() {
				*leftStripBlocks = true
			})
			itShouldSetTheExpectedResult(terraformCode, "test")

			Context("but `left_strip_blocks = false` at the provider level", func() {
				JustBeforeEach(func() {
					*terraformCode = heredoc.Doc(`
						provider jinja {
							left_strip_blocks = false
						}
						` + *terraformCode + `
					`)
				})
				itShouldSetTheExpectedResult(terraformCode, "test")
			})
		})
	})

	Context("when using `trim_blocks`", func() {
		var (
			trimBlocks = new(bool)
		)
		JustBeforeEach(func() {
			*terraformCode = heredoc.Doc(`
				data "jinja_template" "test" {
					trim_blocks = ` + strconv.FormatBool(*trimBlocks) + `
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

			Context("but `trim_blocks = true` at the provider level", func() {
				JustBeforeEach(func() {
					*terraformCode = heredoc.Doc(`
						provider jinja {
							trim_blocks = true
						}
						` + *terraformCode + `
					`)
				})
				itShouldSetTheExpectedResult(terraformCode, "\ntest")
			})
		})
		Context("when `trim_blocks = true`", func() {
			BeforeEach(func() {
				*trimBlocks = true
			})
			itShouldSetTheExpectedResult(terraformCode, "test")

			Context("but `trim_blocks = false` at the provider level", func() {
				JustBeforeEach(func() {
					*terraformCode = heredoc.Doc(`
						provider jinja {
							trim_blocks = false
						}
						` + *terraformCode + `
					`)
				})
				itShouldSetTheExpectedResult(terraformCode, "test")
			})
		})
	})

	Context("when using `strict_undefined`", func() {
		var (
			strictUndefined = new(bool)
		)
		JustBeforeEach(func() {
			*terraformCode = heredoc.Doc(`
				data "jinja_template" "test" {
					strict_undefined = ` + strconv.FormatBool(*strictUndefined) + `
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

			Context("but `strict_undefined = true` at the provider level", func() {
				JustBeforeEach(func() {
					*terraformCode = heredoc.Doc(`
						provider jinja {
							strict_undefined = true
						}
						` + *terraformCode + `
					`)
				})
				itShouldSetTheExpectedResult(terraformCode, "Nothing: ")
			})
		})
		Context("when `strict_undefined = true`", func() {
			BeforeEach(func() {
				*strictUndefined = true
			})
			itShouldFailToRender(terraformCode, ".*attribute 'nope' not found")

			Context("but `strict_undefined = false` at the provider level", func() {
				JustBeforeEach(func() {
					*terraformCode = heredoc.Doc(`
						provider jinja {
							strict_undefined = false
						}
						` + *terraformCode + `
					`)
				})
				itShouldFailToRender(terraformCode, ".*attribute 'nope' not found")
			})
		})
	})

	Context("when setting different `delimiters`", func() {
		BeforeEach(func() {
			*terraformCode = heredoc.Doc(`
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
					delimiters {
 						block_start = "|##"
 						block_end = "##|"
 						variable_start = "<<"
 						variable_end = ">>"
 						comment_start = "[#"
 						comment_end = "#]"
 					}
				}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`

			I am cornered
			but pointy
			
		`))

		Context("when `delimiters` are already set at the provider level", func() {
			JustBeforeEach(func() {
				*terraformCode = heredoc.Doc(`
					provider jinja {
						delimiters {
 							block_start = "[%"
 							block_end = "%]"
 							variable_start = "[["
 							variable_end = "]]"
 							comment_start = "/*"
 							comment_end = "*/"
 						}
					}
					` + *terraformCode + `
				`)
			})
			itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`

				I am cornered
				but pointy

			`))
		})
	})

	Context("when passing a `context`", func() {
		Context("as YAML", func() {
			BeforeEach(func() {
				*terraformCode = heredoc.Doc(`
					data "jinja_template" "test" {
						source {
							template  = <<-EOF
								{{ data.integer }}
								{{ data.string }}
								{{ data.float }}
								{{ data.boolean }}
								{{ data.array[0] }}
								{{ data.object["key"] }}
							EOF
							directory = path.module
						}
						context {
							type = "yaml"
							data = yamlencode({
								data = {
									integer = 123
									string  = "str"
									float 	= 1.23
									boolean = true
									array = [
										"first item"
									]
									object = {
										"key": "value"
									}
								}
							})
						}
					}
				`)
			})
			itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
				123
				str
				1.23
				True
				first item
				value
			`))
		})
		Context("as JSON", func() {
			BeforeEach(func() {
				*terraformCode = heredoc.Doc(`
					data "jinja_template" "test" {
						source {
							template  = <<-EOF
								{{ data.integer }}
								{{ data.string }}
								{{ data.float }}
								{{ data.boolean }}
								{{ data.array[0] }}
								{{ data.object["key"] }}
							EOF
							directory = path.module
						}
						context {
							type = "json"
							data = jsonencode({
								data = {
									integer = 123
									string  = "str"
									float 	= 1.23
									boolean = true
									array = [
										"first item"
									]
									object = {
										"key": "value"
									}
								}
							})
						}
					}
				`)
			})
			itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
				123
				str
				1.23
				True
				first item
				value
			`))
			Context("when the layer has an integer", func() {
				BeforeEach(func() {
					*terraformCode = heredoc.Doc(`
						data "jinja_template" "test" {
							source {
								template  = "{{ value }}"
								directory = path.module
							}
							context {
								type = "json"
								data = "{\"value\": 123}"
							}
						}
					`)
				})
				itShouldSetTheExpectedResult(terraformCode, "123")
			})
		})
		Context("as TOML", func() {
			BeforeEach(func() {
				*terraformCode = heredoc.Doc(`
					data "jinja_template" "test" {
						source {
							template  = <<-EOF
								{{ data.integer }}
								{{ data.string }}
								{{ data.float }}
								{{ data.boolean }}
								{{ data.array[0] }}
								{{ data.object["key"] }}
							EOF
							directory = path.module
						}
						context {
							type = "toml"
							data = <<-EOF
								[data]
								integer = 123
								string = "str"
								float = 1.23
								boolean = true
								array = [
								  "first item"
								]

								[data.object]
								key = "value"
							EOF
						}
					}
				`)
			})
			itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
				123
				str
				1.23
				True
				first item
				value
			`))

		})
		Context("when passing multiple layers", func() {
			BeforeEach(func() {
				*terraformCode = heredoc.Doc(`
				 	data "jinja_template" "test" {
						source {
							template  = <<-EOF
								The service name is {{ name }} in {{ country }}
								{%- if flags %}
								The flags in the file are:
									{%- for flag in flags %}
								- {{ flag }}
									{%- endfor %}
								{% endif %}
							EOF
							directory = path.module
						}
						
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
					}
				`)
			})
			itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
				The service name is overridden in the US
			`))
			Context("when merging an object on an integer", func() {
				BeforeEach(func() {
					*terraformCode = heredoc.Doc(`
						data "jinja_template" "test" {
							source {
								template  = "{{ value.nested }}"
								directory = path.module
							}
							context {
								type = "json"
								data = "{\"value\": 1}"
							}
							context {
								type = "json"
								data = "{\"value\": {\"nested\": 2}}"
							}
						}
					`)
				})
				itShouldSetTheExpectedResult(terraformCode, "2")
			})
		})
	})

	Context("when using `validation`", func() {
		BeforeEach(func() {
			*terraformCode = heredoc.Doc(`
				data "jinja_template" "test" {
					source {
						template  = "The name field is: \"{{ name }}\""
						directory = path.module
					}
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
						schema = <<-EOF
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
						EOF
					}
				}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, "The name field is: \"schema\"")

		Context("when the validation fails", func() {
			BeforeEach(func() {
				*terraformCode = heredoc.Doc(`
					data "jinja_template" "test" {
						source {
							template  = "{{ property }}"
							directory = path.module
						}
						context {
							type = "yaml"
							data = yamlencode({
								property = 123 
							})
						}
						validation = {
							test = <<-EOF
								{
									"type": "object",
									"required": [
										"property"
									],
									"properties": {
										"property": {
											"type": "string"
										}
									}
								}
							EOF
						}
					}
				`)
			})
			itShouldFailToRender(terraformCode, "failed to pass 'test' JSON schema validation: jsonschema: '/property' does not validate with file:///.*#/properties/property/type: expected string, but got number")
		})

		Context("when several schemas are passed", func() {
			var (
				firstSchema  = new(string)
				secondSchema = new(string)
			)
			BeforeEach(func() {
				*firstSchema = heredoc.Doc(`
					<<-EOF
						{
							"type": "object",
							"required": ["name"],
							"properties": {
								"name": {
									"type": "string"
								}
							}
						}
					EOF
				`)
				*secondSchema = heredoc.Doc(`
					jsonencode({
						type       = "object"
						required   = ["other"]
						properties = {
							other = {
								type = "integer"
							}
						}
					})
				`)
			})
			JustBeforeEach(func() {
				*terraformCode = heredoc.Doc(`
					data "jinja_template" "test" {
						source {
							template  = "name: \"{{ name }}\" | other: {{ other }}"
							directory = path.module
						}
						context {
							type = "yaml"
							data = yamlencode({
								name  = "schema"
								other = 123
							})
						}
						validation = {
							first  = ` + *firstSchema + `
							second = ` + *secondSchema + `
						}
					}
				`)
			})
			itShouldSetTheExpectedResult(terraformCode, "name: \"schema\" | other: 123")

			Context("when the first schema fails", func() {
				BeforeEach(func() {
					*firstSchema = heredoc.Doc(`
						<<-EOF
							{
								"type": "object",
								"required": ["name"],
								"properties": {
									"name": {
										"type": "integer"
									}
								}
							}
						EOF
					`)
				})
				itShouldFailToRender(terraformCode, "failed to pass 'first' JSON schema validation: jsonschema: '/name' does not validate with file:///.*#/properties/name/type: expected integer, but got string")
			})
			Context("when the second schema fails", func() {
				BeforeEach(func() {
					*secondSchema = heredoc.Doc(`
						jsonencode({
							type       = "object"
							required   = ["other"]
							properties = {
								other = {
									type = "string"
								}
							}
						})
					`)
				})
				itShouldFailToRender(terraformCode, ".*failed to pass 'second' JSON schema validation: jsonschema: '/other' does not validate with file:///.*#/properties/other/type: expected string, but got number")
			})
		})
	})

	Context("when using legacy fields", func() {
		Context("when using the `template` field", func() {
			BeforeEach(func() {
				*terraformCode = heredoc.Doc(`
					data "jinja_template" "test" {
						template = "{{ 'hello !' | capitalize }}"
					}
				`)
			})
			itShouldSetTheExpectedResult(terraformCode, "Hello !")

			Context("and the template contains a relative include statement", func() {
				var (
					included *os.File
				)
				AfterEach(func() {
					os.RemoveAll(included.Name())
				})
				BeforeEach(func() {
					included = MustReturn(os.CreateTemp(MustReturn(os.Getwd()), ""))
					MustReturn(included.WriteString("How are you ?"))
					*terraformCode = heredoc.Doc(`
						data "jinja_template" "test" {
							template = "{{ 'hello !' | capitalize }} {% include './` + path.Base(included.Name()) + `' %}"
						}
					`)
				})
				itShouldSetTheExpectedResult(terraformCode, "Hello ! How are you ?")
			})

			Context("when the provided string is a path", func() {
				var (
					file      *os.File
					directory = new(string)
				)
				AfterEach(func() {
					if file != nil {
						os.RemoveAll(file.Name())
					}
				})
				BeforeEach(func() {
					*directory = os.TempDir()
					file = MustReturn(os.CreateTemp(*directory, ""))
					MustReturn(file.WriteString("{{ 'hello !' | capitalize }}"))
					*terraformCode = heredoc.Doc(`
						data "jinja_template" "test" {
							template = "` + file.Name() + `"
						}
					`)
				})
				itShouldSetTheExpectedResult(terraformCode, "Hello !")

				Context("and the template contains a relative include statement", func() {
					var (
						included *os.File
					)
					AfterEach(func() {
						os.RemoveAll(included.Name())
					})
					BeforeEach(func() {
						included = MustReturn(os.CreateTemp(*directory, ""))
						MustReturn(included.WriteString("How are you ?"))
						MustReturn(file.WriteString(" {% include './" + path.Base(included.Name()) + "' %}"))
					})
					itShouldSetTheExpectedResult(terraformCode, "Hello ! How are you ?")
				})
			})

			Context("and setting the `source` block", func() {
				BeforeEach(func() {
					*terraformCode = heredoc.Doc(`
						data "jinja_template" "test" {
							template = "{{ 'hello !' | capitalize }}"
							source {
								template  = "other template"
								directory = path.module
							}
						}
					`)
				})
				itShouldFailToRender(terraformCode, `These attributes cannot be configured together: \[source,template\]`)
			})
		})

		Context("when using the `header` field", func() {
			BeforeEach(func() {
				*terraformCode = heredoc.Doc(`
					data "jinja_template" "test" {
						header   = "Hello"
						template = "World"
					}
				`)
			})
			itShouldSetTheExpectedResult(terraformCode, "Hello\nWorld")

			Context("and setting the `source` block", func() {
				BeforeEach(func() {
					*terraformCode = heredoc.Doc(`
						data "jinja_template" "test" {
							header = "head"
							source {
								template  = "other template"
								directory = path.module
							}
						}
					`)
				})
				itShouldFailToRender(terraformCode, `These attributes cannot be configured together: \[source,header\]`)
			})
		})

		Context("when using the `footer` field", func() {
			BeforeEach(func() {
				*terraformCode = heredoc.Doc(`
					data "jinja_template" "test" {
						template = "Hello"
						footer   = "World"
					}
				`)
			})
			itShouldSetTheExpectedResult(terraformCode, "Hello\nWorld")

			Context("and setting the `source` block", func() {
				BeforeEach(func() {
					*terraformCode = heredoc.Doc(`
						data "jinja_template" "test" {
							footer = "foot"
							source {
								template  = "other template"
								directory = path.module
							}
						}
					`)
				})
				itShouldFailToRender(terraformCode, `These attributes cannot be configured together: \[source,footer\]`)
			})
		})
	})

	Context("when neither `source` nor `template` was set", func() {
		BeforeEach(func() {
			*terraformCode = heredoc.Doc(`
					data "jinja_template" "test" {
					}
				`)
		})
		itShouldFailToRender(terraformCode, `.*At least one of these attributes must be configured: \[source,template\]`)
	})
})
