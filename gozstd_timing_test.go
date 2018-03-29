package gozstd

import (
	"fmt"
	"sync/atomic"
	"testing"
)

var Sink uint64

func BenchmarkCompress(b *testing.B) {
	for randomness := 1; randomness <= 256; randomness *= 2 {
		b.Run(fmt.Sprintf("randomness_%d", randomness), func(b *testing.B) {
			benchmarkCompress(b, randomness)
		})
	}
}

func benchmarkCompress(b *testing.B, randomness int) {
	testStr := newTestString(64*1024, randomness)
	src := []byte(testStr)
	b.ReportAllocs()
	b.SetBytes(int64(len(testStr)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		n := 0
		var dst []byte
		for pb.Next() {
			dst = Compress(dst[:0], src)
			n += len(dst)
		}
		atomic.AddUint64(&Sink, uint64(n))
	})
}

func BenchmarkDecompress(b *testing.B) {
	for randomness := 1; randomness <= 256; randomness *= 2 {
		b.Run(fmt.Sprintf("randomness_%d", randomness), func(b *testing.B) {
			benchmarkDecompress(b, randomness)
		})
	}
}

func benchmarkDecompress(b *testing.B, randomness int) {
	testStr := newTestString(64*1024, randomness)
	src := Compress(nil, []byte(testStr))
	b.ReportAllocs()
	b.SetBytes(int64(len(testStr)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		n := 0
		var dst []byte
		var err error
		for pb.Next() {
			dst, err = Decompress(dst[:0], src)
			if err != nil {
				panic(fmt.Errorf("unexpected error: %s", err))
			}
			n += len(dst)
		}
		atomic.AddUint64(&Sink, uint64(n))
	})
}
