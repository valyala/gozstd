package gozstd

import (
	"fmt"
	"io"
)

// Reader implements zstd reader.
type Reader struct {
}

// NewReader returns new zstd reader reading compressed data from r.
//
// Call Release when the Reader is no longer needed.
func NewReader(r io.Reader) *Reader {
	return &Reader{}
}

// NewReaderDict returns new zstd reader reading compressed data from r
// using the given DDict.
//
// Call Release when the Reader is no longer needed.
func NewReaderDict(r io.Reader, dd *DDict) *Reader {
	return &Reader{}
}

// Reset resets zr to read from r using the given dictionary dd.
func (zr *Reader) Reset(r io.Reader, dd *DDict) {
}

// Release releases all the resources occupied by zr.
//
// zr cannot be used after the release.
func (zr *Reader) Release() {
}

// WriteTo writes all the data from zr to w.
//
// It returns the number of bytes written to w.
func (zr *Reader) WriteTo(w io.Writer) (int64, error) {
	return -1, fmt.Errorf("zstd not supported without cgo")
}

// Read reads up to len(p) bytes from zr to p.
func (zr *Reader) Read(p []byte) (int, error) {
	return -1, fmt.Errorf("zstd not supported without cgo")
}
