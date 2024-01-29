package lib

import (
	"github.com/nikolalohinski/gonja/v2/exec"
)

// The following constants are replaced at build time
var (
	Version    = "0.0.0+trunk"
	Commit     = "0000000000000000000000000000000000000000"
	Date       = "1970-01-01T00:00:00+00:00"
	Repository = "github.com/nikolalohinski/terraform-provider-jinja"
	Registry   = "registry.terraform.io/NikolaLohinski/jinja"
)

var Globals = exec.NewContext(map[string]interface{}{
	"provider": map[string]interface{}{
		"version":    Version,
		"commit":     Commit,
		"date":       Date,
		"repository": Repository,
		"registry":   Registry,
	},
	// "file": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global file function similar to the file filter
	// "fileset": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global fileset function similar to the fileset filter
	// "dirname": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global dirname function similar to the dirname filter
	// "basename": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global basename function similar to the basename filter
	// "abspath": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global abspath function similar to the abspath filter
	// "env": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global env function similar to the env filter // TODO: implement https://terragrunt.gruntwork.io/docs/reference/built-in-functions/#get_env
})
