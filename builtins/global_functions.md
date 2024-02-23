## The `abspath` function

The `abspath` function takes a `path` string containing a filesystem path and converts it to an absolute path. If the path is not absolute, it is resolved according to the directory of the template it is called from.

```
{{ abspath('./path/to/file') }}
```

## The `basename` function

The `basename` function takes a string containing a filesystem path and returns the last portion from it.

```
{{ dirname("path/to/folder/file.txt") }}
```
Will render into:
```
file.txt
```

## The `dirname` function

The `dirname` function takes a string containing a filesystem path and removes the last portion from it.

```
{{ dirname("path/to/folder/file.txt") }}
```
Will render into:
```
path/to/folder
```

## The `uuid` function

The `uuid` function generates a UUID based on [RFC 4122](https://datatracker.ietf.org/doc/html/rfc4122) and DCE 1.1: Authentication and Security Services as implemented in https://pkg.go.dev/github.com/google/uuid.

```
{{ uuid() }}
```

## The `env` function

The `env` function retrieves a environment variable. It will fail if the environment variable is not found but take an additional `default` keyword parameter to set a default value as a fallback.

```
{{ env("USER", default="root") }}
```
## The `file` function

The `file` function is meant to load a local file. It works with both absolute and relative (to the place it's called from) paths. The `file` function does not process the file as a template but simply loads the contents of it.

```
{{ file("some/path") }}
```

## The `fileset` function

The `fileset` function is an operator to explore filesystem trees. It supports glob patterns (using `*`) and double glob patterns (using `**`) in paths, and operates relatively to the
folder that contains the file it is called from.

```
{% for path in fileset("folder/*") %}
{% path %}
{% endfor %}
```
