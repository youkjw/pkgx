package recovery

import (
	"errors"
	"fmt"
	"runtime/debug"
)

func Hook[V any](call func() (V, error)) (result V, err error) {
	defer func() {
		if p := recover(); p != nil {
			stackInfo := string(debug.Stack())
			err = errors.New(fmt.Sprintf("panic error: %v, %s", p, stackInfo))
		}
	}()
	result, err = call()
	return
}
