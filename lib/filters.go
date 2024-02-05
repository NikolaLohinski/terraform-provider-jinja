package lib

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/dustin/go-humanize"
	json "github.com/json-iterator/go"
	tfvars_parser "github.com/musukvl/tfvars-parser"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
	"github.com/yargevad/filepathx"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

var Filters = exec.FilterSet{
	"abspath":    filterAbsPath,
	"add":        filterAdd,
	"append":     filterAppend,
	"basename":   filterBasename,
	"bool":       filterBool,
	"concat":     filterConcat,
	"dir":        filterDirname,
	"dirname":    filterDirname,
	"distinct":   filterDistinct,
	"env":        filterEnv,
	"fail":       filterFail,
	"file":       filterFile,
	"fileset":    filterFileSet,
	"flatten":    filterFlatten,
	"fromjson":   filterFromJSON,
	"fromyaml":   filterFromYAML,
	"fromtoml":   filterFromTOML,
	"frombase64": filterFromBase64,
	"fromcsv":    filterFromCSV,
	"fromtfvars": filterFromTFVars,
	"get":        filterGet,
	"ifelse":     filterIfElse,
	"insert":     filterInsert,
	"keys":       filterKeys,
	"match":      filterMatch,
	// "merge": filterMerge, 	// TODO: implement something like merge/mergeOverwrite https://masterminds.github.io/sprig/dicts.html
	"sha1":     filterSha1,
	"sha256":   filterSha256,
	"sha512":   filterSha512,
	"md5":      filterMd5,
	"split":    filterSplit,
	"totoml":   filterToToml,
	"toyaml":   filterToYAML,
	"tobase64": filterToBase64,
	"try":      filterTry,
	"unset":    filterUnset,
	"values":   filterValues,
}

func filterBool(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	switch {
	case in.IsBool():
		return exec.AsValue(in.Bool())
	case in.IsString():
		yeses := []string{"true", "yes", "on", "1"}
		nos := []string{"false", "no", "off", "0", ""}
		loweredString := strings.ToLower(in.String())
		if slices.Contains(yeses, loweredString) {
			return exec.AsValue(true)
		} else if slices.Contains(nos, loweredString) {
			return exec.AsValue(false)
		} else {
			return exec.AsValue(fmt.Errorf("%s can not be cast to boolean as it's not in ['%s'] nor ['%s']", in.String(), strings.Join(yeses, "','"), strings.Join(nos, "','")))
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
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("failed to cast: %s", in.String())))
	}
}

