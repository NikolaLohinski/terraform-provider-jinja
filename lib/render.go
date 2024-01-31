package lib

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/dustin/go-humanize"
	tfvars_parser "github.com/musukvl/tfvars-parser"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/config"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/nikolalohinski/gonja/v2/loaders"
	"github.com/pelletier/go-toml/v2"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

type valuesFormat string

const (
	FormatJSON   valuesFormat = "json"
	FormatYAML   valuesFormat = "yaml"
	FormatTOML   valuesFormat = "toml"
	FormatTFVars valuesFormat = "tfvars"
)

var (
	SupportedValuesFormats = []string{
		string(FormatJSON),
		string(FormatYAML),
		string(FormatTOML),
		string(FormatTFVars),
	}
)

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
				result.Err = fmt.Errorf("a runtime error led the jinja engine to panic: %s", err)
			}
			channel <- result
		}()

		template, err := parseTemplate(ctx)
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

		result.Result, result.Err = template.ExecuteToString(exec.NewContext(result.Values))
	}()
	select {
	case output := <-channel:
		if output.Err != nil {
			return nil, nil, fmt.Errorf("failed to execute template: %s", output.Err)
		}
		return []byte(output.Result), output.Values, nil
	case <-time.After(ctx.Timeout):
		return nil, nil, fmt.Errorf("rendering timed out after %s", ctx.Timeout.String())
	}
}

func getValues(values []Values) (map[string]interface{}, error) {
	var mergedValues map[string]interface{}
	for index, value := range values {
		layer := make(map[string]interface{})
		switch valuesFormat(strings.ToLower(value.Type)) {
		case FormatJSON:
			// Validate JSON context format before unmarshalling with YAML decoder to avoid casting ints to floats
			// see https://stackoverflow.com/questions/71525600/golang-json-converts-int-to-float-what-can-i-do
			if err := json.Unmarshal(value.Data, &map[string]interface{}{}); err != nil {
				return nil, fmt.Errorf("failed to decode JSON context: %s", err)
			}
			if err := yaml.Unmarshal(value.Data, &layer); err != nil {
				return nil, fmt.Errorf("failed to unmarshal JSON context: %s", err)
			}
		case FormatYAML:
			if err := yaml.Unmarshal(value.Data, &layer); err != nil {
				return nil, fmt.Errorf("failed to unmarshal YAML context: %s", err)
			}
		case FormatTOML:
			if err := toml.Unmarshal(value.Data, &layer); err != nil {
				return nil, fmt.Errorf("failed to unmarshal TOML context: %s", err)
			}
		case FormatTFVars:
			varsJson, err := tfvars_parser.Bytes([]byte(value.Data), "", tfvars_parser.Options{Simplify: true})
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal TFVars context: %s", err.Error())
			}
			if err := yaml.Unmarshal(varsJson, &layer); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("provided context has an unsupported type: %v", value.Type)
		}

		if mergedValues == nil {
			mergedValues = layer
			continue
		}
		if err := mergo.Merge(&mergedValues, layer, mergo.WithOverride, mergo.WithOverrideEmptySlice); err != nil {
			return nil, fmt.Errorf("failed to merge %s values layer: %s", humanize.Ordinal(index+1), err)
		}
	}
	return mergedValues, nil
}

func parseTemplate(ctx *Context) (*exec.Template, error) {
	gonjaConfig := config.New()

	gonjaConfig.BlockStartString = ctx.Configuration.Delimiters.BlockStart
	gonjaConfig.BlockEndString = ctx.Configuration.Delimiters.BlockEnd
	gonjaConfig.VariableStartString = ctx.Configuration.Delimiters.VariableStart
	gonjaConfig.VariableEndString = ctx.Configuration.Delimiters.VariableEnd
	gonjaConfig.CommentStartString = ctx.Configuration.Delimiters.CommentStart
	gonjaConfig.CommentEndString = ctx.Configuration.Delimiters.CommentEnd
	gonjaConfig.LeftStripBlocks = ctx.Configuration.LeftStripBlocks
	gonjaConfig.TrimBlocks = ctx.Configuration.TrimBlocks

	gonjaConfig.StrictUndefined = ctx.Configuration.StrictUndefined

	environment := gonja.DefaultEnvironment

	environment.Filters.Update(Filters)
	environment.Tests.Update(Tests)
	environment.Context.Update(Globals)

	fileSystemLoader, err := loaders.NewFileSystemLoader(ctx.Source.Directory)
	if err != nil {
		return nil, fmt.Errorf("failed to create a file system loader: %v", err)
	}

	rootID := fmt.Sprintf("root-%s", string(sha256.New().Sum([]byte(ctx.Source.Template))))
	shiftedLoader, err := loaders.NewShiftedLoader(rootID, bytes.NewBufferString(ctx.Source.Template), fileSystemLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to create a shifted loader: %v", err)
	}

	return exec.NewTemplate(rootID, gonjaConfig, shiftedLoader, environment)
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
		return fmt.Errorf("\n%s", strings.Join(schemaErrors, "\n"))
	}

	return nil
}
