package memory

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestLruMemory_Add(t *testing.T) {
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

func TestLruMemory_Get(t *testing.T) {
	lru, err := Lru("test")
	if err != nil {
		fmt.Println(err)
	}

	lru.Add("a", "test")

	var wait sync.WaitGroup
	for i := 0; i < 20; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			val, ok := lru.Get("a")

			fmt.Println(val, ok)
			assert.True(t, ok)
			assert.Equal(t, val, "test")
		}()
	}
	wait.Wait()

	fmt.Println(lru.Stat())
}

func TestLruMemory_Take(t *testing.T) {
	lru, err := Lru("test")
	if err != nil {
		fmt.Println(err)
	}

	var wait sync.WaitGroup
	for i := 0; i < 20; i++ {
		var val interface{}
		wait.Add(1)

		go func() {
			defer wait.Done()
			val, err = lru.Take("a", func() (interface{}, error) {
				return "test", nil
			})

			fmt.Println(val, err)
			assert.Nil(t, err)
			assert.Equal(t, val, "test")
		}()
	}
	wait.Wait()

	fmt.Println(lru.Stat())

	//val, ok := lru.Get("a")
	//assert.True(t, ok)
	//assert.Equal(t, val, "test")
	//fmt.Println(val)
}