func filterIfElse(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	var (
		ifValue   interface{}
		elseValue interface{}
	)

	if err := params.Take(
		exec.PositionalArgument("if", nil, exec.AnyArgument(&ifValue)),
		exec.PositionalArgument("else", nil, exec.AnyArgument(&elseValue)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if in.IsTrue() {
		return exec.ToValue(ifValue)
	} else {
		return exec.ToValue(elseValue)
	}
}

func filterGet(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	var (
		key      string
		strict   bool
		fallback interface{}
	)
	if err := params.Take(
		exec.PositionalArgument("key", nil, exec.StringArgument(&key)),
		exec.KeywordArgument("strict", exec.AsValue(false), exec.BoolArgument(&strict)),
		exec.KeywordArgument("default", exec.AsValue(nil), exec.AnyArgument(&fallback)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsDict() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a dict", in.String())))
	}
	value, ok := in.GetItem(key)
	if !ok {
		if fallback != nil {
			return exec.AsValue(fallback)
		}
		if strict {
			return exec.AsValue(fmt.Errorf("item '%s' not found in: %s", key, in.String()))
		}
	}
	return value
}

func filterValues(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}

	if !in.IsDict() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a dict", in.String())))
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
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsDict() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a dict", in.String())))
	}
	out := make([]interface{}, 0)
	in.Iterate(func(idx, count int, key, value *exec.Value) bool {
		out = append(out, key.Interface())
		return true
	}, func() {})
	return exec.AsValue(out)
}

func filterTry(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
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
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() || in.String() == "" {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a non-empty string", in.String())))
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
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a list", in.String())))
	}
	out := make([]interface{}, 0)
	in.Iterate(func(idx, count int, item, _ *exec.Value) bool {
		out = append(out, item.Interface())
		return true
	}, func() {})
	for index, argument := range params.Args {
		if !argument.IsList() {
			return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s argument %s is not a list", humanize.Ordinal(index+1), argument.String())))
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
	var (
		delimiter string
	)
	if err := params.Take(
		exec.PositionalArgument("delimiter", nil, exec.StringArgument(&delimiter)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}

	output := make([]interface{}, 0)
	for _, item := range strings.Split(in.String(), delimiter) {
		output = append(output, item)
	}

	return exec.AsValue(output)
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

	return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is neither a dict nor a list", in.String())))
}

func filterFail(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return exec.AsValue(fmt.Errorf("%s: %s", in.String(), in.Error()))
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	return exec.AsValue(errors.New(in.String()))
}

func filterInsert(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	var (
		key   string
		value interface{}
	)
	if err := params.Take(
		exec.PositionalArgument("key", nil, exec.StringArgument(&key)),
		exec.PositionalArgument("value", nil, exec.AnyArgument(&value)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsDict() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a dict", in.String())))
	}

	out := make(map[string]interface{})
	in.Iterate(func(idx, count int, key, value *exec.Value) bool {
		out[key.String()] = value.Interface()
		return true
	}, func() {})
	out[key] = value

	return exec.AsValue(out)
}

func filterUnset(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	var (
		key string
	)
	if err := params.Take(
		exec.PositionalArgument("key", nil, exec.StringArgument(&key)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsDict() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a dict", in.String())))
	}

	out := make(map[string]interface{})
	in.Iterate(func(idx, count int, existingKey, value *exec.Value) bool {
		if existingKey.String() == key {
			return true
		}
		out[existingKey.String()] = value.Interface()
		return true
	}, func() {})

	return exec.AsValue(out)
}

func filterAppend(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	var (
		item interface{}
	)
	if err := params.Take(
		exec.PositionalArgument("item", nil, exec.AnyArgument(&item)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsList() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a list", in.String())))
	}

	out := make([]interface{}, 0)
	in.Iterate(func(idx, count int, item, _ *exec.Value) bool {
		out = append(out, item.Interface())
		return true
	}, func() {})
	out = append(out, item)

	return exec.AsValue(out)
}

func filterFlatten(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsList() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a list", in.String())))
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

func filterFileSet(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
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
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
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
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}

	return exec.AsValue(filepath.Base(in.String()))
}

func filterDirname(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}

	return exec.AsValue(filepath.Dir(in.String()))
}

func filterFromYAML(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() || in.String() == "" {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a non-empty string", in.String())))
	}
	var object interface{}
	if err := yaml.Unmarshal([]byte(in.String()), &object); err != nil {
		return exec.AsValue(fmt.Errorf("failed to unmarshal %s: %s", in.String(), err))
	}
	return exec.AsValue(object)
}

func filterToYAML(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	var (
		indent int
	)
	if err := params.Take(
		exec.KeywordArgument("indent", exec.AsValue(2), exec.IntArgument(&indent)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if in.IsNil() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is undefined", in.String())))
	}

	output := bytes.NewBuffer(nil)
	encoder := yaml.NewEncoder(output)
	encoder.SetIndent(indent)

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
	if in.IsError() {
		return in
	}

	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if in.IsNil() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is undefined", in.String())))
	}

	casted := in.ToGoSimpleType(false)
	if err, ok := casted.(error); ok {
		return exec.AsValue(err)
	}

	out, err := toml.Marshal(casted)
	if err != nil {
		return exec.AsValue(errors.Wrap(err, "unable to marshal to toml"))
	}

	return exec.AsSafeValue(string(out))
}

func filterFromTOML(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() || in.String() == "" {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a non-empty string", in.String())))
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
	var (
		regex string
	)
	if err := params.Take(
		exec.KeywordArgument("regex", exec.AsValue(2), exec.StringArgument(&regex)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}
	matcher, err := regexp.Compile(regex)
	if err != nil {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("failed to compile: %s: %s", regex, err)))
	}

	return exec.AsValue(matcher.MatchString(in.String()))
}

func filterAbsPath(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}
	resolved, err := e.Loader.Resolve(in.String())
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to resolve path: %s", in.String()))
	}

	path, err := filepath.Abs(resolved)
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to derive an absolute path of: %s", resolved))
	}

	return exec.AsValue(path)
}

func filterDistinct(_ *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("wrong signature for filter 'distinct': %s", err)))
	}
	if !in.IsList() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a list", in.String())))
	}
	out := make([]interface{}, 0)
	in.Iterate(func(idx, count int, item, _ *exec.Value) bool {
		for _, alreadyThere := range out {
			if reflect.DeepEqual(exec.AsValue(alreadyThere).ToGoSimpleType(false), item.ToGoSimpleType(false)) {
				return true
			}
		}
		out = append(out, item.Interface())
		return true
	}, func() {})

	return exec.AsValue(out)
}

