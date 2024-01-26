package lib

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dustin/go-humanize"
	json "github.com/json-iterator/go"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
	"github.com/yargevad/filepathx"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	"github.com/nikolalohinski/gonja/v2/exec"
)

var Filters = exec.FilterSet{
	// "abspath":      filterAbsPath, // TODO: implement https://developer.hashicorp.com/terraform/language/functions/abspath
	"add":      filterAdd,
	"append":   filterAppend,
	"basename": filterBasename,
	"bool":     filterBool,
	"concat":   filterConcat,
	"dir":      filterDirname,
	"dirname":  filterDirname,
	// "distinct": filterDistinct,	// TODO: implement https://developer.hashicorp.com/terraform/language/functions/distinct
	// "env": filterEnv, 			// TODO: implement https://terragrunt.gruntwork.io/docs/reference/built-in-functions/#get_env
	"fail":     filterFail,
	"file":     filterFile,
	"fileset":  filterFileset,
	"flatten":  filterFlatten,
	"fromjson": filterFromJSON,
	"fromyaml": filterFromYAML,
	"fromtoml": filterFromTOML,
	// "frombase64": filterFromBase64, 	// TODO: implement https://developer.hashicorp.com/terraform/language/functions/base64decode
	// "fromcsv": filterFromCSV, 		// TODO: implement https://developer.hashicorp.com/terraform/language/functions/csvdecode
	// "fromtfvars": filterFromTFVars, 	// TODO: implement https://terragrunt.gruntwork.io/docs/reference/built-in-functions/#read_tfvars_file
	"get":    filterGet,
	"ifelse": filterIfElse,
	"insert": filterInsert,
	"keys":   filterKeys,
	"match":  filterMatch,
	// "merge": filterMerge, 	// TODO: implement something like merge/mergeOverwrite https://masterminds.github.io/sprig/dicts.html
	// "sha1": filterSha1, 		// TODO: implement https://developer.hashicorp.com/terraform/language/functions/sha1
	// "sha256": filterSha256, 	// TODO: implement https://developer.hashicorp.com/terraform/language/functions/sha256
	// "sha512": filterSha512, 	// TODO: implement https://developer.hashicorp.com/terraform/language/functions/sha512
	// "uuid": filterUUID, 		// TODO: implement https://developer.hashicorp.com/terraform/language/functions/uuid
	"split":  filterSplit,
	"totoml": filterToToml,
	"toyaml": filterToYAML,
	// "tobase64": filterToBase64, // TODO: implement https://developer.hashicorp.com/terraform/language/functions/base64encode
	"try":    filterTry,
	"unset":  filterUnset,
	"values": filterValues,
}

func filterBool(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'bool'"))
	}
	switch {
	case in.IsBool():
		return exec.AsValue(in.Bool())
	case in.IsString():
		trues := []string{"true", "yes", "on", "1"}
		falses := []string{"false", "no", "off", "0", ""}
		loweredString := strings.ToLower(in.String())
		if slices.Contains(trues, loweredString) {
			return exec.AsValue(true)
		} else if slices.Contains(falses, loweredString) {
			return exec.AsValue(false)
		} else {
			return exec.AsValue(fmt.Errorf("\"%s\" can not be cast to boolean as it's not in [\"%s\"] nor [\"%s\"]", in.String(), strings.Join(trues, "\",\""), strings.Join(falses, "\",\"")))
		}
	case in.IsInteger():
		if in.Integer() == 1 {
			return exec.AsValue(true)
		} else if in.Integer() == 0 {
			return exec.AsValue(false)
		} else {
			return exec.AsValue(fmt.Errorf("%d can not be cast to boolean as it's not in [0,1]", in.Integer()))
		}
	case in.IsFloat():
		if in.Float() == 1.0 {
			return exec.AsValue(true)
		} else if in.Float() == 0.0 {
			return exec.AsValue(false)
		} else {
			return exec.AsValue(fmt.Errorf("%f can not be cast to boolean as it's not in [0.0,1.0]", in.Float()))
		}
	case in.IsNil():
		return exec.AsValue(false)
	default:
		return exec.AsValue(fmt.Errorf("filter 'bool' failed to cast: %s", in.String()))
	}
}

