package gozstd

/*
#include "zstd.h"

#include <stdlib.h>  // for malloc/free
*/
import "C"

import (
	"fmt"
	"io"
	"reflect"
	"runtime"
	"unsafe"
)

var (
	cstreamInBufSize  = C.ZSTD_CStreamInSize()
	cstreamOutBufSize = C.ZSTD_CStreamOutSize()
)

// Writer implements zstd writer.
type Writer struct {
	w                io.Writer
	compressionLevel int
	cs               *C.ZSTD_CStream

	inBuf  *C.ZSTD_inBuffer
	outBuf *C.ZSTD_outBuffer

	inBufGo  []byte
	outBufGo []byte
}

// NewWriter returns new zstd writer writing compressed data to w.
//
// The returned writer must be closed with Close call in order
// to finalize the compressed stream.
//
// Call Release when the Writer is no longer needed.
func NewWriter(w io.Writer) *Writer {
	return NewWriterLevel(w, DefaultCompressionLevel)
}

// NewWriterLevel returns new zstd writer writing compressed data to w
// at the given compression level.
//
// The returned writer must be closed with Close call in order
// to finalize the compressed stream.
//
// Call Release when the Writer is no longer needed.
func NewWriterLevel(w io.Writer, compressionLevel int) *Writer {
	cs := C.ZSTD_createCStream()
	initCStream(cs, compressionLevel)

	inBuf := (*C.ZSTD_inBuffer)(C.malloc(C.sizeof_ZSTD_inBuffer))
	inBuf.src = C.malloc(cstreamInBufSize)
	inBuf.size = 0
	inBuf.pos = 0

	outBuf := (*C.ZSTD_outBuffer)(C.malloc(C.sizeof_ZSTD_outBuffer))
	outBuf.dst = C.malloc(cstreamOutBufSize)
	outBuf.size = cstreamOutBufSize
	outBuf.pos = 0

	zw := &Writer{
		w:                w,
		compressionLevel: compressionLevel,
		cs:               cs,
		inBuf:            inBuf,
		outBuf:           outBuf,
	}

	zw.inBufGo = *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(inBuf.src),
		Len:  int(cstreamInBufSize),
		Cap:  int(cstreamInBufSize),
	}))
	zw.outBufGo = *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(outBuf.dst),
		Len:  int(cstreamOutBufSize),
		Cap:  int(cstreamOutBufSize),
	}))

	runtime.SetFinalizer(zw, freeCStream)
	return zw
}

// Reset resets zw to write to w.
//
// Reset doesn't change compression level for zw.
func (zw *Writer) Reset(w io.Writer) {
	zw.inBuf.size = 0
	zw.inBuf.pos = 0
	zw.outBuf.size = cstreamOutBufSize
	zw.outBuf.pos = 0

	initCStream(zw.cs, zw.compressionLevel)

	zw.w = w
}

func initCStream(cs *C.ZSTD_CStream, compressionLevel int) {
	result := C.ZSTD_initCStream(cs, C.int(compressionLevel))
	ensureNoError("ZSTD_initCStream", result)
}

func freeCStream(v interface{}) {
	v.(*Writer).Release()
}

// Release releases all the resources occupied by zw.
//
// zw cannot be used after the release.
func (zw *Writer) Release() {
	if zw.cs == nil {
		return
	}

	result := C.ZSTD_freeCStream(zw.cs)
	ensureNoError("ZSTD_freeCStream", result)
	zw.cs = nil

	C.free(zw.inBuf.src)
	C.free(unsafe.Pointer(zw.inBuf))
	zw.inBuf = nil

	C.free(zw.outBuf.dst)
	C.free(unsafe.Pointer(zw.outBuf))
	zw.outBuf = nil

	zw.w = nil
}

// Write writes p to zw.
//
// Write doesn't flush the compressed data to the underlying writer
// due to performance reasons.
// Call Flush or Close when the compressed data must propagate
// to the underlying writer.
func (zw *Writer) Write(p []byte) (int, error) {
	pLen := len(p)
	if pLen == 0 {
		return 0, nil
	}

	for {
		n := copy(zw.inBufGo[zw.inBuf.size:], p)
		zw.inBuf.size += C.size_t(n)
		p = p[n:]
		if len(p) == 0 {
			// Fast path - just copy the data to input buffer.
			return pLen, nil
		}
		if err := zw.flushInBuf(); err != nil {
			return 0, err
		}
	}
}

func (zw *Writer) flushInBuf() error {
	result := C.ZSTD_compressStream(zw.cs, zw.outBuf, zw.inBuf)
	ensureNoError("ZSTD_compressStream", result)

	// Adjust inBuf.
	copy(zw.inBufGo, zw.inBufGo[zw.inBuf.pos:zw.inBuf.size])
	zw.inBuf.size -= zw.inBuf.pos
	zw.inBuf.pos = 0

	// Flush outBuf.
	return zw.flushOutBuf()
}

func (zw *Writer) flushOutBuf() error {
	if zw.outBuf.pos == 0 {
		return nil
	}
	outBuf := zw.outBufGo[:zw.outBuf.pos]
	n, err := zw.w.Write(outBuf)
	zw.outBuf.pos = 0
	if err != nil {
		return fmt.Errorf("cannot flush internal buffer to the underlying writer: %s", err)
	}
	if n != len(outBuf) {
		panic(fmt.Errorf("BUG: the underlying writer violated io.Writer contract and didn't return error after writing incomplete data; written %d bytes; want %d bytes",
			n, len(outBuf)))
	}
	return nil
}

// Flush flushes the remaining data from zw to the underlying writer.
func (zw *Writer) Flush() error {
	// Flush inBuf.
	for zw.inBuf.size > 0 {
		if err := zw.flushInBuf(); err != nil {
			return err
		}
	}

	// Flush the internal buffer to outBuf.
	for {
		result := C.ZSTD_flushStream(zw.cs, zw.outBuf)
		ensureNoError("ZSTD_flushStream", result)
		if err := zw.flushOutBuf(); err != nil {
			return err
		}
		if result == 0 {
			// No more data left in the internal buffer.
			return nil
		}
	}
}

// Close finalizes the compressed stream and flushes all the compressed data
// to the underlying writer.
//
// It doesn't close the underlying writer passed to New* functions.
func (zw *Writer) Close() error {
	if err := zw.Flush(); err != nil {
		return err
	}

	for {
		result := C.ZSTD_endStream(zw.cs, zw.outBuf)
		ensureNoError("ZSTD_endStream", result)
		if err := zw.flushOutBuf(); err != nil {
			return err
		}
		if result == 0 {
			return nil
		}
	}
}
