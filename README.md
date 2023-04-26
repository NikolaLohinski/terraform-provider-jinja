<div align="center">
<img src="./misc/logo.png" width="200"/>
<h1><code>terraform-provider-jinja</code></h1>
</div>

A `terraform` provider that makes it possible to render [Jinja](https://jinja.palletsprojects.com/) templates within a `terraform` project.

## Requirements

- Install `make` with [the official instructions](https://www.gnu.org/software/make/) ;
- Install go `>= 1.20` by following the [official documentation](https://go.dev/doc/install).

| ⚠️ You also need to install `golangci-lint` by following the [official instructions](https://golangci-lint.run/usage/install/#local-installation) to be able to run `make lint`. |
| --- |

The provider can be installed locally, using:

```shell
make install
```

See the [`Makefile`](./Makefile) for more details on what it means.

## Example

```hcl
provider "jinja" {
  strict_undefined = true
}

data "jinja_template" "render" {
  template = "${path.module}/template.j2"
  context {
    type = "yaml"
    data = "${path.module}/src/context.yaml"
  }
}

output "rendered" {
  value = data.jinja_template.render.result
}
```

You can run a full example from [the dedicated sub-folder](./examples/).

## Provider documentation

* Provider configuration: [`jinja`](./docs/index.md)
* Data sources:
  - [`jinja_template`](./docs/data-sources/template.md)

| ℹ️ The [documentation folder](./docs) is generated using [`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs) and running `make docs`. |
| --- |
