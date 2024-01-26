package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/nikolalohinski/terraform-provider-jinja/v2/internal/provider"
	"github.com/nikolalohinski/terraform-provider-jinja/v2/lib"
)

const (
	registryAddress = "registry.terraform.io/NikolaLohinski/jinja"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: registryAddress,
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(lib.Version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
