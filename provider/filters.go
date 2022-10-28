package jinja

import (
	"math/rand"
	"time"

	"github.com/noirbizarre/gonja/exec"
	"github.com/pkg/errors"
)

var Filters = exec.FilterSet{
	"ifelse": filterIfElse,
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func filterIfElse(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	p := params.Expect(2, []*exec.KwArg{{Name: "none_val", Default: nil}})
	if p.IsError() {
		return exec.AsValue(errors.Wrap(p, "Wrong signature for 'ifelse'"))
	}

	true_val := p.Args[0].String()
	false_val := p.Args[1].String()
	none_val := p.KwArgs["none_val"]

	if in.IsNil() && !none_val.IsNil() {
		return exec.AsValue(none_val)
	} else if in.IsTrue() {
		return exec.AsValue(true_val)
	} else {
		return exec.AsValue(false_val)
	}
}
