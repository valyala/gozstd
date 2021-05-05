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
static size_t ZSTD_CCtx_setParameter_wrapper(void *cs, ZSTD_cParameter param, int value) {
    return ZSTD_CCtx_setParameter((ZSTD_CStream*)cs, param, value);
}

static size_t ZSTD_initCStream_wrapper(void *cs, int compressionLevel) {
    return ZSTD_initCStream((ZSTD_CStream*)cs, compressionLevel);
}

static size_t ZSTD_CCtx_refCDict_wrapper(void *cc, void *dict) {
    return ZSTD_CCtx_refCDict((ZSTD_CCtx*)cc, (ZSTD_CDict*)dict);
}

static size_t ZSTD_freeCStream_wrapper(void *cs) {
    return ZSTD_freeCStream((ZSTD_CStream*)cs);
}

static size_t ZSTD_compressStream_wrapper(void *cs, void* dst, const void* src, ZSTD_EXT_BufferSizes* sizes, ZSTD_EndDirective endOp) {
	return ZSTD_compressStream2_simpleArgs((ZSTD_CStream*)cs, dst, sizes->dstSize, &sizes->dstPos, src, sizes->srcSize, &sizes->srcPos, endOp);
}

static size_t ZSTD_flushStream_wrapper(void *cs, void *dst, ZSTD_EXT_BufferSizes* sizes) {
	size_t res;
	ZSTD_outBuffer outBuf;

	outBuf.dst = dst;
	outBuf.size = sizes->dstSize;
	outBuf.pos = sizes->dstPos;

	res = ZSTD_flushStream((ZSTD_CStream*)cs, &outBuf);
	sizes->dstPos = outBuf.pos;
	return res;
}

