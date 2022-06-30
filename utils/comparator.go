package utils

import "time"

type Comparator[V any] func(a, b V) int

// StringComparator provides a fast comparison on strings
func StringComparator[V string](a, b V) int {
	min := len(b)
	if len(a) < len(b) {
		min = len(a)
	}
	diff := 0
	for i := 0; i < min && diff == 0; i++ {
		diff = int(a[i]) - int(b[i])
	}
	if diff == 0 {
		diff = len(a) - len(b)
	}
	if diff < 0 {
		return -1
	}
	if diff > 0 {
		return 1
	}
	return 0
}

type Number interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64
}

// IntComparator provides a basic comparison on int
func IntComparator[V Number](a, b V) int {
	switch {
	case a > b:
		return 1
	case b < a:
		return -1
	default:
		return 0
	}
}

// TimeComparator provides a basic comparison on time.Time
func TimeComparator(a, b time.Time) int {
	switch {
	case a.After(b):
		return 1
	case a.Before(b):
		return -1
	default:
		return 0
	}
}
