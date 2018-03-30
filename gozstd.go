package gozstd

// #cgo CFLAGS: -O3 -I${SRCDIR}/zstd/lib
// #cgo LDFLAGS: ${SRCDIR}/zstd/lib/libzstd.a
//
// #include "zstd.h"
// #include "common/zstd_errors.h"
//
// #include <stdint.h>  // for uintptr_t
//
// // The following *_wrapper functions allow avoiding memory allocations
// // durting calls from Go.
// // See https://github.com/golang/go/issues/24450 .
//
// static size_t ZSTD_compressCCtx_wrapper(ZSTD_CCtx* ctx, uintptr_t dst, size_t dstCapacity, uintptr_t src, size_t srcSize, int compressionLevel) {
//     return ZSTD_compressCCtx(ctx, (void*)dst, dstCapacity, (const void*)src, srcSize, compressionLevel);
// }
//
// static size_t ZSTD_decompressDCtx_wrapper(ZSTD_DCtx* ctx, uintptr_t dst, size_t dstCapacity, uintptr_t src, size_t srcSize) {
//     return ZSTD_decompressDCtx(ctx, (void*)dst, dstCapacity, (const void*)src, srcSize);
// }
//
// static unsigned long long ZSTD_getFrameContentSize_wrapper(uintptr_t src, size_t srcSize) {
//     return ZSTD_getFrameContentSize((const void*)src, srcSize);
// }
import "C"

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// DefaultCompressionLevel is the default compression level.
const DefaultCompressionLevel = 3 // Obtained from ZSTD_CLEVEL_DEFAULT.

// Compress appends compressed src to dst and returns the result.
func Compress(dst, src []byte) []byte {
	return CompressLevel(dst, src, DefaultCompressionLevel)
}

// CompressLevel appends compressed src to dst and returns the result.
//
// The given compression level is used for the compression.
func CompressLevel(dst, src []byte, compressionLevel int) []byte {
	compressInitOnce.Do(compressInit)

	cw := getCompressWork()
	cw.dst = dst
	cw.src = src
	cw.compressionLevel = compressionLevel
	compressWorkCh <- cw
	<-cw.done
	dst = cw.dst
	putCompressWork(cw)
	return dst
}

func getCompressWork() *compressWork {
	v := compressWorkPool.Get()
	if v == nil {
		v = &compressWork{
			done: make(chan struct{}, 1),
		}
	}
	return v.(*compressWork)
}

func putCompressWork(cw *compressWork) {
	cw.src = nil
	cw.dst = nil
	cw.compressionLevel = 0
	compressWorkPool.Put(cw)
}

type compressWork struct {
	dst              []byte
	src              []byte
	compressionLevel int
	done             chan struct{}
}

var (
	compressWorkCh   chan *compressWork
	compressWorkPool sync.Pool
	compressInitOnce sync.Once
)

func compressInit() {
	gomaxprocs := runtime.GOMAXPROCS(-1)

	compressWorkCh = make(chan *compressWork, gomaxprocs)
	for i := 0; i < gomaxprocs; i++ {
		go compressWorker()
	}
}

func compressWorker() {
	runtime.LockOSThread()
	cctx := C.ZSTD_createCCtx()
	for cw := range compressWorkCh {
		cw.dst = compress(cctx, cw.dst, cw.src, cw.compressionLevel)
		cw.done <- struct{}{}
	}
	C.ZSTD_freeCCtx(cctx)
}

func compress(cctx *C.ZSTD_CCtx, dst, src []byte, compressionLevel int) []byte {
	if len(src) == 0 {
		return dst
	}

	dstLen := len(dst)
	if cap(dst) > dstLen {
		// Fast path - try compressing without dst resize.
		dst = dst[:cap(dst)]
		dstPtr := C.uintptr_t(uintptr(unsafe.Pointer(&dst[dstLen])))
		srcPtr := C.uintptr_t(uintptr(unsafe.Pointer(&src[0])))
		result := C.ZSTD_compressCCtx_wrapper(cctx, dstPtr, C.size_t(cap(dst)-dstLen), srcPtr, C.size_t(len(src)), C.int(compressionLevel))

		compressedSize := int(result)
		if compressedSize >= 0 {
			// All OK.
			return dst[:dstLen+compressedSize]
		}

		if C.ZSTD_getErrorCode(result) != C.ZSTD_error_dstSize_tooSmall {
			// Unexpected error.
			panic(fmt.Errorf("BUG: unexpected error during compression: %s", errStr(result)))
		}
	}

	// Slow path - resize dst to fit compressed data.
	compressBound := int(C.ZSTD_compressBound(C.size_t(len(src)))) + 1
	for cap(dst)-dstLen < compressBound {
		dst = append(dst, 0)
	}
	dst = dst[:cap(dst)]

	dstPtr := C.uintptr_t(uintptr(unsafe.Pointer(&dst[dstLen])))
	srcPtr := C.uintptr_t(uintptr(unsafe.Pointer(&src[0])))
	result := C.ZSTD_compressCCtx_wrapper(cctx, dstPtr, C.size_t(cap(dst)-dstLen), srcPtr, C.size_t(len(src)), C.int(compressionLevel))

	compressedSize := int(result)
	if compressedSize >= 0 {
		// All OK.
		return dst[:dstLen+compressedSize]
	}

	// Unexpected error.
	panic(fmt.Errorf("BUG: unexpected error during compression: %s", errStr(result)))
}

