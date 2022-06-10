package recovery

import (
	"fmt"
	"testing"
	"time"
)

func TestHook(t *testing.T) {
	//go func() {
	res, err := Hook[string](func() (string, error) {
		panic("panic test")
		return "test", nil
	})
	fmt.Println(res)
	fmt.Println(err)
	//}()
	time.Sleep(time.Second)
}