func filterIfElse(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	p := params.ExpectArgs(2)
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'ifelse'"))
	}
	if in.IsTrue() {
		return p.Args[0]
	} else {
		return p.Args[1]
	}
}

func filterGet(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	p := params.Expect(1, []*exec.KwArg{
		{Name: "strict", Default: false},
		{Name: "default", Default: nil},
	})
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'get'"))
	}
	if !in.IsDict() {
		return exec.AsValue(fmt.Errorf("filter 'get' was passed '%s' which is not a dict", in.String()))
	}
	item := p.First().String()
	value, ok := in.GetItem(item)
	if !ok {
		if fallback := p.GetKeywordArgument("default", nil); !fallback.IsNil() {
			return fallback
		}
		if p.GetKeywordArgument("strict", false).Bool() {
			return exec.AsValue(fmt.Errorf("item '%s' not found in: %s", item, in.String()))
		}
	}
	return value
}

func filterValues(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'values'"))
	}

	if !in.IsDict() {
		return exec.AsValue(fmt.Errorf("filter 'values' was passed '%s' which is not a dict", in.String()))
	}

	out := make([]interface{}, 0)
	in.Iterate(func(idx, count int, key, value *exec.Value) bool {
		out = append(out, value.Interface())
		return true
	}, func() {})

	return exec.AsValue(out)
}

func filterKeys(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'keys'"))
	}
	if !in.IsDict() {
		return exec.AsValue(fmt.Errorf("filter 'keys' was passed '%s' which is not a dict", in.String()))
	}
	out := make([]interface{}, 0)
	in.Iterate(func(idx, count int, key, value *exec.Value) bool {
		out = append(out, key.Interface())
		return true
	}, func() {})
	return exec.AsValue(out)
}

func filterTry(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'try'"))
	}
	if in == nil || in.IsError() || !in.IsTrue() {
		return exec.AsValue(nil)
	}
	return in
}

func filterFromJSON(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'fromjson'"))
	}

	if !in.IsString() || in.String() == "" {
		return exec.AsValue(fmt.Errorf("filter 'fromjson' was passed '%s' which is not a string or is empty", in.String()))
	}
	object := new(interface{})
	// first check if it's a JSON indeed
	if err := json.Unmarshal([]byte(in.String()), object); err != nil {
		return exec.AsValue(fmt.Errorf("failed to unmarshal %s: %s", in.String(), err))
	}
	// then use YAML because native JSON lib does not handle integers properly
	if err := yaml.Unmarshal([]byte(in.String()), object); err != nil {
		return exec.AsValue(fmt.Errorf("failed to unmarshal %s: %s", in.String(), err))
	}
	return exec.AsValue(*object)
}

func filterConcat(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if !in.IsList() {
		return exec.AsValue(fmt.Errorf("filter 'concat' was passed '%s' which is not a list", in.String()))
	}
	out := make([]interface{}, 0)
	in.Iterate(func(idx, count int, item, _ *exec.Value) bool {
		out = append(out, item.Interface())
		return true
	}, func() {})
	for index, argument := range params.Args {
		if !argument.IsList() {
			return exec.AsValue(fmt.Errorf("%s argument passed to filter 'concat' is not a list: %s", humanize.Ordinal(index+1), argument.String()))
		}
		argument.Iterate(func(idx, count int, item, _ *exec.Value) bool {
			out = append(out, item.Interface())
			return true
		}, func() {})
	}
	return exec.AsValue(out)
}

func filterSplit(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if !in.IsString() {
		return exec.AsValue(fmt.Errorf("filter 'split' was passed '%s' which is not a string", in.String()))
	}

	p := params.ExpectArgs(1)
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'split'"))
	}
	if !p.Args[0].IsString() {
		return exec.AsValue(fmt.Errorf("1st argument passed to filter 'split' is not a string: %s", p.Args[0].String()))
	}
	delimiter := p.First().String()

	list := strings.Split(in.String(), delimiter)

	out := make([]interface{}, len(list))
	for index, item := range list {
		out[index] = item
	}

	return exec.AsValue(out)
}

