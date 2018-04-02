package gozstd

import (
	"fmt"
	"sync/atomic"
	"testing"
)

var Sink uint64

func BenchmarkCompress(b *testing.B) {
	for _, blockSize := range []int{1, 10, 100, 1000, 64 * 1024} {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, randomness := range []int{1, 2, 10, 256} {
				b.Run(fmt.Sprintf("randomness_%d", randomness), func(b *testing.B) {
					benchmarkCompress(b, blockSize, randomness)
				})
			}
		})
	}
}

func benchmarkCompress(b *testing.B, blockSize, randomness int) {
	testStr := newTestString(blockSize, randomness)
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
	for _, blockSize := range []int{1, 10, 100, 1000, 64 * 1024} {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, randomness := range []int{1, 2, 10, 256} {
				b.Run(fmt.Sprintf("randomness_%d", randomness), func(b *testing.B) {
					benchmarkDecompress(b, blockSize, randomness)
				})
			}
		})
	}
}

func benchmarkDecompress(b *testing.B, blockSize, randomness int) {
	testStr := newTestString(blockSize, randomness)
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
