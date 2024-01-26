<div align="center">
<img src="./logo.png" width="200"/>
<h1><code>terraform-provider-jinja</code></h1>
</div>

A `terraform` provider that makes it possible to render [Jinja](https://jinja.palletsprojects.com/) templates within a `terraform` project.

The Jinja engine used under the hood is based on [the `gonja` Golang library](https://github.com/nikolalohinski/gonja/v2) and aims to be "mostly" compliant with the Jinja API.

The JSON schema validation engine is based on [the `jsonschema` Golang library](https://github.com/santhosh-tekuri/jsonschema).

## Example

```hcl
provider "jinja" {
  strict_undefined = true
}

data "jinja_template" "render" {
  source {
    template  = file("${path.module}/template.j2")
    directory = path.module
  }
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

## Development

### Requirements

- Install `make` with [the official instructions](https://www.gnu.org/software/make/) ;
- Install go `>= 1.20` by following the [official documentation](https://go.dev/doc/install).

| ⚠️ You also need to install `golangci-lint` by following the [official instructions](https://golangci-lint.run/usage/install/#local-installation) to be able to run `make lint`. |
| --- |

### Tests

The unit tests can be run using:

```shell
make test
```

### Local provider installation

The provider can be installed locally, using:

```shell
make install
```

See the [`Makefile`](./Makefile) for more details on what it means.
