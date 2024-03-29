package hash

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"github.com/spaolacci/murmur3"
)

func Hash(data []byte) []byte {
	digest := sha1.New()
	digest.Write(data)
	return digest.Sum(nil)
}

// Md5 returns the md5 bytes of data.
func Md5(data []byte) []byte {
	digest := md5.New()
	digest.Write(data)
	return digest.Sum(nil)
}

// DigestHex returns the hex string of data.
func DigestHex(data []byte) string {
	return fmt.Sprintf("%x", Md5(data))
}

// DigestSum64 returns the unit64 of data.
func DigestSum64(data []byte) uint64 {
	return murmur3.Sum64(data)
}
