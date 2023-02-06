package lib

import (
	"github.com/noirbizarre/gonja/exec"
	"github.com/pkg/errors"
)

var Filters = exec.FilterSet{
	"ifelse": filterIfElse,
	"get":    filterGet,
	"values": filterValues,
	"keys":   filterKeys,
	"try":    filterTry,
}

func filterIfElse(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	p := params.Expect(2, []*exec.KwArg{{Name: "noneValue", Default: nil}})
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'ifelse'"))
	}

	trueValue := p.Args[0].String()
	falseValue := p.Args[1].String()
	noneValue := p.KwArgs["noneValue"]

	if in.IsNil() && !noneValue.IsNil() {
		return exec.AsValue(noneValue)
	} else if in.IsTrue() {
		return exec.AsValue(trueValue)
	} else {
		return exec.AsValue(falseValue)
	}
}

func filterGet(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	p := params.ExpectArgs(1)
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'get'"))
	}
	item := p.First().String()
	value, _ := in.Getitem(item)
	return value
}

func filterValues(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'values'"))
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