static size_t ZSTD_endStream_wrapper(void *cs, void *dst, ZSTD_EXT_BufferSizes* sizes) {
	size_t res;
	ZSTD_outBuffer outBuf;

	outBuf.dst = dst;
	outBuf.size = sizes->dstSize;
	outBuf.pos = sizes->dstPos;

	res = ZSTD_endStream((ZSTD_CStream*)cs, &outBuf);
	sizes->dstPos = outBuf.pos;
	return res;
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

var (
	cstreamInBufSize  = C.ZSTD_CStreamInSize()
	cstreamOutBufSize = C.ZSTD_CStreamOutSize()
)

// Writer implements zstd writer.
type Writer struct {
	w                io.Writer
	compressionLevel int
	wlog             int
	cs               *C.ZSTD_CStream
	cd               *CDict

	inBufWrapper  *bytes.Buffer
	outBufWrapper *bytes.Buffer

	inBuf  []byte
	outBuf []byte
	sizes  C.ZSTD_EXT_BufferSizes
}

// NewWriter returns new zstd writer writing compressed data to w.
//
// The returned writer must be closed with Close call in order
// to finalize the compressed stream.
//
// Call Release when the Writer is no longer needed.
func NewWriter(w io.Writer) *Writer {
	return NewWriterParams(w, nil)
}

// NewWriterLevel returns new zstd writer writing compressed data to w
// at the given compression level.
//
// The returned writer must be closed with Close call in order
// to finalize the compressed stream.
//
// Call Release when the Writer is no longer needed.
func NewWriterLevel(w io.Writer, compressionLevel int) *Writer {
	params := &WriterParams{
		CompressionLevel: compressionLevel,
	}
	return NewWriterParams(w, params)
}

// NewWriterDict returns new zstd writer writing compressed data to w
// using the given cd.
//
// The returned writer must be closed with Close call in order
// to finalize the compressed stream.
//
// Call Release when the Writer is no longer needed.
func NewWriterDict(w io.Writer, cd *CDict) *Writer {
	params := &WriterParams{
		Dict: cd,
	}
	return NewWriterParams(w, params)
}

const (
	// WindowLogMin is the minimum value of the windowLog parameter.
	WindowLogMin = 10 // from zstd.h
	// WindowLogMax32 is the maximum value of the windowLog parameter on 32-bit architectures.
	WindowLogMax32 = 30 // from zstd.h
	// WindowLogMax64 is the maximum value of the windowLog parameter on 64-bit architectures.
	WindowLogMax64 = 31 // from zstd.h

	// DefaultWindowLog is the default value of the windowLog parameter.
	DefaultWindowLog = 0
)

// A WriterParams allows users to specify compression parameters by calling
// NewWriterParams.
//
// Calling NewWriterParams with a nil WriterParams is equivalent to calling
// NewWriter.
type WriterParams struct {
	// Compression level. Special value 0 means 'default compression level'.
	CompressionLevel int

	// WindowLog. Must be clamped between WindowLogMin and WindowLogMin32/64.
	// Special value 0 means 'use default windowLog'.
	//
	// Note: enabling log distance matching increases memory usage for both
	// compressor and decompressor. When set to a value greater than 27, the
	// decompressor requires special treatment.
	WindowLog int

	// Dict is optional dictionary used for compression.
	Dict *CDict
}

// NewWriterParams returns new zstd writer writing compressed data to w
// using the given set of parameters.
//
// The returned writer must be closed with Close call in order
// to finalize the compressed stream.
//
// Call Release when the Writer is no longer needed.
func NewWriterParams(w io.Writer, params *WriterParams) *Writer {
	if params == nil {
		params = &WriterParams{}
	}

	cs := C.ZSTD_createCStream()
	initCStream(cs, *params)

	inBufWrapper := compInBufPool.Get().(*bytes.Buffer)
	outBufWrapper := compOutBufPool.Get().(*bytes.Buffer)

	zw := &Writer{
		w:                w,
		compressionLevel: params.CompressionLevel,
		wlog:             params.WindowLog,
		cs:               cs,
		cd:               params.Dict,
		inBufWrapper:     inBufWrapper,
		outBufWrapper:    outBufWrapper,
		inBuf:            inBufWrapper.Bytes(),
		outBuf:           outBufWrapper.Bytes(),
	}

	runtime.SetFinalizer(zw, freeCStream)
	return zw
}

// Reset resets zw to write to w using the given dictionary cd and the given
// compressionLevel. Use ResetWriterParams if you wish to change other
// parameters that were set via WriterParams.
func (zw *Writer) Reset(w io.Writer, cd *CDict, compressionLevel int) {
	params := WriterParams{
		CompressionLevel: compressionLevel,
		WindowLog:        zw.wlog,
		Dict:             cd,
	}
	zw.ResetWriterParams(w, &params)
}

// ResetWriterParams resets zw to write to w using the given set of parameters.
func (zw *Writer) ResetWriterParams(w io.Writer, params *WriterParams) {
	zw.inBuf = zw.inBuf[:0]
	zw.outBuf = zw.outBuf[:0]
	zw.sizes = C.ZSTD_EXT_BufferSizes{}

	zw.cd = params.Dict
	initCStream(zw.cs, *params)

	zw.w = w
}

func initCStream(cs *C.ZSTD_CStream, params WriterParams) {
	if params.Dict != nil {
		result := C.ZSTD_CCtx_refCDict_wrapper(
			unsafe.Pointer(cs),
			unsafe.Pointer(params.Dict.p))
		ensureNoError("ZSTD_CCtx_refCDict", result)
	} else {
		result := C.ZSTD_initCStream_wrapper(
			unsafe.Pointer(cs),
			C.int(params.CompressionLevel))
		ensureNoError("ZSTD_initCStream", result)
	}

	result := C.ZSTD_CCtx_setParameter_wrapper(
		unsafe.Pointer(cs),
		C.ZSTD_cParameter(C.ZSTD_c_windowLog),
		C.int(params.WindowLog))
	ensureNoError("ZSTD_CCtx_setParameter", result)
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

	result := C.ZSTD_freeCStream_wrapper(unsafe.Pointer(zw.cs))
	ensureNoError("ZSTD_freeCStream", result)
	zw.cs = nil

	zw.w = nil
	zw.cd = nil

	if zw.inBufWrapper != nil {
		zw.inBuf = nil
		compInBufPool.Put(zw.inBufWrapper)
		zw.inBufWrapper = nil
	}

	if zw.outBufWrapper != nil {
		zw.outBuf = nil
		compOutBufPool.Put(zw.outBufWrapper)
		zw.outBufWrapper = nil
	}
}

// ReadFrom reads all the data from r and writes it to zw.
//
// Returns the number of bytes read from r.
//
// ReadFrom may not flush the compressed data to the underlying writer
// due to performance reasons.
// Call Flush or Close when the compressed data must propagate
// to the underlying writer.
func (zw *Writer) ReadFrom(r io.Reader) (int64, error) {
	nn := int64(0)
	for {
		inBuf := zw.inBuf[len(zw.inBuf):cap(zw.inBuf)]
		// Fill the inBuf.
		for len(inBuf) > 0 {
			n, err := r.Read(inBuf)

			// Sometimes n > 0 even when Read() returns an error.
			// This is true especially if the error is io.EOF.
			inBuf = inBuf[n:]
			zw.inBuf = zw.inBuf[:len(zw.inBuf)+n]
			nn += int64(n)

			if err != nil {
				if err == io.EOF {
					return nn, nil
				}
				return nn, err
			}
		}

		// Flush the inBuf.
		if err := zw.flushInBuf(); err != nil {
			return nn, err
		}
	}
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
		n := copy(zw.inBuf[len(zw.inBuf):cap(zw.inBuf)], p)
		zw.inBuf = zw.inBuf[:len(zw.inBuf)+n]
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
	zw.sizes.dstSize = C.size_t(cap(zw.outBuf))
	zw.sizes.dstPos = C.size_t(len(zw.outBuf))
	zw.sizes.srcSize = C.size_t(len(zw.inBuf))
	zw.sizes.srcPos = 0

	outHdr := (*reflect.SliceHeader)(unsafe.Pointer(&zw.outBuf))
	inHdr := (*reflect.SliceHeader)(unsafe.Pointer(&zw.inBuf))

	result := C.ZSTD_compressStream_wrapper(
		unsafe.Pointer(zw.cs), unsafe.Pointer(outHdr.Data), unsafe.Pointer(inHdr.Data),
		&zw.sizes, C.ZSTD_e_continue)
	ensureNoError("ZSTD_compressStream_wrapper", result)

	zw.outBuf = zw.outBuf[:zw.sizes.dstPos]

	// Move the remaining data to the start of inBuf.
	if int(zw.sizes.srcPos) < len(zw.inBuf) {
		copy(zw.inBuf[:cap(zw.inBuf)], zw.inBuf[zw.sizes.srcPos:len(zw.inBuf)])
		zw.inBuf = zw.inBuf[:len(zw.inBuf)-int(zw.sizes.srcPos)]
	} else {
		zw.inBuf = zw.inBuf[:0]
	}

	if cap(zw.outBuf)-int(zw.sizes.dstPos) > int(zw.sizes.dstPos) && zw.sizes.srcPos > 0 {
		// There is enough space in outBuf and the last compression
		// succeeded, so don't flush outBuf yet.
		return nil
	}

	// Flush outBuf, since there is low space in it or the last compression
	// attempt was unsuccessful.
	return zw.flushOutBuf()
}

func (zw *Writer) flushOutBuf() error {
	if len(zw.outBuf) == 0 {
		// Nothing to flush.
		return nil
	}

	bufLen := len(zw.outBuf)
	n, err := zw.w.Write(zw.outBuf)
	zw.outBuf = zw.outBuf[:0]
	if err != nil {
		return fmt.Errorf("cannot flush internal buffer to the underlying writer: %s", err)
	}
	if n != bufLen {
		panic(fmt.Errorf("BUG: the underlying writer violated io.Writer contract and didn't return error after writing incomplete data; written %d bytes; want %d bytes",
			n, bufLen))
	}
	return nil
}

// Flush flushes the remaining data from zw to the underlying writer.
func (zw *Writer) Flush() error {
	// Flush inBuf.
	for len(zw.inBuf) > 0 {
		if err := zw.flushInBuf(); err != nil {
			return err
		}
	}

	// Flush the internal buffer to outBuf.
	for {
		outHdr := (*reflect.SliceHeader)(unsafe.Pointer(&zw.outBuf))
		zw.sizes.dstSize = C.size_t(cap(zw.outBuf))
		zw.sizes.dstPos = C.size_t(len(zw.outBuf))

		result := C.ZSTD_flushStream_wrapper(
			unsafe.Pointer(zw.cs), unsafe.Pointer(outHdr.Data), &zw.sizes)
		ensureNoError("ZSTD_flushStream", result)
		zw.outBuf = zw.outBuf[:zw.sizes.dstPos]
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
		outHdr := (*reflect.SliceHeader)(unsafe.Pointer(&zw.outBuf))
		zw.sizes.dstSize = C.size_t(cap(zw.outBuf))
		zw.sizes.dstPos = C.size_t(len(zw.outBuf))

		result := C.ZSTD_endStream_wrapper(
			unsafe.Pointer(zw.cs),
			unsafe.Pointer(outHdr.Data), &zw.sizes)
		ensureNoError("ZSTD_endStream", result)
		zw.outBuf = zw.outBuf[:zw.sizes.dstPos]
		if err := zw.flushOutBuf(); err != nil {
			return err
		}
		if result == 0 {
			return nil
		}
	}
}
