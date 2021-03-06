package lru

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestLruMemory_Init(t *testing.T) {
	cache, err := NewLru("test",
		WithSize(3),
		WithOnRemove(func(key string, value interface{}) error {
			fmt.Println(key, value)
			return nil
		}),
	)
	assert.Nil(t, err)

	cache.Add("a", 5)
	fmt.Println(cache.Get("a"))
	//tim := time.After(5 * time.Second)
	//for {
	//	select {
	//	case <-tim:
	//		break
	//	default:
	//		lru.Remove("a")
	//	}
	//}
}

func TestLruMemory_Add(t *testing.T) {
	lru, err := NewLru("test", WithSize(2))
	assert.Nil(t, err)

	lru.Add("a", "test")
	val, ok := lru.Get("a")
	assert.True(t, ok)
	assert.Equal(t, val, "test")
	fmt.Println(val)

	ok = lru.Add("a", "test1")
	assert.True(t, ok)

	val, ok = lru.Get("a")
	assert.True(t, ok)
	assert.Equal(t, val, "test1")
	fmt.Println(val)

	lru.Add("b", "test2")
	lru.Add("c", "test3")
	lru.Add("d", "test4")
	lenght := lru.Len()
	assert.Equal(t, lenght, 2)
	fmt.Println(lenght)
}

func TestLruMemory_Get(t *testing.T) {
	lru, err := NewLru("test")
	assert.Nil(t, err)

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
	lru, err := NewLru("test")
	assert.Nil(t, err)

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

func TestLruMemory_Remove(t *testing.T) {
	lru, err := NewLru("test")
	assert.Nil(t, err)

	lru.Add("a", "test")
	val, ok := lru.Get("a")
	assert.True(t, ok)
	assert.Equal(t, val, "test")

	ok = lru.Remove("a")
	assert.True(t, ok)

	val, ok = lru.Get("a")
	fmt.Println(val, ok)
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestLruMemory_Contains(t *testing.T) {
	lru, err := NewLru("test")
	assert.Nil(t, err)

	lru.Add("a", "test")
	ok := lru.Contains("a")
	assert.True(t, ok)
}

func TestLruMemory_Keys(t *testing.T) {
	lru, err := NewLru("test")
	assert.Nil(t, err)

	lru.Add("a", "test")
	lru.Add("a", "test1")
	lru.Add("b", "test")
	lru.Add("c", "test")
	lru.Add("d", "test")
	lru.Add("e", "test")
	lru.Add("f", "test")

	keys := lru.Keys()
	fmt.Println(keys)
}

func TestLruMemory_Range(t *testing.T) {
	lru, err := NewLru("test")
	assert.Nil(t, err)

	lru.Add("a", "test")
	lru.Add("a", "test1")
	lru.Add("b", "test")
	lru.Add("c", "test")
	lru.Add("d", "test")
	lru.Add("e", "test")
	lru.Add("f", "test")

	err = lru.Range(func(key string, value interface{}) error {
		fmt.Println(key, value)
		return nil
	})
	assert.Nil(t, err)
}
