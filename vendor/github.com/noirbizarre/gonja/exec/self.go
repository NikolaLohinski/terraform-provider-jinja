package exec

import (
	"strings"

	"github.com/noirbizarre/gonja/nodes"
)

func getBlocks(tpl *nodes.Template) map[string]*nodes.Wrapper {
	if tpl == nil {
		return map[string]*nodes.Wrapper{}
	}
	blocks := getBlocks(tpl.Parent)
	for name, wrapper := range tpl.Blocks {
		blocks[name] = wrapper
	}
	return blocks
}

func Self(r *Renderer) map[string]func() string {
	blocks := map[string]func() string{}
	for name, block := range getBlocks(r.Root) {
		blocks[name] = func() string {
			sub := r.Inherit()
			var out strings.Builder
			sub.Out = &out
			sub.ExecuteWrapper(block)
			return out.String()
		}
	}
	return blocks
}
