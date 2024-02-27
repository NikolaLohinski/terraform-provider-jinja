package lib

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/nikolalohinski/gonja/v2/exec"
)

var Tests = exec.NewTestSet(map[string]exec.TestFunction{
	"empty": testEmpty,
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
