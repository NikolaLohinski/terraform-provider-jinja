package jinja

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider returns a *schema.Provider.
func Provider() *schema.Provider {
	delimiters := delimitersSchema()
	delimiters.Description = "Provider-wide custom delimiters for the jinja engine"
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"delimiters": delimiters,
		},
		DataSourcesMap: map[string]*schema.Resource{
			"jinja_template": dataSourceJinjaTemplate(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	// Must provide default at runtime because TypeList and TypSet ignore DefaultFunc
	// See https://github.com/hashicorp/terraform-plugin-sdk/issues/142
	delimiters, ok := d.GetOk("delimiters.0")
	if !ok {
		return default_delimiters, nil
	}
	return delimiters, nil
}
