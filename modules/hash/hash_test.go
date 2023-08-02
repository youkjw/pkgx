package hash

import (
	"fmt"
	"testing"
)

func TestDigestSum64(t *testing.T) {
	fmt.Println(DigestHex(Hash([]byte("123456"))))
	fmt.Println(DigestSum64(Hash([]byte("123456"))))
}
