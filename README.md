<div align="center">
<img src="./logo.png" width="200"/>
<h1><code>terraform-provider-jinja</code></h1>
</div>

A `terraform` provider that makes it possible to render [Jinja](https://jinja.palletsprojects.com/) templates within a `terraform` project.

The Jinja engine used under the hood is based on [the `gonja` Golang library](https://github.com/nikolalohinski/gonja/v2) and aims to be as close as possible to `python`'s Jinja.

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

### Guidelines

Please read through the [contribution guidelines](./CONTRIBUTING.md) before diving into any work.

### Requirements

- Get `make` with [the online instructions](https://www.gnu.org/software/make/) ;
- Install go `>= 1.21` by following the [official documentation](https://go.dev/doc/install) ;
- Grab `tfplugindocs` using the [dedicated procedure](https://github.com/hashicorp/terraform-plugin-docs?tab=readme-ov-file#installation) ;
- Download `goreleaser` from [its website](https://goreleaser.com/install/) ;
- Fetch the latest `terraform` binary from [Hashicorp's web page](https://developer.hashicorp.com/terraform/install).

### Tests

The unit tests can be run using:

```sh
make test
```

### Local provider installation

The provider can be installed locally, using:

```sh
make install
```

See the [`Makefile`](./Makefile) for more details on what it means.

### Running the example

The example located under `examples/` can be ran with:

```sh
make example
```