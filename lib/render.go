package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/dustin/go-humanize"
	"github.com/imdario/mergo"
	"github.com/nikolalohinski/gonja"
	"github.com/nikolalohinski/gonja/config"
	"github.com/nikolalohinski/gonja/exec"
	"github.com/nikolalohinski/gonja/loaders"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

const defaultRenderTimeout = 5 * time.Second

func Render(ctx *Context) ([]byte, map[string]interface{}, error) {
	channel := make(chan struct {
		Result string
		Values map[string]interface{}
		Err    error
	})
	go func() {
		result := struct {
			Result string
			Values map[string]interface{}
			Err    error
		}{}
		defer func() {
			if err := recover(); err != nil {
				result.Err = fmt.Errorf(heredoc.Doc(`
				a runtime error led gonja to panic: %s

				Known possible reasons for gonja panic attacks are:
				- call to the panic filter
				- call to a non existent macro
				- trying to do python-like object indexing using something like 'object["key"]'
				`), err)
			}
			channel <- result
		}()

		environment, err := getEnvironment(ctx)
		if err != nil {
			result.Err = fmt.Errorf("failed to build jinja environment: %s", err)
			return
		}

		template, err := getTemplate(ctx, environment)
		if err != nil {
			result.Err = fmt.Errorf("failed to parse template: %s", err)
			return
		}

		result.Values, err = getValues(ctx.Values)
		if err != nil {
			result.Err = fmt.Errorf("failed to parse values: %s", err)
			return
		}

		if err := validate(result.Values, ctx.Schemas); err != nil {
			result.Err = fmt.Errorf("failed to validate context against schema: %s", err)
			return
		}

		result.Result, result.Err = template.Execute(result.Values)
	}()
	select {
	case output := <-channel:
		if output.Err != nil {
			return nil, nil, fmt.Errorf("failed to execute template: %s", output.Err)
		}
		return []byte(output.Result), output.Values, nil
	case <-time.After(defaultRenderTimeout):
		return nil, nil, fmt.Errorf(heredoc.Doc(`
			rendering timed out after %s: known possible reasons for timeouts are:
			- an unclosed string
			- an unclosed variable block in an included template
		`), defaultRenderTimeout.String())
	}
}

func getValues(values []Values) (map[string]interface{}, error) {
	var mergedValues map[string]interface{}
	for index, value := range values {
		layer := make(map[string]interface{})
		switch strings.ToLower(value.Type) {
		case "json":
			// Validate JSON context format before unmarshalling with YAML decoder to avoid casting ints to floats
			// see https://stackoverflow.com/questions/71525600/golang-json-converts-int-to-float-what-can-i-do
			if err := json.Unmarshal(value.Data, &map[string]interface{}{}); err != nil {
				return nil, fmt.Errorf("failed to decode JSON context: %s", err)
			}
			if err := yaml.Unmarshal(value.Data, &layer); err != nil {
				return nil, fmt.Errorf("failed to unmarshal JSON context: %s", err)
			}
		case "yaml":
			if err := yaml.Unmarshal(value.Data, &layer); err != nil {
				return nil, fmt.Errorf("failed to unmarshal YAML context: %s", err)
			}
		default:
			return nil, fmt.Errorf("provided context has an unsupported type: %v", value.Type)
		}

		if mergedValues == nil {
			mergedValues = layer
			continue
		}
		if err := mergo.Merge(&mergedValues, layer, mergo.WithOverride, mergo.WithOverwriteWithEmptyValue); err != nil {
			return nil, fmt.Errorf("failed to merge %s values layer: %s", humanize.Ordinal(index+1), err)
		}
	}
	return mergedValues, nil
}

func getTemplate(ctx *Context, environment *gonja.Environment) (*exec.Template, error) {
	file, err := environment.Loader.Get(path.Base(ctx.Template.Location))
	if err != nil {
		return nil, fmt.Errorf("error loading file: %s", err)
	}

	buffer, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading template: %s", err)
	}

	bundle := string(buffer)
	if ctx.Template.Header != "" {
		bundle = ctx.Template.Header + "\n" + bundle
	}
	if ctx.Template.Footer != "" {
		bundle = bundle + "\n" + ctx.Template.Footer
	}

	return exec.NewTemplate(ctx.Template.Location, bundle, environment.EvalConfig)
}

func validate(values map[string]interface{}, schemas map[string]json.RawMessage) error {
	schemaErrors := []string{}
	names := make([]string, 0)
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		schema, ok := schemas[name]
		if !ok {
			return fmt.Errorf("could not find name '%s' in schemas: %v", name, schemas)
		}
		validator, err := jsonschema.CompileString("", string(schema))
		if err != nil {
			return fmt.Errorf("failed to compile '%s' JSON schema %s: %s", name, schema, err)
		}

		if err := validator.Validate(values); err != nil {
			schemaErrors = append(schemaErrors, fmt.Errorf("failed to pass '%s' JSON schema validation: %s", name, err).Error())
			continue
		}
	}

	if len(schemaErrors) > 0 {
		return fmt.Errorf("\n\t%s", strings.Join(schemaErrors, "\n\t"))
	}

	return nil
}

func getEnvironment(ctx *Context) (*gonja.Environment, error) {
	gonjaConfig := config.DefaultConfig

	gonjaConfig.BlockStartString = ctx.Configuration.Delimiters.BlockStart
	gonjaConfig.BlockEndString = ctx.Configuration.Delimiters.BlockEnd
	gonjaConfig.VariableStartString = ctx.Configuration.Delimiters.VariableStart
	gonjaConfig.VariableEndString = ctx.Configuration.Delimiters.VariableEnd
	gonjaConfig.CommentStartString = ctx.Configuration.Delimiters.CommentStart
	gonjaConfig.CommentEndString = ctx.Configuration.Delimiters.CommentEnd

	gonjaConfig.StrictUndefined = ctx.Configuration.StrictUndefined

	loader, err := loaders.NewFileSystemLoader(path.Dir(ctx.Template.Location))
	if err != nil {
		return nil, fmt.Errorf("failed get a file system loader: %v", err)
	}

	environment := gonja.NewEnvironment(gonjaConfig, loader)

	return environment, nil
}
