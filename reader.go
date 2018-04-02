package gozstd

/*
#define ZSTD_STATIC_LINKING_ONLY
#include "zstd.h"
#include "common/zstd_errors.h"

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
	dstreamInBufSize  = C.ZSTD_DStreamInSize()
	dstreamOutBufSize = C.ZSTD_DStreamOutSize()
)

// Reader implements zstd reader.
type Reader struct {
	r  io.Reader
	ds *C.ZSTD_DStream
	dd *DDict

	inBuf  *C.ZSTD_inBuffer
	outBuf *C.ZSTD_outBuffer

	inBufGo  []byte
	outBufGo []byte
}

// NewReader returns new zstd reader reading compressed data from r.
//
// Call Release when the Reader is no longer needed.
func NewReader(r io.Reader) *Reader {
	return NewReaderWithDict(r, nil)
}

// NewReaderWithDict returns new zstd reader reading compressed data from r
// using the given DDict.
//
// Call Release when the Reader is no longer needed.
func NewReaderWithDict(r io.Reader, dd *DDict) *Reader {
	ds := C.ZSTD_createDStream()
	initDStream(ds, dd)

	inBuf := (*C.ZSTD_inBuffer)(C.malloc(C.sizeof_ZSTD_inBuffer))
	inBuf.src = C.malloc(dstreamInBufSize)
	inBuf.size = 0
	inBuf.pos = 0

	outBuf := (*C.ZSTD_outBuffer)(C.malloc(C.sizeof_ZSTD_outBuffer))
	outBuf.dst = C.malloc(dstreamOutBufSize)
	outBuf.size = 0
	outBuf.pos = 0

	zr := &Reader{
		r:      r,
		ds:     ds,
		dd:     dd,
		inBuf:  inBuf,
		outBuf: outBuf,
	}

	zr.inBufGo = *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(inBuf.src),
		Len:  int(dstreamInBufSize),
		Cap:  int(dstreamInBufSize),
	}))
	zr.outBufGo = *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(outBuf.dst),
		Len:  int(dstreamOutBufSize),
		Cap:  int(dstreamOutBufSize),
	}))

	runtime.SetFinalizer(zr, freeDStream)
	return zr
}

// Reset resets zr to read from r using the given dictionary dd.
func (zr *Reader) Reset(r io.Reader, dd *DDict) {
	zr.inBuf.size = 0
	zr.inBuf.pos = 0
	zr.outBuf.size = 0
	zr.outBuf.pos = 0

	zr.dd = dd
	initDStream(zr.ds, zr.dd)

	zr.r = r
}

func initDStream(ds *C.ZSTD_DStream, dd *DDict) {
	var ddict *C.ZSTD_DDict
	if dd != nil {
		ddict = dd.p
	}
	result := C.ZSTD_initDStream_usingDDict(ds, ddict)
	ensureNoError("ZSTD_initDStream_usingDDict", result)
}

func freeDStream(v interface{}) {
	v.(*Reader).Release()
}

// Release releases all the resources occupied by zr.
//
// zr cannot be used after the release.
func (zr *Reader) Release() {
	if zr.ds == nil {
		return
	}

	result := C.ZSTD_freeDStream(zr.ds)
	ensureNoError("ZSTD_freeDStream", result)
	zr.ds = nil

	C.free(zr.inBuf.src)
	C.free(unsafe.Pointer(zr.inBuf))
	zr.inBuf = nil

	C.free(zr.outBuf.dst)
	C.free(unsafe.Pointer(zr.outBuf))
	zr.outBuf = nil

	zr.r = nil
	zr.dd = nil
}

// Read reads up to len(p) bytes from zr to p.
func (zr *Reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	for zr.outBuf.pos == zr.outBuf.size {
		if err := zr.readInBuf(); err != nil {
			return 0, err
		}
	}

	n := copy(p, zr.outBufGo[zr.outBuf.pos:zr.outBuf.size])
	zr.outBuf.pos += C.size_t(n)
	return n, nil
}

func (zr *Reader) readInBuf() error {
	if zr.inBuf.size == 0 {
		// Read inBuf.
		n, err := zr.r.Read(zr.inBufGo)
		zr.inBuf.size = C.size_t(n)
		if err != nil {
			if err != io.EOF {
				return fmt.Errorf("cannot read data from the underlying reader: %s", err)
			}
			if n == 0 {
				return io.EOF
			}
		}
	}

	// Decompress inBuf.
	zr.outBuf.size = dstreamOutBufSize
	zr.outBuf.pos = 0
	result := C.ZSTD_decompressStream(zr.ds, zr.outBuf, zr.inBuf)
	zr.outBuf.size = zr.outBuf.pos
	zr.outBuf.pos = 0

	// Adjust inBuf.
	copy(zr.inBufGo, zr.inBufGo[zr.inBuf.pos:zr.inBuf.size])
	zr.inBuf.size -= zr.inBuf.pos
	zr.inBuf.pos = 0

	if C.ZSTD_getErrorCode(result) != 0 {
		return fmt.Errorf("cannot decompress data: %s", errStr(result))
	}

	return nil
}
