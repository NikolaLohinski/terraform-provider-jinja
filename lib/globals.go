package lib

import (
	"github.com/nikolalohinski/gonja/v2/exec"
)

// The following constants are replaced at build time
const (
	version   = "0.0.0+trunk"
	commit    = "0000000000000000000000000000000000000000"
	published = "1970-01-01T00:00:00+00:00"
	source    = "https://github.com/NikolaLohinski/terraform-provider-jinja"
	registry  = "https://registry.terraform.io/providers/NikolaLohinski/jinja/0.0.0+trunk"
)

var Globals = exec.NewContext(map[string]interface{}{
	"provider": map[string]interface{}{
		"version":   version,
		"commit":    commit,
		"published": published,
		"source":    source,
		"registry":  registry,
	},
	// "file": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global file function similar to the file filter
	// "fileset": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global fileset function similar to the fileset filter
	// "dirname": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global dirname function similar to the dirname filter
	// "basename": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global basename function similar to the basename filter
	// "abspath": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global abspath function similar to the abspath filter
	// "env": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global env function similar to the env filter // TODO: implement https://terragrunt.gruntwork.io/docs/reference/built-in-functions/#get_env
})
