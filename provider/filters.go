package jinja

import (
	"github.com/noirbizarre/gonja/exec"
	"github.com/pkg/errors"
)

var Filters = exec.FilterSet{
	"ifelse": filterIfElse,
	"get":    filterGet,
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
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'attr'"))
	}
	item := p.First().String()
	value, _ := in.Getitem(item)
	return value
}
