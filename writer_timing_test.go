package gozstd

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func BenchmarkWriter(b *testing.B) {
	for _, blockSize := range []int{1, 10, 100, 1000, 64 * 1024} {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, randomness := range []int{1, 2, 10, 256} {
				b.Run(fmt.Sprintf("randomness_%d", randomness), func(b *testing.B) {
					benchmarkWriter(b, blockSize, randomness)
				})
			}
		})
	}
}

func benchmarkWriter(b *testing.B, blockSize, randomness int) {
	block := []byte(newTestString(blockSize*100, randomness))
	b.ReportAllocs()
	b.SetBytes(int64(len(block)))
	b.RunParallel(func(pb *testing.PB) {
		zw := NewWriter(ioutil.Discard)
		defer zw.Release()
		for pb.Next() {
			for i := 0; i < 100; i++ {
				_, err := zw.Write(block[i*blockSize : (i+1)*blockSize])
				if err != nil {
					panic(fmt.Errorf("unexpected error: %s", err))
				}
			}
			if err := zw.Close(); err != nil {
				panic(fmt.Errorf("unexpected error: %s", err))
			}
			zw.Reset(ioutil.Discard, nil, DefaultCompressionLevel)
		}
	})
}