func filterAdd(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}

	if in.IsList() {
		return filterAppend(e, in, params)
	}

	if in.IsDict() {
		return filterInsert(e, in, params)
	}

	return exec.AsValue(fmt.Errorf("filter 'add' was passed '%s' which is neither a dict nor a list", in.String()))
}

func filterFail(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return exec.AsValue(fmt.Errorf("%s: %s", in.String(), in.Error()))
	}
	if p := params.ExpectNothing(); p.IsError() || !in.IsString() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'fail'"))
	}

	return exec.AsValue(errors.New(in.String()))
}

func filterInsert(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if !in.IsDict() {
		return exec.AsValue(fmt.Errorf("filter 'insert' was passed '%s' which is not a dict", in.String()))
	}
	p := params.ExpectArgs(2)
	if p.IsError() || len(p.Args) != 2 {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'insert'"))
	}
	newKey := p.Args[0]
	newValue := p.Args[1]

	out := make(map[string]interface{})
	in.Iterate(func(idx, count int, key, value *exec.Value) bool {
		out[key.String()] = value.Interface()
		return true
	}, func() {})
	out[newKey.String()] = newValue.Interface()
	return exec.AsValue(out)
}

func filterUnset(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if !in.IsDict() {
		return exec.AsValue(fmt.Errorf("filter 'unset' was passed '%s' which is not a dict", in.String()))
	}
	p := params.ExpectArgs(1)
	if p.IsError() || len(p.Args) != 1 {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'unset'"))
	}
	if !p.Args[0].IsString() {
		return exec.AsValue(fmt.Errorf("1st argument passed to filter 'unset' is not a string: %s", p.Args[0].String()))
	}
	toRemove := p.Args[0]

	out := make(map[string]interface{})
	in.Iterate(func(idx, count int, key, value *exec.Value) bool {
		if key.String() == toRemove.String() {
			return true
		}
		out[key.String()] = value.Interface()
		return true
	}, func() {})

	return exec.AsValue(out)
}

func filterAppend(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if !in.IsList() {
		return exec.AsValue(fmt.Errorf("filter 'append' was passed '%s' which is not a list", in.String()))
	}

	p := params.ExpectArgs(1)
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'append'"))
	}
	newItem := p.First()

	out := make([]interface{}, 0)
	in.Iterate(func(idx, count int, item, _ *exec.Value) bool {
		out = append(out, item.Interface())
		return true
	}, func() {})
	out = append(out, newItem)

	return exec.AsValue(out)
}

func filterFlatten(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if !in.IsList() {
		return exec.AsValue(fmt.Errorf("filter 'flatten' was passed '%s' which is not a list", in.String()))
	}

	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'flatten'"))
	}

	out := make([]interface{}, 0)
	in.Iterate(func(_, _ int, item, _ *exec.Value) bool {
		if !item.IsList() {
			out = append(out, item.Interface())
		} else {
			item.Iterate(func(_, _ int, subItem, _ *exec.Value) bool {
				out = append(out, subItem.Interface())
				return true
			}, func() {})
		}
		return true
	}, func() {})

	return exec.AsValue(out)
}

func filterFileset(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if !in.IsString() {
		return exec.AsValue(fmt.Errorf("filter 'fileset' was passed '%s' which is not a string", in.String()))
	}

	p := params.ExpectNothing()
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'fileset'"))
	}

	base, err := e.Loader.Resolve(".")
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to resolve path %s with loader: %s", in.String(), err))
	}
	out, err := filepathx.Glob(path.Join(base, in.String()))
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to traverse %s: %s", in.String(), err))
	}
	return exec.AsValue(out)
}

func filterFile(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if !in.IsString() {
		return exec.AsValue(fmt.Errorf("filter 'file' was passed '%s' which is not a string", in.String()))
	}
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'file'"))
	}

	path := in.String()
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

func filterBasename(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if !in.IsString() {
		return exec.AsValue(fmt.Errorf("filter 'basename' was passed '%s' which is not a string", in.String()))
	}

	p := params.ExpectNothing()
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'basename'"))
	}

	return exec.AsValue(filepath.Base(in.String()))
}

