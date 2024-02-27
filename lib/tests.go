package lib

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/nikolalohinski/gonja/v2/exec"
)

var Tests = exec.NewTestSet(map[string]exec.TestFunction{
	"empty": testEmpty,
	"match": testMatch,
})

func testEmpty(ctx *exec.Context, in *exec.Value, params *exec.VarArgs) (bool, error) {
	if in.IsError() {
		return false, errors.New(in.Error())
	}
	if !in.IsList() && !in.IsDict() && !in.IsString() {
		return false, exec.ErrInvalidCall(fmt.Errorf("%s is neither a list, a dict nor a string", in.String()))
	} else {
		return in.Len() == 0, nil
	}
}

func testMatch(ctx *exec.Context, in *exec.Value, params *exec.VarArgs) (bool, error) {
	if in.IsError() {
		return false, errors.New(in.Error())
	}
	var (
		regex string
	)
	if err := params.Take(
		exec.KeywordArgument("regex", exec.AsValue(2), exec.StringArgument(&regex)),
	); err != nil {
		return false, exec.AsValue(exec.ErrInvalidCall(err))
	}
	if !in.IsString() {
		return false, exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("%s is not a string", in.String())))
	}
	matcher, err := regexp.Compile(regex)
	if err != nil {
		return false, exec.AsValue(exec.ErrInvalidCall(fmt.Errorf("failed to compile: %s: %s", regex, err)))
	}

	return matcher.MatchString(in.String()), nil
}
