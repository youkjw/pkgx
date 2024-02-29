package singledb

import "os"

type DB struct {
	path string
	file *os.File
	data *[maxDbSize]byte

	MmapFlags int
	// truncate() and fsync() when growing the data file.
	AllocSize int
}

func NewDB() *DB {
	return &DB{}
}
