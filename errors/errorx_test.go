package errors

import (
	"testing"
)

var (
	Testing1 = NewErrorx(1, "testing error 1")
	Testing2 = NewErrorx(2, "testing error 2")
)

func TestNewErrorx(t *testing.T) {
	t.Log(Testing1.Code())
	t.Log(Testing1.Error())

	t.Log(Testing2.Code())
	t.Log(Testing2.Error())
}
