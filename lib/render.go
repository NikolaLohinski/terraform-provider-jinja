package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/noirbizarre/gonja"
	"github.com/noirbizarre/gonja/config"
	"github.com/noirbizarre/gonja/exec"
	"github.com/noirbizarre/gonja/loaders"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

func Render(ctx *Context) ([]byte, error) {
	environment, err := getEnvironment(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build jinja environment: %s", err)
	}

	template, err := getTemplate(ctx, environment)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %s", err)
	}

	values, err := getValues(ctx.Values)
	if err != nil {
		return nil, fmt.Errorf("failed to parse values: %s", err)
	}

	if err := validate(values, ctx.Schemas); err != nil {
		return nil, fmt.Errorf("failed to validate context against schema: %s", err)
	}

	result, err := template.Execute(values)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %s", err)
	}

	return []byte(result), err
}

func getValues(values *Values) (map[string]interface{}, error) {
	parsedValues := make(map[string]interface{})
	if values != nil {
		switch strings.ToLower(values.Type) {
		case "json":
			// Validate JSON context format before unmarshalling with YAML decoder to avoid casting ints to floats
			// see https://stackoverflow.com/questions/71525600/golang-json-converts-int-to-float-what-can-i-do
			if err := json.Unmarshal(values.Data, &map[string]interface{}{}); err != nil {
				return nil, fmt.Errorf("failed to decode JSON context: %s", err)
			}
			if err := yaml.Unmarshal(values.Data, &parsedValues); err != nil {
				return nil, fmt.Errorf("failed to unmarshal JSON context: %s", err)
			}
		case "yaml":
			if err := yaml.Unmarshal(values.Data, &parsedValues); err != nil {
				return nil, fmt.Errorf("failed to unmarshal YAML context: %s", err)
			}
		default:
			return nil, fmt.Errorf("provided context has an unsupported type: %v", values.Type)
		}
	}
	return parsedValues, nil
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

func validate(values map[string]interface{}, schemas []json.RawMessage) error {
	schemaErrors := []string{}
	for index, schema := range schemas {
		validator, err := jsonschema.CompileString("", string(schema))
		if err != nil {
			return fmt.Errorf("failed to compile JSON schema %s: %s", err, schema)
		}

		if err := validator.Validate(values); err != nil {
			schemaErrors = append(schemaErrors, fmt.Errorf("failed to pass %s JSON schema validation: %s", humanize.Ordinal(index+1), err).Error())
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

	for name, filter := range Filters {
		if err := environment.Filters.Register(name, filter); err != nil {
			return nil, fmt.Errorf("failed register filter %s: %s", name, err)
		}
	}

	return environment, nil
}
