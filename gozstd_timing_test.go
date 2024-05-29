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
					benchmarkDecompressDict(b, blockSize, level, false)
				})
			}
		})
	}
}

func BenchmarkDecompressDictByRef(b *testing.B) {
	for _, blockSize := range benchBlockSizes {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, level := range benchCompressionLevels {
				b.Run(fmt.Sprintf("level_%d", level), func(b *testing.B) {
					benchmarkDecompressDict(b, blockSize, level, true)
				})
			}
		})
	}
}

func benchmarkDecompressDict(b *testing.B, blockSize, level int, byReference bool) {
	block := newBenchString(blockSize)
	var bd *benchDicts
	if byReference {
		bd = getBenchDictsByRef(level)
	} else {
		bd = getBenchDicts(level)
	}
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
					benchmarkCompressDict(b, blockSize, level, false)
				})
			}
		})
	}
}

func BenchmarkCompressDictByRef(b *testing.B) {
	for _, blockSize := range benchBlockSizes {
		b.Run(fmt.Sprintf("blockSize_%d", blockSize), func(b *testing.B) {
			for _, level := range benchCompressionLevels {
				b.Run(fmt.Sprintf("level_%d", level), func(b *testing.B) {
					benchmarkCompressDict(b, blockSize, level, true)
				})
			}
		})
	}
}

func benchmarkCompressDict(b *testing.B, blockSize, level int, byReference bool) {
	src := newBenchString(blockSize)
	var bd *benchDicts
	if byReference {
		bd = getBenchDictsByRef(level)
	} else {
		bd = getBenchDicts(level)
	}
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

func getBenchDictsByRef(level int) *benchDicts {
	benchDictsByRefLock.Lock()
	tmp := benchDictsByRefMap[level]
	if tmp == nil {
		tmp = newBenchDictsByRef(level)
		benchDictsByRefMap[level] = tmp
	}
	benchDictsByRefLock.Unlock()
	return tmp
}

type benchDicts struct {
	cd *CDict
	dd *DDict
}

var (
	benchDictsMap       = make(map[int]*benchDicts)
	benchDictsLock      sync.Mutex
	benchDictsByRefMap  = make(map[int]*benchDicts)
	benchDictsByRefLock sync.Mutex
)

func newBenchDicts(level int) *benchDicts {
	return createNewBenchDicts(NewCDictLevel, NewDDict, level)
}

func newBenchDictsByRef(level int) *benchDicts {
	return createNewBenchDicts(NewCDictLevelByRef, NewDDictByRef, level)
}

// Make it easier to toggle between copying the underlying bytes on creation
// vs. sharing by reference.
type (
	cdictFactory func(dict []byte, level int) (*CDict, error)
	ddictFactory func(dict []byte) (*DDict, error)
)

func createNewBenchDicts(createCDict cdictFactory, createDDict ddictFactory, level int) *benchDicts {
	var samples [][]byte
	for i := 0; i < 300; i++ {
		sampleLen := rand.Intn(300)
		sample := newBenchString(sampleLen)
		samples = append(samples, sample)
	}

	dict := BuildDict(samples, 32*1024)
	cd, err := createCDict(dict, level)
	if err != nil {
		panic(fmt.Errorf("cannot create CDict: %s", err))
	}
	dd, err := createDDict(dict)
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
