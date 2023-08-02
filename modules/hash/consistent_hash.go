package hash

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
)

type Func func(data []byte) uint64

type ConsistentHash struct {
	hashFunc Func
	hashKeys []uint64
	ring     map[uint64][]any
	nodes    map[string]struct{}
	lock     sync.RWMutex
}

func NewConsistentHash(fn Func) *ConsistentHash {
	if fn == nil {
		fn = DigestSum64
	}

	return &ConsistentHash{
		hashFunc: fn,
		hashKeys: make([]uint64, 0),
		ring:     make(map[uint64][]any),
		nodes:    make(map[string]struct{}),
	}
}

func GetString(node any) string {
	if node == nil {
		return ""
	}

	switch node.(type) {
	case fmt.Stringer:
		return node.(fmt.Stringer).String()
	}

	val := reflect.ValueOf(node)
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	switch vt := val.Interface().(type) {
	case bool:
		return strconv.FormatBool(vt)
	case float32:
		return strconv.FormatFloat(float64(vt), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(vt, 'f', -1, 64)
	case int:
		return strconv.Itoa(vt)
	case int8:
		return strconv.Itoa(int(vt))
	case int16:
		return strconv.Itoa(int(vt))
	case int32:
		return strconv.Itoa(int(vt))
	case int64:
		return strconv.FormatInt(int64(vt), 10)
	case string:
		return vt
	case uint:
		return strconv.FormatUint(uint64(vt), 10)
	case uint8:
		return strconv.FormatUint(uint64(vt), 10)
	case uint16:
		return strconv.FormatUint(uint64(vt), 10)
	case uint32:
		return strconv.FormatUint(uint64(vt), 10)
	case uint64:
		return strconv.FormatUint(uint64(vt), 10)
	case []byte:
		return string(vt)
	case fmt.Stringer:
		return vt.String()
	case error:
		return vt.Error()
	default:
		return fmt.Sprint(val.Interface())
	}
}
