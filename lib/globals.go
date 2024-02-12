package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/yargevad/filepathx"
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
	"env":     envGlobal,
	"file":    fileGlobal,
	"fileset": fileSetGlobal,
	// "dirname": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global dirname function similar to the dirname filter
	// "basename": func(e *exec.Evaluator, arguments *exec.VarArgs) *exec.Value { return nil }, // TODO: define a global basename function similar to the basename filter
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

func envGlobal(e *exec.Evaluator, params *exec.VarArgs) *exec.Value {
	var (
		name         string
		defaultValue string
	)
	if err := params.Take(
		exec.KeywordArgument("name", nil, exec.StringArgument(&name)),
		exec.KeywordArgument("default", exec.AsValue(""), exec.StringArgument(&defaultValue)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}

	value, ok := os.LookupEnv(name)
	if !ok {
		if defaultValue == "" {
			return exec.AsValue(fmt.Errorf("failed to get '%s' environment variable without default", name))
		}
		value = defaultValue
	}

	return exec.AsValue(value)
}

func fileGlobal(e *exec.Evaluator, params *exec.VarArgs) *exec.Value {
	var (
		path string
	)
	if err := params.Take(
		exec.KeywordArgument("path", nil, exec.StringArgument(&path)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}

	if !filepath.IsAbs(path) {
		base, err := e.Loader.Resolve(".")
		if err != nil {
			return exec.AsValue(fmt.Errorf("failed to get current path with loader: %s", err))
		}
		path, err = filepath.Abs(filepath.Join(base, path))
		if err != nil {
			return exec.AsValue(fmt.Errorf("failed to resolve path %s with loader: %s", path, err))
		}
	}

	out, err := os.ReadFile(path)
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to read file at path %s: %s", path, err))
	}

	return exec.AsValue(string(out))
}

func fileSetGlobal(e *exec.Evaluator, params *exec.VarArgs) *exec.Value {
	var (
		path string
	)
	if err := params.Take(
		exec.PositionalArgument("path", nil, exec.StringArgument(&path)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	base, err := e.Loader.Resolve(".")
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to resolve path %s with loader: %s", path, err))
	}
	out, err := filepathx.Glob(filepath.Join(base, path))
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to traverse %s: %s", path, err))
	}
	return exec.AsValue(out)
}
