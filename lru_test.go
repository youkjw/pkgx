package memory

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAdd(t *testing.T) {
	lru, err := Lru("test")
	if err != nil {
		fmt.Println(err)
	}

	lru.Add("a", "test")
	val, ok := lru.Get("a")
	assert.True(t, ok)
	assert.Equal(t, val, "test")
	fmt.Println(val)
}
