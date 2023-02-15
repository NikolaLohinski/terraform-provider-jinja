package lib

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/noirbizarre/gonja/builtins"
	"github.com/noirbizarre/gonja/exec"
	"github.com/pkg/errors"
	"github.com/yargevad/filepathx"
	"gopkg.in/yaml.v3"
)

var Filters = exec.FilterSet{
	"ifelse":   filterIfElse,
	"get":      filterGet,
	"insert":   filterInsert,
	"unset":    filterUnset,
	"values":   filterValues,
	"keys":     filterKeys,
	"try":      filterTry,
	"tojson":   filterToJSON,
	"fromjson": filterFromJSON,
	"concat":   filterConcat,
	"split":    filterSplit,
	"add":      filterAdd,
	"append":   filterAppend,
	"flatten":  filterFlatten,
	"fail":     filterFail,
	"fileset":  filterFileset,
}

func filterIfElse(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	p := params.ExpectArgs(2)
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'ifelse'"))
	}
	if in.IsError() {
		return in
	}
	if in.IsTrue() {
		return p.Args[0]
	} else {
		return p.Args[1]
	}
}

func filterGet(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	p := params.Expect(1, []*exec.KwArg{
		{Name: "strict", Default: false},
		{Name: "default", Default: nil},
	})
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'get'"))
	}
	if !in.IsDict() {
		return exec.AsValue(errors.New("Filter 'get' was passed a non-dict type"))
	}
	item := p.First().String()
	value, ok := in.Getitem(item)
	if !ok {
		if fallback := p.GetKwarg("default", nil); !fallback.IsNil() {
			return fallback
		}
		if p.GetKwarg("strict", false).Bool() {
			return exec.AsValue(fmt.Errorf("item '%s' not found in: %s", item, in.String()))
		}
	}
	return value
}

func filterValues(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'values'"))
	}

	if in.IsError() {
		return in
	}

	if !in.IsDict() {
		return exec.AsValue(errors.New("Filter 'values' was passed a non-dict type"))
	}

	out := []*exec.Value{}
	in.Iterate(func(idx, count int, key, value *exec.Value) bool {
		out = append(out, value)
		return true
	}, func() {})

	return exec.AsValue(out)
}

func filterKeys(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'keys'"))
	}
	if in.IsError() {
		return in
	}

	if !in.IsDict() {
		return exec.AsValue(errors.New("Filter 'keys' was passed a non-dict type"))
	}
	out := []*exec.Value{}
	in.Iterate(func(idx, count int, key, value *exec.Value) bool {
		out = append(out, key)
		return true
	}, func() {})
	return exec.AsValue(out)
}

func filterTry(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'try'"))
	}
	if in == nil || in.IsError() || !in.IsTrue() {
		return exec.AsValue(nil)
	}
	return in
}

func filterToJSON(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	// Monkey patching because the builtin json filter is broken for arrays
	if in.IsList() {
		inCast := make([]interface{}, in.Len())
		for index := range inCast {
			item := exec.ToValue(in.Index(index).Val)
			inCast[index] = item.Val.Interface()
		}
		in = exec.AsValue(inCast)
	}

	return builtins.Filters["tojson"](e, in, params)
}

func filterFromJSON(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'keys'"))
	}
	if in.IsError() {
		return in
	}
	if !in.IsString() || in.String() == "" {
		return exec.AsValue(errors.New("Filter 'fromJSON' was passed an empty or non-string type"))
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
		return exec.AsValue(errors.New("Filter 'concat' was passed a non-list type"))
	}
	out := make([]*exec.Value, 0)
	in.Iterate(func(idx, count int, item, _ *exec.Value) bool {
		out = append(out, item)
		return true
	}, func() {})
	for index, argument := range params.Args {
		if !argument.IsList() {
			return exec.AsValue(fmt.Errorf("%s argument passed to filter 'concat' is not a list: %s", humanize.Ordinal(index+1), argument))
		}
		argument.Iterate(func(idx, count int, item, _ *exec.Value) bool {
			out = append(out, item)
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
		return exec.AsValue(errors.New("Filter 'split' was passed a non-string type"))
	}
	p := params.ExpectArgs(1)
	if p.IsError() || !p.First().IsString() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'split'"))
	}
	delimiter := p.First().String()

	list := strings.Split(in.String(), delimiter)

	out := make([]*exec.Value, len(list))
	for index, item := range list {
		out[index] = exec.AsValue(item)
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

	return exec.AsValue(errors.New("Filter 'add' was passed a non-dict nor list type"))
}

func filterFail(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return exec.AsValue(fmt.Errorf("%s: %s", in.String(), in.Error()))
	}
	if p := params.ExpectNothing(); p.IsError() || !in.IsString() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'fail'"))
	}

	return exec.AsValue(errors.New(in.String()))
}

func filterInsert(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if !in.IsDict() {
		return exec.AsValue(errors.New("Filter 'set' was passed a non-dict type"))
	}
	p := params.ExpectArgs(2)
	if p.IsError() || len(p.Args) != 2 {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'set'"))
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
		return exec.AsValue(errors.New("Filter 'unset' was passed a non-dict type"))
	}
	p := params.ExpectArgs(1)
	if p.IsError() || len(p.Args) != 1 {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'unset'"))
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
		return exec.AsValue(errors.New("Filter 'append' was passed a non-list type"))
	}

	p := params.ExpectArgs(1)
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'append'"))
	}
	newItem := p.First()

	out := make([]*exec.Value, 0)
	in.Iterate(func(idx, count int, item, _ *exec.Value) bool {
		out = append(out, item)
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
		return exec.AsValue(errors.New("Filter 'append' was passed a non-list type"))
	}

	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'flatten'"))
	}

	out := make([]*exec.Value, 0)
	in.Iterate(func(_, _ int, item, _ *exec.Value) bool {
		if !item.IsList() {
			out = append(out, item)
		} else {
			item.Iterate(func(_, _ int, subItem, _ *exec.Value) bool {
				out = append(out, subItem)
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
		return exec.AsValue(errors.New("Filter 'fileset' was passed a non-string type"))
	}

	p := params.ExpectNothing()
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'fileset'"))
	}

	base, err := e.Loader.Path(".")
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to resolve path %s with loader: %s", in.String(), err))
	}
	out, err := filepathx.Glob(path.Join(base, in.String()))
	if err != nil {
		return exec.AsValue(fmt.Errorf("failed to traverse %s: %s", in.String(), err))
	}
	return exec.AsValue(out)
}
