package lib

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
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
	"abspath": absPathGlobal,
	"uuid":    uuidGlobal,
	// "file": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global file function similar to the file filter
	// "fileset": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global fileset function similar to the fileset filter
	// "dirname": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global dirname function similar to the dirname filter
	// "basename": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global basename function similar to the basename filter
	// "env": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global env function similar to the env filter // TODO: implement https://terragrunt.gruntwork.io/docs/reference/built-in-functions/#get_env
})

func absPathGlobal(e *exec.Evaluator, params *exec.VarArgs) *exec.Value {
	var (
		path string
	)
	if err := params.Take(
		exec.KeywordArgument("path", nil, exec.StringArgument(&path)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}

	resolved, err := e.Loader.Resolve(path)
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to resolve path: %s", path))
	}

	p, err := filepath.Abs(resolved)
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to derive an absolute path of: %s", resolved))
	}

	return exec.AsValue(p)
}

func uuidGlobal(e *exec.Evaluator, params *exec.VarArgs) *exec.Value {
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	return exec.AsValue(uuid.New().String())
}
