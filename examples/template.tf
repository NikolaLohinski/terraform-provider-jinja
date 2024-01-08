data "jinja_template" "render" {
  context {
    type = "yaml"
    data = "${path.module}/src/context.yaml"
  }
  source {
    template  = file("${path.module}/src/template.j2")
    directory = "${path.module}/src"
  }
  validation = {
    "schema" = "${path.module}/src/schema.json"
  }
  strict_undefined  = false
  left_strip_blocks = false
  trim_blocks       = false
}
