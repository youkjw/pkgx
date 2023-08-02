package singledb

import "os"

type Db struct {
	path string
	file *os.File
	data *[maxDbSize]byte

	MmapFlags int
	// truncate() and fsync() when growing the data file.
	AllocSize int
}