func filterDirname(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if !in.IsString() {
		return exec.AsValue(fmt.Errorf("filter 'dir|dirname' was passed '%s' which is not a string", in.String()))
	}

	p := params.ExpectNothing()
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'dir|dirname'"))
	}

	return exec.AsValue(filepath.Dir(in.String()))
}

func filterFromYAML(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'fromyaml'"))
	}
	if !in.IsString() || in.String() == "" {
		return exec.AsValue(fmt.Errorf("filter 'fromyaml' was passed '%s' which is not a string or is empty", in.String()))
	}
	object := new(interface{})
	if err := yaml.Unmarshal([]byte(in.String()), object); err != nil {
		return exec.AsValue(fmt.Errorf("failed to unmarshal %s: %s", in.String(), err))
	}
	return exec.AsValue(*object)
}

func filterToYAML(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	const defaultIndent = 2

	p := params.Expect(0, []*exec.KwArg{{Name: "indent", Default: defaultIndent}})
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'toyaml'"))
	}

	indent, ok := p.KwArgs["indent"]
	if !ok || indent.IsNil() {
		indent = exec.AsValue(defaultIndent)
	}

	if !indent.IsInteger() {
		return exec.AsValue(errors.Errorf("expected an integer for 'indent', got %s", indent.String()))
	}
	if in.IsNil() {
		return exec.AsValue(errors.New("filter 'toyaml' was called with an undefined object"))
	}
	output := bytes.NewBuffer(nil)
	encoder := yaml.NewEncoder(output)
	encoder.SetIndent(indent.Integer())

	// Monkey patching because the pipeline input parser is broken when the input is a list
	if in.IsList() {
		inCast := make([]interface{}, in.Len())
		for index := range inCast {
			item := exec.ToValue(in.Index(index).Val)
			inCast[index] = item.Val.Interface()
		}
		in = exec.AsValue(inCast)
	}

	castedType := in.ToGoSimpleType(true)
	if err, ok := castedType.(error); ok {
		return exec.AsValue(err)
	}

	if err := encoder.Encode(castedType); err != nil {
		return exec.AsValue(fmt.Errorf("unable to marshal to yaml: %s: %s", in.String(), err))
	}

	return exec.AsValue(output.String())
}

func filterToToml(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	// Done not mess around with trying to marshall error pipelines
	if in.IsError() {
		return in
	}

	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'totoml'"))
	}

	casted := in.ToGoSimpleType(false)
	if err, ok := casted.(error); ok {
		return exec.AsValue(err)
	}

	out, err := toml.Marshal(casted)
	if err != nil {
		return exec.AsValue(errors.Wrap(err, "unable to marhsal to toml"))
	}

	return exec.AsSafeValue(string(out))
}

func filterFromTOML(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'fromtoml'"))
	}
	if !in.IsString() || in.String() == "" {
		return exec.AsValue(fmt.Errorf("filter 'fromtoml' was passed '%s' which is not a string or is empty", in.String()))
	}
	object := new(interface{})
	if err := toml.Unmarshal([]byte(in.String()), object); err != nil {
		return exec.AsValue(fmt.Errorf("failed to unmarshal from toml %s: %s", in.String(), err))
	}
	return exec.AsValue(*object)
}

func filterMatch(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	p := params.ExpectArgs(1)
	if p.IsError() || len(p.Args) != 1 {
		return exec.AsValue(errors.Wrap(p, "wrong signature for 'match'"))
	}
	if !in.IsString() {
		return exec.AsValue(fmt.Errorf("filter 'match' was passed '%s' which is not a string", in.String()))
	}
	if !p.Args[0].IsString() {
		return exec.AsValue(fmt.Errorf("1st argument passed to filter 'match' is not a string: %s", p.Args[0].String()))
	}
	expression := p.Args[0].String()
	matcher, err := regexp.Compile(expression)
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to compile: %s: %s", expression, err))
	}

	return exec.AsValue(matcher.MatchString(in.String()))
}
