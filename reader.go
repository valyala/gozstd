package gozstd

/*
#cgo CFLAGS: -O3

#define ZSTD_STATIC_LINKING_ONLY
#include "zstd.h"
#include "zstd_errors.h"

typedef struct {
	size_t dstSize;
	size_t srcSize;
	size_t dstPos;
	size_t srcPos;
} ZSTD_EXT_BufferSizes;

// The following *_wrapper functions allow avoiding memory allocations
// durting calls from Go.
// See https://github.com/golang/go/issues/24450 .

static size_t ZSTD_initDStream_usingDDict_wrapper(void *ds, void *dict) {
    ZSTD_DStream *zds = (ZSTD_DStream *)ds;
    size_t rv = ZSTD_DCtx_reset(zds, ZSTD_reset_session_only);
    if (rv != 0) {
        return rv;
    }
    return ZSTD_DCtx_refDDict(zds, (ZSTD_DDict *)dict);
}

static size_t ZSTD_freeDStream_wrapper(void *ds) {
    return ZSTD_freeDStream((ZSTD_DStream*)ds);
}

static size_t ZSTD_decompressStream_wrapper(void *ds, void* dst, const void* src, ZSTD_EXT_BufferSizes* sizes) {
    return ZSTD_decompressStream_simpleArgs((ZSTD_DStream*)ds, dst, sizes->dstSize, &sizes->dstPos, src, sizes->srcSize, &sizes->srcPos);
}
*/
import "C"

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"unsafe"
)

const minDirectWriteBufferSize = 16 * 1024

var (
	dstreamInBufSize  = C.ZSTD_DStreamInSize()
	dstreamOutBufSize = C.ZSTD_DStreamOutSize()
)

// Reader implements zstd reader.
type Reader struct {
	r  io.Reader
	ds *C.ZSTD_DStream
	dd *DDict

	inBufWrapper  *bytes.Buffer
	outBufWrapper *bytes.Buffer

	skipNextRead bool

	readerPos int
	inBuf     []byte
	outBuf    []byte
	// go doesn't allow passing pointers to structs with pointers to Go memory
	// so we can't use ZSTD_inBuffer and ZSTD_outBuffer directly
	sizes C.ZSTD_EXT_BufferSizes
}

// NewReader returns new zstd reader reading compressed data from r.
//
// Call Release when the Reader is no longer needed.
func NewReader(r io.Reader) *Reader {
	return NewReaderDict(r, nil)
}

// NewReaderDict returns new zstd reader reading compressed data from r
// using the given DDict.
//
// Call Release when the Reader is no longer needed.
func NewReaderDict(r io.Reader, dd *DDict) *Reader {
	ds := C.ZSTD_createDStream()
	initDStream(ds, dd)

	inBufWrapper := decInBufPool.Get().(*bytes.Buffer)
	outBufWrapper := decOutBufPool.Get().(*bytes.Buffer)

	zr := &Reader{
		r:             r,
		ds:            ds,
		dd:            dd,
		inBufWrapper:  inBufWrapper,
		outBufWrapper: outBufWrapper,
		inBuf:         inBufWrapper.Bytes(),
		outBuf:        outBufWrapper.Bytes(),
	}

	runtime.SetFinalizer(zr, freeDStream)
	return zr
}

// Reset resets zr to read from r using the given dictionary dd.
func (zr *Reader) Reset(r io.Reader, dd *DDict) {
	zr.readerPos = 0
	zr.sizes = C.ZSTD_EXT_BufferSizes{}
	zr.inBuf = zr.inBuf[:0]
	zr.outBuf = zr.outBuf[:0]

	zr.dd = dd
	initDStream(zr.ds, zr.dd)

	zr.r = r
}

func initDStream(ds *C.ZSTD_DStream, dd *DDict) {
	var ddict *C.ZSTD_DDict
	if dd != nil {
		ddict = dd.p
	}
	result := C.ZSTD_initDStream_usingDDict_wrapper(unsafe.Pointer(ds), unsafe.Pointer(ddict))
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

	result := C.ZSTD_freeDStream_wrapper(unsafe.Pointer(zr.ds))
	ensureNoError("ZSTD_freeDStream", result)
	zr.ds = nil

	zr.r = nil
	zr.dd = nil

	if zr.inBuf != nil {
		zr.inBuf = nil
		decInBufPool.Put(zr.inBufWrapper)
		zr.inBufWrapper = nil
	}
	if zr.outBuf != nil {
		zr.outBuf = nil
		decOutBufPool.Put(zr.outBufWrapper)
		zr.outBufWrapper = nil
	}
}

