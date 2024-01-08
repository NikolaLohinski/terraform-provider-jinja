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
	strictUndefined := strictUndefinedSchema()
	strictUndefined.Description = "Provider-wide toggle to fail on missing attribute/item"
	trimBlocks := trimBlocksSchema()
	trimBlocks.Description = "Provider-wide toggle to trim leading spaces and tabs from the start of a line to a block"
	leftStripBlocks := leftStripBlocksSchema()
	leftStripBlocks.Description = "Provider-wide toggle to remove the first newline after a block"
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"delimiters":        delimiters,
			"strict_undefined":  strictUndefined,
			"trim_blocks":       trimBlocks,
			"left_strip_blocks": leftStripBlocks,
		},
		DataSourcesMap: map[string]*schema.Resource{
			"jinja_template": dataSourceJinjaTemplate(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	meta := make(map[string]interface{})
	// Must provide default at runtime because TypeList and TypSet ignore DefaultFunc
	// See https://github.com/hashicorp/terraform-plugin-sdk/issues/142
	delimiters, ok := d.GetOk("delimiters.0")
	if ok {
		meta["delimiters"] = delimiters
	} else {
		meta["delimiters"] = default_delimiters
	}
	strictUndefined, ok := d.GetOk("strict_undefined")
	if ok {
		meta["strict_undefined"] = strictUndefined
	}
	leftStripBlocks, ok := d.GetOk("left_strip_blocks")
	if ok {
		meta["left_strip_blocks"] = leftStripBlocks
	}
	trimBlocks, ok := d.GetOk("trim_blocks")
	if ok {
		meta["trim_blocks"] = trimBlocks
	}
	return meta, nil
}
