package recovery

import (
	"fmt"
	"runtime/debug"
)

func Hook[V any](callback func() (V, error)) (result V, err error) {
	defer func() {
		if p := recover(); p != nil {
			stackInfo := string(debug.Stack())
			fmt.Printf("panic error: %v, %s", p, stackInfo)
		}
	}()
	result, err = callback()
	return
}
