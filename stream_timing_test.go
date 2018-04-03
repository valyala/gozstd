package gozstd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
)

func BenchmarkStreamCompress(b *testing.B) {
	for _, blockSize := range benchBlockSizes {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, level := range benchCompressionLevels {
				b.Run(fmt.Sprintf("level_%d", level), func(b *testing.B) {
					benchmarkStreamCompress(b, blockSize, level)
				})
			}
		})
	}
}

func benchmarkStreamCompress(b *testing.B, blockSize, level int) {
	block := newBenchString(blockSize * benchBlocksPerStream)
	b.ReportAllocs()
	b.SetBytes(int64(len(block)))
	b.RunParallel(func(pb *testing.PB) {
		r := bytes.NewReader(block)
		for pb.Next() {
			if err := StreamCompressLevel(ioutil.Discard, r, level); err != nil {
				panic(fmt.Errorf("unexpected error: %s", err))
			}
			r.Reset(block)
		}
	})
}

func BenchmarkStreamDecompress(b *testing.B) {
	for _, blockSize := range benchBlockSizes {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, level := range benchCompressionLevels {
				b.Run(fmt.Sprintf("level_%d", level), func(b *testing.B) {
					benchmarkStreamDecompress(b, blockSize, level)
				})
			}
		})
	}
}

func benchmarkStreamDecompress(b *testing.B, blockSize, level int) {
	block := newBenchString(blockSize * benchBlocksPerStream)
	cd := CompressLevel(nil, block, level)
	b.Logf("compressionRatio: %f", float64(len(block))/float64(len(cd)))
	b.ReportAllocs()
	b.SetBytes(int64(len(block)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		r := bytes.NewReader(cd)
		for pb.Next() {
			if err := StreamDecompress(ioutil.Discard, r); err != nil {
				panic(fmt.Errorf("unexpected error: %s", err))
			}
			r.Reset(cd)
		}
	})
}
