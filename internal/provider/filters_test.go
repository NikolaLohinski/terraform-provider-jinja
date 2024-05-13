package provider_test

import (
	"fmt"
	"os"
	"path"

	"github.com/MakeNowJust/heredoc"

	. "github.com/onsi/ginkgo/v2"
	"github.com/openconfig/goyang/pkg/indent"
)

var _ = Context("filters", func() {
	var (
		template      = new(string)
		context       = new(string)
		directory     = new(string)
		terraformCode = new(string)
	)
	BeforeEach(func() {
		*template = ""
		*context = ""
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
				context {
					type = "json"
					data = jsonencode({
						` + indent.String("\t\t", *context) + `
					})
				}
			}
		`)
	})
	Context("add", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
				{%- set object = {"existing": "value", "overridden": 123} | add("other", true) | add("overridden", "new") -%}
				{%- set array = ["one"] | add("two") | add("three") -%}
				{{ object | tojson }}
				{{ array | tojson }}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
			{"existing":"value","other":true,"overridden":"new"}
			["one","two","three"]
			
		`))
		Context("when the input is neither a list nor a dict", func() {
			BeforeEach(func() {
				*template = `{{- true | add("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "True is neither a dict nor a list")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | add("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("append", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
				{%- set array = ["one"] | append("two") | append("three") -%}
				{{- array | tojson -}}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, `["one","two","three"]`)
		Context("when the input is not a list", func() {
			BeforeEach(func() {
				*template = `{{- true | append("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a list")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | append("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("basename", func() {
		BeforeEach(func() {
			*template = `{{- "test/folder/base" | basename -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, `base`)
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- true | basename -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | basename -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("bool", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
			 	{#- just to eat whitespace -#}
				"true": {{ "true" | bool }}
				"True": {{ "True" | bool }}
				"yes": {{ "yes" | bool }}
				"Yes": {{ "Yes" | bool }}
				"on": {{ "on" | bool }}
				"On": {{ "On" | bool }}
				"TrUe": {{ "TrUe" | bool }}
				"1": {{ "1" | bool }}
				"false": {{ "false" | bool }}
				"False": {{ "False" | bool }}
				"no": {{ "no" | bool }}
				"No": {{ "No" | bool }}
				"off": {{ "off" | bool }}
				"Off": {{ "Off" | bool }}
				"FaLSe": {{ "FaLSe" | bool }}
				"0": {{ "0" | bool }}
				"": {{ "" | bool }}
				1: {{ 1 | bool }}
				1.0: {{ 1.0 | bool }}
				0: {{ 0 | bool }}
				0.0: {{ 0 | bool }}
				False: {{ False | bool }}
				True: {{ True | bool }}
				false: {{ false | bool }}
				true: {{ true | bool }}
				nil: {{ nil | bool }}
				None: {{ None | bool }}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
			"true": True
			"True": True
			"yes": True
			"Yes": True
			"on": True
			"On": True
			"TrUe": True
			"1": True
			"false": False
			"False": False
			"no": False
			"No": False
			"off": False
			"Off": False
			"FaLSe": False
			"0": False
			"": False
			1: True
			1.0: True
			0: False
			0.0: False
			False: False
			True: True
			false: False
			true: True
			nil: False
			None: False
			
		`))
		Context("when the input is a list", func() {
			BeforeEach(func() {
				*template = `{{- [] | bool -}}`
			})
			itShouldFailToRender(terraformCode, "failed to cast: \\[\\]")
		})
		Context("when the input is a dict", func() {
			BeforeEach(func() {
				*template = `{{- {} | bool -}}`
			})
			itShouldFailToRender(terraformCode, "failed to cast: {}")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | bool -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("concat", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
				{%- set one = [] | concat(["one"]) -%}
				{%- set multiple = one | concat(["two"], ["three"]) -%}
				{{ one | tojson }}
				{{ multiple | tojson }}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
			["one"]
			["one","two","three"]

		`))
		Context("when the input is not a list", func() {
			BeforeEach(func() {
				*template = `{{- true | concat([]) -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a list")
		})
		Context("when the argument is not a list", func() {
			BeforeEach(func() {
				*template = `{{- [] | concat(true) -}}`
			})
			itShouldFailToRender(terraformCode, "1st argument True is not a list")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | concat([]) -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("dirname", func() {
		BeforeEach(func() {
			*template = `{{- "test/folder/base" | dirname -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, `test/folder`)
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- true | dirname -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | dirname -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("fail", func() {
		BeforeEach(func() {
			*template = `{{- "returned error from tests" | fail -}}`
		})
		itShouldFailToRender(terraformCode, "returned error from tests")
	})
	Context("file", func() {
		const (
			fileContent = "{{ this looks like a template but should not be templated }}"
		)
		var (
			file *os.File
		)
		BeforeEach(func() {
			file = MustReturn(os.CreateTemp(os.TempDir(), ""))
			MustReturn(file.WriteString(fileContent))
		})
		AfterEach(func() {
			os.RemoveAll(file.Name())
		})
		Context("when the path is absolute", func() {
			BeforeEach(func() {
				*template = `{{- "` + file.Name() + `" | file -}}`
			})
			itShouldSetTheExpectedResult(terraformCode, fileContent)
			Context("but the file does not exist", func() {
				BeforeEach(func() {
					*template = `{{- "/does/not/exist" | file -}}`
				})
				itShouldFailToRender(terraformCode, "failed to read file at path /does/not/exist")
			})
		})
		Context("when the path is relative to the current directory", func() {
			BeforeEach(func() {
				*directory = os.TempDir()
				*template = `{{- "./` + path.Base(file.Name()) + `" | file -}}`
			})
			itShouldSetTheExpectedResult(terraformCode, fileContent)
			Context("but the file does not exist", func() {
				BeforeEach(func() {
					*template = `{{- "./does/not/exist" | file -}}`
				})
				itShouldFailToRender(terraformCode, "failed to read file at path .+/does/not/exist")
			})
		})
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- true | file -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
	})
	Context("fileset", Ordered, func() {
		BeforeAll(func() {
			*directory = os.TempDir()

			Must(os.MkdirAll(path.Join(*directory, "fileset", "folder"), 0700))

			MustReturn(os.Create(path.Join(*directory, "fileset", "root.txt"))).Close()
			MustReturn(os.Create(path.Join(*directory, "fileset", "folder", "nested.txt"))).Close()

			*template = `{{- "./fileset/**/*.txt" | fileset -}}`
		})
		AfterAll(func() {
			os.RemoveAll(*directory)
		})
		itShouldSetTheExpectedResult(terraformCode, fmt.Sprintf("['%s', '%s']", os.TempDir()+"/fileset/root.txt", os.TempDir()+"/fileset/folder/nested.txt"))
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- true | fileset -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
	})
	Context("flatten", func() {
		BeforeEach(func() {
			*template = `{{- [["one"], ["two", "three"]] | flatten -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, `['one', 'two', 'three']`)
		Context("when the input is not a list", func() {
			BeforeEach(func() {
				*template = `{{- true | flatten -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a list")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | flatten -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("fromjson", func() {
		BeforeEach(func() {
			*context = heredoc.Doc(`
				inlined = <<-EOF
					{
					  "string": "one",
					  "list": ["item"],
					  "nested": {
					    "field": 123
					  }
					}
				EOF
			`)
			*template = heredoc.Doc(`
				{%- set var = inlined | fromjson -%}
				{{ var.string }}
				{{ var.list }}
				{{ var.nested.field }}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
			one
			['item']
			123

		`))
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- true | fromjson -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a non-empty string")
		})
		Context("when the input is an empty string", func() {
			BeforeEach(func() {
				*template = `{{- "" | fromjson -}}`
			})
			itShouldFailToRender(terraformCode, " is not a non-empty string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | fromjson -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("fromyaml", func() {
		BeforeEach(func() {
			*context = heredoc.Doc(`
				inlined = <<-EOF
					---
					string: one
					list:
					  - item
					nested:
					  field: 123
				EOF
			`)
			*template = heredoc.Doc(`
				{%- set var = inlined | fromyaml -%}
				{{ var.string }}
				{{ var.list }}
				{{ var.nested.field }}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
			one
			['item']
			123

		`))
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- true | fromyaml -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a non-empty string")
		})
		Context("when the input is an empty string", func() {
			BeforeEach(func() {
				*template = `{{- "" | fromyaml -}}`
			})
			itShouldFailToRender(terraformCode, " is not a non-empty string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | fromyaml -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("fromtoml", func() {
		BeforeEach(func() {
			*context = heredoc.Doc(`
				inlined = <<-EOF
				 	root = "string"
					[section]
					bool = true
					[section.nested]
					value = 123
				EOF
			`)
			*template = heredoc.Doc(`
				{%- set var = inlined | fromtoml -%}
				{{ var.root }}
				{{ var.section.bool }}
				{{ var.section.nested.value }}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
			string
			True
			123

		`))
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- true | fromtoml -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a non-empty string")
		})
		Context("when the input is an empty string", func() {
			BeforeEach(func() {
				*template = `{{- "" | fromtoml -}}`
			})
			itShouldFailToRender(terraformCode, " is not a non-empty string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | fromtoml -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("get", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
				{%- set dictionary = {"field": "content"} -%}
				{%- set key = "field" -%}
				{{- dictionary | get(key) -}}
				{{- dictionary | get("not strict, should just print empty") -}}
				{{- dictionary | get("with default", default=" default") -}}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, "content default")
		Context("when the input is not a dict", func() {
			BeforeEach(func() {
				*template = `{{- true | get("nothing") -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a dict")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | get("nothing") -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("ifelse", func() {
		BeforeEach(func() {
			*template = `{{- (name in "foo bar") | ifelse("name is in 'foo bar'", "name is not in 'foo bar'") -}}`
		})
		Context("first branch", func() {
			BeforeEach(func() {
				*context = `name = "foo"`
			})
			itShouldSetTheExpectedResult(terraformCode, "name is in 'foo bar'")
		})
		Context("second branch", func() {
			BeforeEach(func() {
				*context = `name = "yolo"`
			})
			itShouldSetTheExpectedResult(terraformCode, "name is not in 'foo bar'")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | ifelse("does not", "matter") -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("insert", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
				{%- set object = {"existing": "value", "overridden": 123} | insert("other", true) | insert("overridden", "new") -%}
				{{ object | tojson }}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
			{"existing":"value","other":true,"overridden":"new"}
			
		`))
		Context("when the input is not a dict", func() {
			BeforeEach(func() {
				*template = `{{- [] | insert("does not", "matter") -}}`
			})
			itShouldFailToRender(terraformCode, "\\[\\] is not a dict")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | insert("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("keys", func() {
		BeforeEach(func() {
			*template = `{{- {"a": "hey", "b": "bee", "c": "see"} | keys -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, "['a', 'b', 'c']")
		Context("when the input is not a dict", func() {
			BeforeEach(func() {
				*template = `{{- [] | keys -}}`
			})
			itShouldFailToRender(terraformCode, "\\[\\] is not a dict")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | keys -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("match", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
				{{- "foo" | match("bar") }}
				{{ "123" | match("^[0-9]+$") }}
				{{ "nope" | match("^[0-9]+$") }}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
			False
			True
			False
			
		`))
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- [] | match("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "\\[\\] is not a string")
		})
		Context("when the argument is not a string", func() {
			BeforeEach(func() {
				*template = `{{- "does not matter" | match(True) -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | match("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("split", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
				{{- "root/folder/file" | split("/") }}
				{{ "c,s,v" | split(",") }}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
			['root', 'folder', 'file']
			['c', 's', 'v']
			
		`))
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- [] | split("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "\\[\\] is not a string")
		})
		Context("when the argument is not a string", func() {
			BeforeEach(func() {
				*template = `{{- "does not matter" | split(True) -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | split("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("totoml", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
				{{- {} | totoml }}
				{{ ['one'] | totoml }}
				{{ True | totoml }}
				{{ 'test' | totoml }}
				{{ {"root": {"nested": { "foo": "bar" }}} | totoml }}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`

			['one']
			true
			'test'
			[root]
			[root.nested]
			foo = 'bar'
			
			
		`))
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | totoml -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("toyaml", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
				{{- {} | toyaml }}
				{{ {"simple": {"nested": "field"}} | toyaml }}
				{{ ["array"] | toyaml }}
				{{ "string" | toyaml }}
				{{ 42 | toyaml }}
				{{ {"indented": {4: "spaces"}} | toyaml(indent=4) }}
				{{ {"indented": {4: "spaces"}} | toyaml(indent=4) }}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
			{}

			simple:
			  nested: field

			- array

			string

			42

			indented:
			    4: spaces

			indented:
			    4: spaces
			
			
		`))
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | toyaml -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("try", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
				{%- if (fail | try) is undefined -%}
				Now you see me without errors!
				{%- endif -%}
				{%- if (no | try) -%}
				But here you don't!
				{%- endif -%}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, "Now you see me without errors!")
	})
	Context("unset", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
				{{- {"key": "will disappear", "still": "there"} | unset("key") }}
				{{ {} | unset("nope") }}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, heredoc.Doc(`
			{'still': 'there'}
			{}

		`))
		Context("when the input is not a dict", func() {
			BeforeEach(func() {
				*template = `{{- [] | unset("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "\\[\\] is not a dict")
		})
		Context("when the argument is not a string", func() {
			BeforeEach(func() {
				*template = `{{- {} | unset(True) -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | unset -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("values", func() {
		BeforeEach(func() {
			*template = `{{- {"one": 1, "two": 2, "three": 3} | values -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, "[1, 2, 3]")
		Context("when the input is not a dict", func() {
			BeforeEach(func() {
				*template = `{{- [] | values -}}`
			})
			itShouldFailToRender(terraformCode, "\\[\\] is not a dict")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | values -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("abspath", Ordered, func() {
		BeforeAll(func() {
			*directory = os.TempDir()

			Must(os.MkdirAll(path.Join(*directory, "abspath"), 0700))

			MustReturn(os.Create(path.Join(*directory, "abspath", "file.txt"))).Close()

			*template = `{{- "./abspath/file.txt" | abspath -}}`
		})
		AfterAll(func() {
			os.RemoveAll(*directory)
		})
		itShouldSetTheExpectedResult(terraformCode, path.Join(os.TempDir(), "abspath", "file.txt"))
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- true | abspath -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
	})
	Context("distinct", func() {
		BeforeEach(func() {
			*template = `{{- ["a", "b", "a", "c", "d", "b"] | distinct -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, "['a', 'b', 'c', 'd']")
		Context("when the input is a list of integers", func() {
			BeforeEach(func() {
				*template = `{{- [1,1,1,2,3,3,2,3] | distinct -}}`
			})
			itShouldSetTheExpectedResult(terraformCode, "[1, 2, 3]")
		})
		Context("when the input is a list of objects", func() {
			BeforeEach(func() {
				*template = `{{- [{"one": 1}, {"two": 2}, {"one": 1}] | distinct -}}`
			})
			itShouldSetTheExpectedResult(terraformCode, "[{'one': 1}, {'two': 2}]")
		})
		Context("when the input is a list of lists", func() {
			BeforeEach(func() {
				*template = `{{- [[1,2], [1,"two"], [1,"two"]] | distinct -}}`
			})
			itShouldSetTheExpectedResult(terraformCode, "[[1, 2], [1, 'two']]")
		})
		Context("when the input is not a list", func() {
			BeforeEach(func() {
				*template = `{{- {} | distinct -}}`
			})
			itShouldFailToRender(terraformCode, "{} is not a list")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | distinct -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("env", func() {
		BeforeEach(func() {
			Must(os.Setenv("FOO", "BAR"))

			*template = `{{- "FOO" | env -}}`
		})
		AfterEach(func() {
			Must(os.Unsetenv("FOO"))
		})
		itShouldSetTheExpectedResult(terraformCode, "BAR")
		Context("when the environment variable does not exist", func() {
			BeforeEach(func() {
				*template = `{{- "NOPE" | env -}}`
			})
			itShouldFailToRender(terraformCode, "failed to get 'NOPE' environment variable without default")
			Context("but a default was defined", func() {
				BeforeEach(func() {
					*template = `{{- "NOPE" | env(default="BAR") -}}`
				})
				itShouldSetTheExpectedResult(terraformCode, "BAR")
			})
		})
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- True | env -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | env -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("tobase64", func() {
		BeforeEach(func() {
			*template = `{{- "test" | tobase64 -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, "dGVzdA==")
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- True | tobase64 -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | tobase64 -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("frombase64", func() {
		BeforeEach(func() {
			*template = `{{- "dGVzdA==" | frombase64 -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, "test")
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- True | frombase64 -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is not a valid base64 encoding", func() {
			BeforeEach(func() {
				*template = `{{- "❌" | frombase64 -}}`
			})
			itShouldFailToRender(terraformCode, "failed to decode '❌' from base64")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | frombase64 -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("fromcsv", func() {
		BeforeEach(func() {
			*template = `{{- "a,b,c\n1,2,3\n4,5,6" | fromcsv -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, "[{'a': '1', 'b': '2', 'c': '3'}, {'a': '4', 'b': '5', 'c': '6'}]")
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- True | fromcsv -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is not a valid csv", func() {
			BeforeEach(func() {
				*template = `{{- "nope,;s\nssss" | fromcsv -}}`
			})
			itShouldFailToRender(terraformCode, "record on line 2: wrong number of fields")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | fromcsv -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("fromtfvars", func() {
		BeforeEach(func() {
			*template = `{{- 'foo = "bar"\nobj = {\ntest = 123\n}' | fromtfvars -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, "{'foo': 'bar', 'obj': {'test': 123}}")
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- True | fromtfvars -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is not a valid tfvars file", func() {
			BeforeEach(func() {
				*template = `{{- "nope,;ssss" | fromtfvars -}}`
			})
			itShouldFailToRender(terraformCode, "failed to parse 'nope,;ssss' as tfvars")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | fromtfvars -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("sha1", func() {
		BeforeEach(func() {
			*template = `{{- 'test' | sha1 -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- True | sha1 -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | sha1 -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("sha256", func() {
		BeforeEach(func() {
			*template = `{{- 'test' | sha256 -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08")
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- True | sha256 -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | sha256 -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("sha512", func() {
		BeforeEach(func() {
			*template = `{{- 'test' | sha512 -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, "ee26b0dd4af7e749aa1a8ee3c10ae9923f618980772e473f8819a5d4940e0db27ac185f8a0e1d5f84f88bc887fd67b143732c304cc5fa9ad8e6f57f50028a8ff")
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- True | sha512 -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | sha512 -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("md5", func() {
		BeforeEach(func() {
			*template = `{{- 'test' | md5 -}}`
		})
		itShouldSetTheExpectedResult(terraformCode, "098f6bcd4621d373cade4e832627b4f6")
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- True | md5 -}}`
			})
			itShouldFailToRender(terraformCode, "invalid call to filter 'md5': True is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | md5 -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})
	Context("merge", func() {
		BeforeEach(func() {
			*template = heredoc.Doc(`
				{{- {"foo": "bar", "still": "untouched"} | merge({"foo": "fizz", "buzz": "bar"}) -}}
			`)
		})
		itShouldSetTheExpectedResult(terraformCode, "{'buzz': 'bar', 'foo': 'bar', 'still': 'untouched'}")
		Context("when merging with overrides", func() {
			BeforeEach(func() {
				*template = heredoc.Doc(`
					{{- {"foo": "bar", "still": "untouched"} | merge({"foo": "fizz", "buzz": "bar"}, override=True) -}}
				`)
			})
			itShouldSetTheExpectedResult(terraformCode, "{'buzz': 'bar', 'foo': 'fizz', 'still': 'untouched'}")
		})
		Context("when the input is not a dict", func() {
			BeforeEach(func() {
				*template = `{{- [] | merge({}) -}}`
			})
			itShouldFailToRender(terraformCode, "\\[\\] is not a dict")
		})
		Context("when the argument is not a dict", func() {
			BeforeEach(func() {
				*template = `{{- {} | merge(True) -}}`
			})
			itShouldFailToRender(terraformCode, "True is not a dict")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | merge({}) -}}`
			})
			itShouldFailToRender(terraformCode, "thrown")
		})
	})

})
