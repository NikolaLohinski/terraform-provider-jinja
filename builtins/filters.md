## The `abspath` filter

The `abspath` filter takes a string containing a filesystem path and converts it to an absolute path. If the path is not absolute, it is resolved according to the directory of the template it is called from.

## The `add`, `append` and `insert` filters

The `insert` filter is meant to add a key value pair to a dict. It expects a key and value to operate.

The `append` filter adds an item to a list. It expects one value to work.

The `add` filter is just the combination of both with type reflection to decide what to do.

```
{%- set object = {"existing": "value", "overridden": 123} | insert("other", true) | add("overridden", "new") -%}
{%- set array = ["one"] | add("two") | append("three") -%}
{{ object | tojson }}
{{ array | tojson }}
```
Will render into:
```
{"existing":"value","other":true,"overridden":"new"}
["one","two","three"]
```

## The `bool` filter

The `bool` filter is meant to cast a string, an int, a bool or `nil` to a boolean value. _Truthful_ values are:
* Any string once lowercased such as `"on"`, `"yes"` `"1"` or `"true"`
* `1` as an integer or `1.0` as a float
* A `True` boolean

_False_ values are:
* Any string once lowercased such as `"off"`, `"no"` `"0"` or `"false"` 
* An empty string
* `0` as an integer or `0.0` as a float
* A `False` boolean
* A `nil` or `None` value

Any other type passed will cause the `bool` filter to fail.


## The `concat` filter

The `concat` filter is meant to concatenate lists together and can take any number of lists to append together.

```
{%- set array = ["one"] | concat(["two"],["three"]) -%}
{{ array | tojson }}
```
Will render into:
```
["one","two","three"]
```

## The `distinct` filter

The `distinct` filter takes any list of elements and returns a new list with duplicates removed.

```
{{ [1, 1, 2, 3, 2, 1] | distinct }}
```
Will render into:
```
[1, 2, 3]
```

## The `basename` filter

The `basename` filter takes a string containing a filesystem path and returns the last portion from it.

```
{{ "path/to/folder/file.txt" | basename }}
```
Will render into:
```
file.txt
```

## The `dirname` filter

The `dirname` filter takes a string containing a filesystem path and removes the last portion from it.

```
{{ "path/to/folder/file.txt" | dirname }}
```
Will render into:
```
path/to/folder
```

## The `env` filter

The `env` filter retrieves a environment variable. It will fail if the environment variable is not found but take an additional `default` keyword parameter to set a default value as a fallback.

```
{{ "USER" | env(default="root") }}
```

## The `fail` filter

The `fail` filter is meant to error out explicitly in a given place of the template.

```
{{ "error message to output" | fail }}
```

## The `file` filter

