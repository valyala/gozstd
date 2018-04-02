package gozstd

import (
	"fmt"
	"io/ioutil"
	"testing"
)

const benchBlocksPerStream = 10

func BenchmarkWriterDict(b *testing.B) {
	for _, blockSize := range benchBlockSizes {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, level := range benchCompressionLevels {
				b.Run(fmt.Sprintf("level_%d", level), func(b *testing.B) {
					benchmarkWriterDict(b, blockSize, level)
				})
			}
		})
	}
}

func benchmarkWriterDict(b *testing.B, blockSize, level int) {
	bd := getBenchDicts(level)
	block := newBenchString(blockSize * benchBlocksPerStream)
	b.ReportAllocs()
	b.SetBytes(int64(len(block)))
	b.RunParallel(func(pb *testing.PB) {
		zw := NewWriterDict(ioutil.Discard, bd.cd)
		defer zw.Release()
		for pb.Next() {
			for i := 0; i < benchBlocksPerStream; i++ {
				_, err := zw.Write(block[i*blockSize : (i+1)*blockSize])
				if err != nil {
					panic(fmt.Errorf("unexpected error: %s", err))
				}
			}
			if err := zw.Close(); err != nil {
				panic(fmt.Errorf("unexpected error: %s", err))
			}
			zw.Reset(ioutil.Discard, bd.cd, level)
		}
	})
}

func BenchmarkWriter(b *testing.B) {
	for _, blockSize := range benchBlockSizes {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, level := range benchCompressionLevels {
				b.Run(fmt.Sprintf("level_%d", level), func(b *testing.B) {
					benchmarkWriter(b, blockSize, level)
				})
			}
		})
	}
}

func benchmarkWriter(b *testing.B, blockSize, level int) {
	block := newBenchString(blockSize * benchBlocksPerStream)
	b.ReportAllocs()
	b.SetBytes(int64(len(block)))
	b.RunParallel(func(pb *testing.PB) {
		zw := NewWriterLevel(ioutil.Discard, level)
		defer zw.Release()
		for pb.Next() {
			for i := 0; i < benchBlocksPerStream; i++ {
				_, err := zw.Write(block[i*blockSize : (i+1)*blockSize])
				if err != nil {
					panic(fmt.Errorf("unexpected error: %s", err))
				}
			}
			if err := zw.Close(); err != nil {
				panic(fmt.Errorf("unexpected error: %s", err))
			}
			zw.Reset(ioutil.Discard, nil, level)
		}
	})
}
