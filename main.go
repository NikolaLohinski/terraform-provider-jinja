package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	jinja "github.com/nikolalohinski/terraform-provider-jinja/provider"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{ProviderFunc: jinja.Provider})
}
