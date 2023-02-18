data "jinja_template" "render" {
  // inlined or path to a template
  // if inlined, filesystem calls won't behave as expected
  template = "${path.module}/src/template.j2"
  context {
    // either yaml or json
    type = "yaml"
    // can be either a path or inline
    data = "${path.module}/src/context.yaml"
  }
  // is a list of either a path or inline, or both
  validation = {
    "schema" = "${path.module}/src/schema.json"
  }

  strict_undefined = false
  header           = "some macro for example"
  footer           = <<-EOF
    some value
  EOF
}
