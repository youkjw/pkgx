package httpx

import (
	"fmt"
	"testing"
)

func Test_cleanPath(t *testing.T) {
	path := "/./a/../b/c"
	//path := "/../a/v/./c"
	//fmt.Println(filepath.Clean(path))
	fmt.Println(cleanPath(path))
}
