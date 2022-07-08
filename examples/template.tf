data "jinja_template" "example" {
  template = "${path.module}/src/template.j2"
  context {
    type = "yaml"
    // data can be either a path or inline
    data = "${path.module}/src/context.yaml"
  }
  schema = "${path.module}/src/schema.json"
}
