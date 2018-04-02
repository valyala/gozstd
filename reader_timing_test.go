package gozstd

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func BenchmarkReaderDict(b *testing.B) {
	for _, blockSize := range benchBlockSizes {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, level := range benchCompressionLevels {
				b.Run(fmt.Sprintf("level_%d", level), func(b *testing.B) {
					benchmarkReaderDict(b, blockSize, level)
				})
			}
		})
	}
}

func benchmarkReaderDict(b *testing.B, blockSize, level int) {
	bd := getBenchDicts(level)
	block := newBenchString(blockSize * benchBlocksPerStream)
	cd := CompressDict(nil, block, bd.cd)
	b.Logf("compressionRatio: %f", float64(len(block))/float64(len(cd)))
	b.ReportAllocs()
	b.SetBytes(int64(len(block)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		r := bytes.NewReader(cd)
		zr := NewReaderDict(r, bd.dd)
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
			zr.Reset(r, bd.dd)
		}
	})
}

func BenchmarkReader(b *testing.B) {
	for _, blockSize := range benchBlockSizes {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, level := range benchCompressionLevels {
				b.Run(fmt.Sprintf("level_%d", level), func(b *testing.B) {
					benchmarkReader(b, blockSize, level)
				})
			}
		})
	}
}

func benchmarkReader(b *testing.B, blockSize, level int) {
	block := newBenchString(blockSize * benchBlocksPerStream)
	cd := CompressLevel(nil, block, level)
	b.Logf("compressionRatio: %f", float64(len(block))/float64(len(cd)))
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
