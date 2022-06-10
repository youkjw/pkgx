package recovery

import (
	"fmt"
	"testing"
	"time"
)

func TestCatch(t *testing.T) {
	go func() {
		res, err := Hook[string](func() (string, error) {
			return "testing", nil
		})
		fmt.Println(res)
		fmt.Println(err)
	}()
	time.Sleep(time.Second)
}
