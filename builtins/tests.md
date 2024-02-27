## The `empty` test

Check if the input is empty. Works on strings, lists and dictionaries.

## The `match` test

Expects a string holding a regular expression to be passed as an argument to match against the input. Returns `true` if the input matches the expression and `false` otherwise. For example:

```
{{ "123" is match("^[0-9]+") }}
```

will evaluate to `True`.