package gozstd

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func BenchmarkReader(b *testing.B) {
	for _, blockSize := range []int{1, 10, 100, 1000, 64 * 1024} {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, randomness := range []int{1, 2, 10, 256} {
				b.Run(fmt.Sprintf("randomness_%d", randomness), func(b *testing.B) {
					benchmarkReader(b, blockSize, randomness)
				})
			}
		})
	}
}

func benchmarkReader(b *testing.B, blockSize, randomness int) {
	block := []byte(newTestString(blockSize*100, randomness))
	cd := Compress(nil, block)
	b.ReportAllocs()
	b.SetBytes(int64(len(block)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		r := bytes.NewReader(cd)
		zr := NewReader(r)
		defer zr.Release()
		buf := make([]byte, blockSize)
		for pb.Next() {
			for {
				_, err := io.ReadFull(zr, buf)
				if err != nil {
					if err == io.EOF {
						break
					}
					panic(fmt.Errorf("unexpected error: %s", err))
				}
			}
			r.Reset(cd)
			zr.Reset(r, nil)
		}
	})
}