// WriteTo writes all the data from zr to w.
//
// It returns the number of bytes written to w.
func (zr *Reader) WriteTo(w io.Writer) (int64, error) {
	nn := int64(0)
	for {
		if zr.readerPos >= len(zr.outBuf) {
			if _, err := zr.fillOutBuf(nil); err != nil {
				if err == io.EOF {
					return nn, nil
				}
				return nn, err
			}
			zr.readerPos = 0
		}
		n, err := w.Write(zr.outBuf[zr.readerPos:])
		zr.readerPos += n
		nn += int64(n)
		if err != nil {
			return nn, err
		}
	}
}

// Read reads up to len(p) bytes from zr to p.
func (zr *Reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	if zr.readerPos >= len(zr.outBuf) {
		if len(p) >= minDirectWriteBufferSize {
			// write directly into the target buffer
			// but make sure to override its capacity
			return zr.fillOutBuf(p[:len(p):len(p)])
		}
		if _, err := zr.fillOutBuf(nil); err != nil {
			return 0, err
		}
		zr.readerPos = 0
	}

	n := copy(p, zr.outBuf[zr.readerPos:])
	zr.readerPos += n
	return n, nil
}

func (zr *Reader) fillOutBuf(target []byte) (int, error) {
	dst := target
	if dst == nil {
		dst = zr.outBuf
	}

	if int(zr.sizes.srcPos) == len(zr.inBuf) && !zr.skipNextRead {
		// inBuf is empty and the previously decompressed data size
		// is smaller than the maximum possible dst.size.
		// This means that the internal buffer in zr.ds doesn't contain
		// more data to decompress, so read new data into inBuf.
		if err := zr.fillInBuf(); err != nil {
			return 0, err
		}
	}

	zr.sizes.dstSize = C.size_t(cap(dst))
	zr.sizes.dstPos = 0

	inHdr := (*reflect.SliceHeader)(unsafe.Pointer(&zr.inBuf))
	outHdr := (*reflect.SliceHeader)(unsafe.Pointer(&dst))
tryDecompressAgain:
	zr.sizes.srcSize = C.size_t(len(zr.inBuf))
	prevInBufPos := zr.sizes.srcPos

	// Try decompressing inBuf into outBuf.
	result := C.ZSTD_decompressStream_wrapper(
		unsafe.Pointer(zr.ds), unsafe.Pointer(outHdr.Data), unsafe.Pointer(inHdr.Data), &zr.sizes)

	zr.skipNextRead = int(zr.sizes.dstPos) == cap(dst)
	if target == nil {
		zr.outBuf = zr.outBuf[:zr.sizes.dstPos]
	}

	if zstdIsError(result) {
		return int(zr.sizes.dstPos), fmt.Errorf("cannot decompress data: %s", errStr(result))
	}

	if zr.sizes.dstPos > 0 {
		// Something has been decompressed to outBuf. Return it.
		return int(zr.sizes.dstPos), nil
	}

	// Nothing has been decompressed from inBuf.
	if zr.sizes.srcPos != prevInBufPos && int(zr.sizes.srcPos) < len(zr.inBuf) {
		// Data has been consumed from inBuf, but decompressed
		// into nothing. There is more data in inBuf, so try
		// decompressing it again.
		goto tryDecompressAgain
	}

	// Either nothing has been consumed from inBuf or it has been
	// decompressed into nothing and inBuf became empty.
	// Read more data into inBuf and try decompressing again.
	if err := zr.fillInBuf(); err != nil {
		return 0, err
	}

	goto tryDecompressAgain
}

func (zr *Reader) fillInBuf() error {
	// Copy the remaining data to the start of inBuf.
	if zr.sizes.srcPos > 0 && int(zr.sizes.srcPos) > cap(zr.inBuf)/2 {
		copy(zr.inBuf[:cap(zr.inBuf)], zr.inBuf[zr.sizes.srcPos:])
		zr.inBuf = zr.inBuf[:len(zr.inBuf)-int(zr.sizes.srcPos)]
		zr.sizes.srcPos = 0
	}

readAgain:
	// Read more data into inBuf.
	n, err := zr.r.Read(zr.inBuf[len(zr.inBuf):cap(zr.inBuf)])
	zr.inBuf = zr.inBuf[:len(zr.inBuf)+n]

	if err == nil {
		if n == 0 {
			// Nothing has been read. Try reading data again.
			goto readAgain
		}
		return nil
	}
	if n > 0 {
		// Do not return error if at least a single byte read, i.e. forward progress is made.
		return nil
	}
	if err == io.EOF {
		// Do not wrap io.EOF, so the caller may notify the end of stream.
		return err
	}
	return fmt.Errorf("cannot read data from the underlying reader: %w", err)
}
