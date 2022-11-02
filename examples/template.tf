data "jinja_template" "render" {
  // must be a path to resolve any nested templates includes
  template = "${path.module}/src/template.j2"
  context {
    // either yaml or json
    type = "yaml"
    // can be either a path or inline
    data = "${path.module}/src/context.yaml"
  }
  // can be either a path or inline
  schema = "${path.module}/src/schema.json"

  header = "some macro for example"
  footer = <<-EOF
    some value
  EOF
}