The `file` filter is meant to load a local file into a variable. It works with both absolute and relative (to the place it's called from) paths. The `file` filter does not process the file as a template but simply loads the contents of it.

```
{% set content = "some/path" | file %}
{{ content }}
```

## The `fileset` filter

The `fileset` filter is a filesystem filter meant to be used with the `include` statement to dynamically include files.
It supports glob patterns (using `*`) and double glob patterns (using `**`) in paths, and operates relatively to the
folder that contains the file being rendered.

```
{% for path in "folder/*" | fileset %}
{% include path %}
{% endfor %}
```

## The `frombase64` filter

The `frombase64` filter is meant to decode a string encoded in base64.

```
{{ 'SGVsbG8gV29ybGQh' | frombase64 }}
```

Will render into:

```
Hello World!
```

## The `fromcsv` filter

The `fromcsv` filter decodes a string containing CSV-formatted data and produces a list of maps representing that data.

CSV stands for Comma-Separated Values, an encoding format for tabular data. There are many variants of CSV, but this function implements the format defined in [RFC 4180](https://datatracker.ietf.org/doc/html/rfc4180).

The first line of the CSV data is interpreted as a "header" row: the values given are used as the keys in the resulting maps. Each subsequent line becomes a single map in the resulting list, matching the keys from the header row with the given values by index. All lines in the file must contain the same number of fields, or this function will produce an error.

```
{{ 'a,b,c\n1,2,3\n4,5,6' | fromcsv }}
```

Will render into:

```
[{'a': '1', 'b': '2', 'c': '3'}, {'a': '4', 'b': '5', 'c': '6'}]
```

## The `fromjson` filter

The `fromjson` filter is meant to parse a JSON string into a useable object.

```
{%- set object = "{ \"nested\": { \"field\": \"value\" } }" | fromjson -%}
{{ object.nested.field }}
```
Will render into:
```
value
```

## The `fromtoml` filter

The `fromtoml` filter is meant to parse a TOML string into a useable object.

```
{%- set object = "[nested]\nfield = \"value\"" | fromtoml -%}
{{ object.nested.field }}
```
Will render into:
```
value
```

## The `fromtfvars` filter

The `fromtfvars` filter is meant to parse a string with terraform variable definitions as formalized in [`tfvars` files](https://developer.hashicorp.com/terraform/language/values/variables#variable-definitions-tfvars-files) into a useable object.

```
{%- set tfvars = 'foo = { bar = "test" }' | fromtfvars -%}
{{ tfvars.foo.bar }}
```
Will render into:
```
test
```

## The `fromyaml` filter

The `fromyaml` filter is meant to parse a YAML string into a useable object.

```
{%- set object = "nested:\n  field: value\n" | fromyaml -%}
{{ object.nested.field }}
```
Will render into:
```
value
```

## The `get` filter

The `get` filter helps getting an item in a map with a dynamic key:

```
{% set tac = "key" %}
{{ tic | get(tac) }}
```
With the following YAML context:
```
tic:
  key: toe
```
Will render into:
```
toe
```

The filter has the following keyword attributes:

- `strict`: a boolean to fail if the key is missing from the map. Defaults to `False` ;
- `default`: any value to pass as default if the key is not found. This takes precedence over the `strict` attribute if defined. Defaults to nil value ;

## The `ifelse` filter

The `ifelse` filter is meant to perform ternary conditions as follows:

```
true is {{ "foo" in "foo bar" | ifelse("yes", "no") }}
false is {{ "yolo" in "foo bar" | ifelse("yes", "no") }}
```
Which will render into:
```
true is yes
false is no
```

## The `keys` filter

The `keys` filter is meant to get the keys of a map as a list:

```
{{ letters | keys | sort | join(" > ") }}
```
With the following YAML context:
```
letters:
  a: hey
  b: bee
  c: see
```
Will render into:
```
a > b > c
```

Note that the order of keys is not guaranteed as there is no ordering in Golang maps.

## The `match` filter

Expects a string holding a regular expression to be passed as an argument to match against the input. Returns `true` if the input matches the expression and `false` otherwise. For example:

```
{{ "123 is a number" | match("^[0-9]+") }}
```

will render as:

```
True
```

## The `merge` filter

Use this filter to deeply merge two dictionaries together. In case of duplicate keys, the value from the pipeline dictionary take precedence unless `override` is set to `True`.

```
{{ {'fizz': 'buzz', 'foo': 'bar'} | merge({'foo': 'test'}, override=True) }}
```

will return:

```
{'fizz': 'buzz', 'foo': 'test'}
```

## The `sha1`, `sha256`, `sha512` and `md5` filters

Classic hashing algorithms that work on strings as depicted in:

- https://pkg.go.dev/crypto/sha1
- https://pkg.go.dev/crypto/sha256
- https://pkg.go.dev/crypto/sha512
- https://pkg.go.dev/crypto/md5

For example:

```
{{ 'test' | md5 }}
```
Will render into:
```
098f6bcd4621d373cade4e832627b4f6
```

## The `split` filter

The `split` filter is meant to split a string into a list of strings using a given delimiter.

```
{%- set array = "one/two/three" | split("/") -%}
{{ array | tojson }}
```
Will render into:
```
["one","two","three"]
```

## The `tobase64` filter

The `tobase64` filter is meant to encode a string to a base64 representation.

```
{{ 'Hello World!' | tobase64 }}
```

Will render into:

```
SGVsbG8gV29ybGQh
```

## The `totoml` filter

The `totoml` filter is meant to render a given object as TOML.

```
{%- set object = "{ \"nested\": { \"field\": \"value\" } }" | fromjson -%}
{{ object | totoml }}
```
Will render into:
```
[nested]
field = "value"
```

## The `toyaml` filter


The `toyaml` filter is meant to render a given object as YAML. It takes an optional argument called `indent` to
specify the indentation to apply to the result, which defaults to `2` spaces.

```
{%- set object = "{ \"nested\": { \"field\": \"value\" } }" | fromjson -%}
{{ object | toyaml }}
```
Will render into:
```
nested:
  field: value
```

## The `try` filter

The `try` filter is meant to gracefully evaluate an expression. It returns an `undefined` value if the passed expression is undefined or throws an error. Otherwise, it returns the value passed in the context of the pipeline.

```
{%- if (empty.missing | try) is undefined -%}
	Now you see {{ value | try }}!
{%- endif -%}
```
With the following YAML context and `strict_undefined` set to `true`:
```
empty: {}
value: me
```
Will render into:
```
Now you see me!
```

This is useful when `strict_undefined = true` is set but you need to handle a missing key without throwing errors in a given template ;

## The `values` filter

The `values` filter is meant to get the values of a map as a list:

```
{{ numbers | values | sort | join(" > ") }}
```
With the following YAML context:
```
numbers:
  first: 1
  second: 2
  third: 3
```
Will render into:
```
1 > 2 > 3
```