func filterEnv(_ *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	var (
		defaultValue string
	)
	if err := params.Take(
		exec.KeywordArgument("default", exec.AsValue(""), exec.StringArgument(&defaultValue)),
	); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}
	value, ok := os.LookupEnv(in.String())
	if !ok {
		if defaultValue == "" {
			return exec.AsValue(fmt.Errorf("failed to get '%s' environment variable without default", in.String()))
		}
		value = defaultValue
	}

	return exec.AsValue(value)
}

func filterFromBase64(_ *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}
	decoded, err := base64.StdEncoding.DecodeString(in.String())
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to decode '%s' from base64: %s", in.String(), err.Error()))
	}
	return exec.AsValue(string(decoded))
}

func filterToBase64(_ *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(in.String()))
	return exec.AsValue(encoded)
}

func filterFromCSV(_ *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}

	r := strings.NewReader(in.String())
	cr := csv.NewReader(r)

	// Read the header row first, since that'll tell us which indices
	// map to which attribute names.
	headers, err := cr.Read()
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to read CSV header row: %s", err))
	}

	var rows []interface{}
	for {
		cols, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return exec.AsValue(fmt.Errorf("failed to read CSV row: %s", err))
		}

		row := make(map[string]string, len(cols))
		for index, value := range cols {
			name := headers[index]
			row[name] = value
		}
		rows = append(rows, row)
	}

	return exec.AsValue(rows)
}

func filterFromTFVars(_ *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}
	varsJson, err := tfvars_parser.Bytes([]byte(in.String()), "", tfvars_parser.Options{Simplify: true})
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to parse '%s' as tfvars: %s", in.String(), err.Error()))
	}

	vars := make(map[string]interface{})
	if err := yaml.Unmarshal(varsJson, &vars); err != nil {
		return exec.AsValue(err)
	}

	return exec.AsValue(vars)
}

func filterSha1(_ *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}
	return exec.AsValue(fmt.Sprintf("%x", sha1.Sum([]byte(in.String()))))
}

func filterSha256(_ *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}
	return exec.AsValue(fmt.Sprintf("%x", sha256.Sum256([]byte(in.String()))))
}
func filterSha512(_ *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}
	return exec.AsValue(fmt.Sprintf("%x", sha512.Sum512([]byte(in.String()))))
}

func filterMd5(_ *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if err := params.Take(); err != nil {
		return exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}
	return exec.AsValue(fmt.Sprintf("%x", md5.Sum([]byte(in.String()))))
}
