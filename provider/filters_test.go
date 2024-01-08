package jinja_test

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
			itShouldFailToRender(terraformCode, "Error: .* filter 'add' was passed 'True' which is neither a dict nor a list")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | add("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'append' was passed 'True' which is not a list")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | append("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'basename' was passed 'True' which is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | basename -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'bool' failed to cast: \\[\\]")
		})
		Context("when the input is a dict", func() {
			BeforeEach(func() {
				*template = `{{- {} | bool -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* filter 'bool' failed to cast: {}")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | bool -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'concat' was passed 'True' which is not a list")
		})
		Context("when the argument is not a list", func() {
			BeforeEach(func() {
				*template = `{{- [] | concat(true) -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* 1st argument passed to filter 'concat' is not a list: True")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | concat([]) -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'dir|dirname' was passed 'True' which is not a string")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | dirname -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
		})
	})
	Context("fail", func() {
		BeforeEach(func() {
			*template = `{{- "returned error from tests" | fail -}}`
		})
		itShouldFailToRender(terraformCode, "Error: .*: returned error from tests")
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
				itShouldFailToRender(terraformCode, "Error: .* failed to read file at path /does/not/exist")
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
				itShouldFailToRender(terraformCode, "Error: .* failed to read file at path .+/does/not/exist")
			})
		})
		Context("when the input is not a string", func() {
			BeforeEach(func() {
				*template = `{{- true | file -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* filter 'file' was passed 'True' which is not a string")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'fileset' was passed 'True' which is not a string")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'flatten' was passed 'True' which is not a list")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | flatten -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'fromjson' was passed 'True' which is not a string or is empty")
		})
		Context("when the input is an empty string", func() {
			BeforeEach(func() {
				*template = `{{- "" | fromjson -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* filter 'fromjson' was passed '' which is not a string or is empty")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | fromjson -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'fromyaml' was passed 'True' which is not a string or is empty")
		})
		Context("when the input is an empty string", func() {
			BeforeEach(func() {
				*template = `{{- "" | fromyaml -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* filter 'fromyaml' was passed '' which is not a string or is empty")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | fromyaml -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'fromtoml' was passed 'True' which is not a string or is empty")
		})
		Context("when the input is an empty string", func() {
			BeforeEach(func() {
				*template = `{{- "" | fromtoml -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* filter 'fromtoml' was passed '' which is not a string or is empty")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | fromtoml -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'get' was passed 'True' which is not a dict")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | get("nothing") -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
		})
	})
	Context("ifelse", func() {
		BeforeEach(func() {
			*template = `{{- name in "foo bar" | ifelse("name is in 'foo bar'", "name is not in 'foo bar'") -}}`
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
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
				*template = `{{- [] | insert("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* filter 'insert' was passed '\\[\\]' which is not a dict")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | insert("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'keys' was passed '\\[\\]' which is not a dict")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | keys -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'match' was passed '\\[\\]' which is not a string")
		})
		Context("when the argument is not a string", func() {
			BeforeEach(func() {
				*template = `{{- "does not matter" | match(True) -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* 1st argument passed to filter 'match' is not a string: True")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | match("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'split' was passed '\\[\\]' which is not a string")
		})
		Context("when the argument is not a string", func() {
			BeforeEach(func() {
				*template = `{{- "does not matter" | split(True) -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* 1st argument passed to filter 'split' is not a string: True")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | split("does not matter") -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'unset' was passed '\\[\\]' which is not a dict")
		})
		Context("when the argument is not a string", func() {
			BeforeEach(func() {
				*template = `{{- {} | unset(True) -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* 1st argument passed to filter 'unset' is not a string: True")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | unset -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
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
			itShouldFailToRender(terraformCode, "Error: .* filter 'values' was passed '\\[\\]' which is not a dict")
		})
		Context("when the input is an error", func() {
			BeforeEach(func() {
				*template = `{{- "thrown" | fail | values -}}`
			})
			itShouldFailToRender(terraformCode, "Error: .* thrown")
		})
	})
})