// Decompress appends decompressed src to dst and returns the result.
func Decompress(dst, src []byte) ([]byte, error) {
	decompressInitOnce.Do(decompressInit)

	dw := getDecompressWork()
	dw.dst = dst
	dw.src = src
	decompressWorkCh <- dw
	<-dw.done
	dst = dw.dst
	err := dw.err
	putDecompressWork(dw)
	return dst, err
}

func getDecompressWork() *decompressWork {
	v := decompressWorkPool.Get()
	if v == nil {
		v = &decompressWork{
			done: make(chan struct{}, 1),
		}
	}
	return v.(*decompressWork)
}

func putDecompressWork(dw *decompressWork) {
	dw.dst = nil
	dw.src = nil
	dw.err = nil
	decompressWorkPool.Put(dw)
}

type decompressWork struct {
	dst  []byte
	src  []byte
	err  error
	done chan struct{}
}

var (
	decompressWorkCh   chan *decompressWork
	decompressWorkPool sync.Pool
	decompressInitOnce sync.Once
)

func decompressInit() {
	gomaxprocs := runtime.GOMAXPROCS(-1)

	decompressWorkCh = make(chan *decompressWork, gomaxprocs)
	for i := 0; i < gomaxprocs; i++ {
		go decompressWorker()
	}
}

func decompressWorker() {
	runtime.LockOSThread()
	dctx := C.ZSTD_createDCtx()
	for dw := range decompressWorkCh {
		dw.dst, dw.err = decompress(dctx, dw.dst, dw.src)
		dw.done <- struct{}{}
	}
	C.ZSTD_freeDCtx(dctx)
}

func decompress(dctx *C.ZSTD_DCtx, dst, src []byte) ([]byte, error) {
	if len(src) == 0 {
		return dst, nil
	}

	dstLen := len(dst)
	if cap(dst) > dstLen {
		// Fast path - try decompressing without dst resize.
		dst = dst[:cap(dst)]
		dstPtr := C.uintptr_t(uintptr(unsafe.Pointer(&dst[dstLen])))
		srcPtr := C.uintptr_t(uintptr(unsafe.Pointer(&src[0])))
		result := C.ZSTD_decompressDCtx_wrapper(dctx, dstPtr, C.size_t(cap(dst)-dstLen), srcPtr, C.size_t(len(src)))

		decompressedSize := int(result)
		if decompressedSize >= 0 {
			// All OK.
			return dst[:dstLen+decompressedSize], nil
		}

		if C.ZSTD_getErrorCode(result) != C.ZSTD_error_dstSize_tooSmall {
			// Error during decompression.
			return dst[:dstLen], fmt.Errorf("decompression error: %s", errStr(result))
		}
	}

	// Slow path - resize dst to fit decompressed data.
	decompressBound := int(C.ZSTD_getFrameContentSize_wrapper(
		C.uintptr_t(uintptr(unsafe.Pointer(&src[0]))), C.size_t(len(src))))
	switch uint(decompressBound) {
	case uint(C.ZSTD_CONTENTSIZE_UNKNOWN):
		return dst, fmt.Errorf("cannot decompress src, since the decompressed size is unknown")
	case uint(C.ZSTD_CONTENTSIZE_ERROR):
		return dst, fmt.Errorf("cannod decompress invalid src")
	}
	decompressBound++

	for cap(dst)-dstLen < decompressBound {
		dst = append(dst, 0)
	}
	dst = dst[:cap(dst)]

	dstPtr := C.uintptr_t(uintptr(unsafe.Pointer(&dst[dstLen])))
	srcPtr := C.uintptr_t(uintptr(unsafe.Pointer(&src[0])))
	result := C.ZSTD_decompressDCtx_wrapper(dctx, dstPtr, C.size_t(cap(dst)-dstLen), srcPtr, C.size_t(len(src)))

	decompressedSize := int(result)
	if decompressedSize >= 0 {
		// All OK.
		return dst[:dstLen+decompressedSize], nil
	}

	// Error during decompression.
	return dst[:dstLen], fmt.Errorf("decompression error: %s", errStr(result))
}

func errStr(result C.size_t) string {
	errCode := C.ZSTD_getErrorCode(result)
	errCStr := C.ZSTD_getErrorString(errCode)
	return C.GoString(errCStr)
}
