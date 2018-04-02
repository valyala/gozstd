package gozstd

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
)

var Sink uint64

var benchBlockSizes = []int{1, 1e1, 1e2, 1e3, 1e4, 1e5, 3e5}
var benchCompressionLevels = []int{3, 5, 10}

func BenchmarkDecompressDict(b *testing.B) {
	for _, blockSize := range benchBlockSizes {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, level := range benchCompressionLevels {
				b.Run(fmt.Sprintf("level_%d", level), func(b *testing.B) {
					benchmarkDecompressDict(b, blockSize, level)
				})
			}
		})
	}
}

func benchmarkDecompressDict(b *testing.B, blockSize, level int) {
	block := newBenchString(blockSize)
	bd := getBenchDicts(level)
	src := CompressDict(nil, block, bd.cd)
	b.Logf("compressionRatio: %f", float64(len(block))/float64(len(src)))
	b.ReportAllocs()
	b.SetBytes(int64(blockSize))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		n := 0
		var dst []byte
		var err error
		for pb.Next() {
			dst, err = DecompressDict(dst[:0], src, bd.dd)
			if err != nil {
				panic(fmt.Errorf("BUG: cannot decompress with dict: %s", err))
			}
			n += len(dst)
		}
		atomic.AddUint64(&Sink, uint64(n))
	})
}

func BenchmarkCompressDict(b *testing.B) {
	for _, blockSize := range benchBlockSizes {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, level := range benchCompressionLevels {
				b.Run(fmt.Sprintf("level_%d", level), func(b *testing.B) {
					benchmarkCompressDict(b, blockSize, level)
				})
			}
		})
	}
}

func benchmarkCompressDict(b *testing.B, blockSize, level int) {
	src := newBenchString(blockSize)
	bd := getBenchDicts(level)
	b.ReportAllocs()
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		n := 0
		var dst []byte
		for pb.Next() {
			dst = CompressDict(dst[:0], src, bd.cd)
			n += len(dst)
		}
		atomic.AddUint64(&Sink, uint64(n))
	})
}

func getBenchDicts(level int) *benchDicts {
	benchDictsLock.Lock()
	tmp := benchDictsMap[level]
	if tmp == nil {
		tmp = newBenchDicts(level)
		benchDictsMap[level] = tmp
	}
	benchDictsLock.Unlock()
	return tmp
}

type benchDicts struct {
	cd *CDict
	dd *DDict
}

var benchDictsMap = make(map[int]*benchDicts)
var benchDictsLock sync.Mutex

func newBenchDicts(level int) *benchDicts {
	var samples [][]byte
	for i := 0; i < 300; i++ {
		sampleLen := rand.Intn(300)
		sample := newBenchString(sampleLen)
		samples = append(samples, sample)
	}

	dict := BuildDict(samples, 32*1024)
	cd, err := NewCDictLevel(dict, level)
	if err != nil {
		panic(fmt.Errorf("cannot create CDict: %s", err))
	}
	dd, err := NewDDict(dict)
	if err != nil {
		panic(fmt.Errorf("cannot create DDict: %s", err))
	}
	return &benchDicts{
		cd: cd,
		dd: dd,
	}
}

func newBenchString(blockSize int) []byte {
	var bb bytes.Buffer
	line := 0
	for bb.Len() < blockSize {
		fmt.Fprintf(&bb, "line %d, size %d, hex %08X\n", line, bb.Len(), line)
		line++
	}
	return bb.Bytes()[:blockSize]
}

func BenchmarkCompress(b *testing.B) {
	for _, blockSize := range benchBlockSizes {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, level := range benchCompressionLevels {
				b.Run(fmt.Sprintf("level_%d", level), func(b *testing.B) {
					benchmarkCompress(b, blockSize, level)
				})
			}
		})
	}
}

func benchmarkCompress(b *testing.B, blockSize, level int) {
	src := newBenchString(blockSize)
	b.ReportAllocs()
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		n := 0
		var dst []byte
		for pb.Next() {
			dst = CompressLevel(dst[:0], src, level)
			n += len(dst)
		}
		atomic.AddUint64(&Sink, uint64(n))
	})
}

func BenchmarkDecompress(b *testing.B) {
	for _, blockSize := range benchBlockSizes {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, level := range benchCompressionLevels {
				b.Run(fmt.Sprintf("level_%d", level), func(b *testing.B) {
					benchmarkDecompress(b, blockSize, level)
				})
			}
		})
	}
}

func benchmarkDecompress(b *testing.B, blockSize, level int) {
	block := newBenchString(blockSize)
	src := CompressLevel(nil, block, level)
	b.Logf("compressionRatio: %f", float64(len(block))/float64(len(src)))
	b.ReportAllocs()
	b.SetBytes(int64(len(block)))
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
