package gozstd

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestStreamCompressDecompress(t *testing.T) {
	testStreamCompressDecompress(t, "")
	testStreamCompressDecompress(t, "a")
	testStreamCompressDecompress(t, "foo bar")

	for blockSize := range []int{11, 111, 1111, 11111, 111111, 211111} {
		data := newTestString(blockSize, 3)
		testStreamCompressDecompress(t, data)
	}
}

func testStreamCompressDecompress(t *testing.T, data string) {
	t.Helper()

	// Serial test.
	if err := testStreamCompressDecompressSerial(data); err != nil {
		t.Fatalf("error in serial test: %s", err)
	}

	// Concurrent test.
	ch := make(chan error, 3)
	for i := 0; i < cap(ch); i++ {
		go func() {
			ch <- testStreamCompressDecompressSerial(data)
		}()
	}
	for i := 0; i < cap(ch); i++ {
		select {
		case err := <-ch:
			if err != nil {
				t.Fatalf("error in concurrent test: %s", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("timeout in concurrent test")
		}
	}
}

func testStreamCompressDecompressSerial(data string) error {
	var bbCompress bytes.Buffer
	if err := StreamCompress(&bbCompress, bytes.NewBufferString(data)); err != nil {
		return fmt.Errorf("cannot compress stream of size %d: %s", len(data), err)
	}

	var bbDecompress bytes.Buffer
	if err := StreamDecompress(&bbDecompress, &bbCompress); err != nil {
		return fmt.Errorf("cannot decompress stream of size %d: %s", len(data), err)
	}
	plainData := bbDecompress.Bytes()
	if string(plainData) != data {
		return fmt.Errorf("unexpected decompressed data; got\n%q; want\n%q", plainData, data)
	}
	return nil
}

func TestStreamCompressDecompressLevel(t *testing.T) {
	for level := 0; level < 20; level++ {
		t.Run(fmt.Sprintf("level_%d", level), func(t *testing.T) {
			testStreamCompressDecompressLevel(t, "", level)
			testStreamCompressDecompressLevel(t, "a", level)
			testStreamCompressDecompressLevel(t, "foo bar", level)

			for blockSize := range []int{11, 111, 1111, 11111, 143333} {
				data := newTestString(blockSize, 3)
				testStreamCompressDecompressLevel(t, data, level)
			}
		})
	}
}

func testStreamCompressDecompressLevel(t *testing.T, data string, level int) {
	t.Helper()

	// Serial test.
	if err := testStreamCompressDecompressLevelSerial(data, level); err != nil {
		t.Fatalf("error in serial test: %s", err)
	}

	// Concurrent test.
	ch := make(chan error, 3)
	for i := 0; i < cap(ch); i++ {
		go func() {
			ch <- testStreamCompressDecompressLevelSerial(data, level)
		}()
	}
	for i := 0; i < cap(ch); i++ {
		select {
		case err := <-ch:
			if err != nil {
				t.Fatalf("error in concurrent test: %s", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout in concurrent test")
		}
	}
}

func testStreamCompressDecompressLevelSerial(data string, level int) error {
	var bbCompress bytes.Buffer
	if err := StreamCompressLevel(&bbCompress, bytes.NewBufferString(data), level); err != nil {
		return fmt.Errorf("cannot compress stream of size %d: %s", len(data), err)
	}

	var bbDecompress bytes.Buffer
	if err := StreamDecompress(&bbDecompress, &bbCompress); err != nil {
		return fmt.Errorf("cannot decompress stream of size %d: %s", len(data), err)
	}
	plainData := bbDecompress.Bytes()
	if string(plainData) != data {
		return fmt.Errorf("unexpected decompressed data; got\n%q; want\n%q", plainData, data)
	}
	return nil
}

func TestStreamCompressDecompressDict(t *testing.T) {
	var samples [][]byte
	for i := 0; i < 1000; i++ {
		sample := fmt.Sprintf("this is a dict sample line %d", i)
		samples = append(samples, []byte(sample))
	}
	dict := BuildDict(samples, 8*1024)

	cd, err := NewCDict(dict)
	if err != nil {
		t.Fatalf("cannot create CDict: %s", err)
	}
	defer cd.Release()

	dd, err := NewDDict(dict)
	if err != nil {
		t.Fatalf("cannot create DDict: %s", err)
	}
	defer dd.Release()

	// Create data for the compression.
	var bb bytes.Buffer
	for bb.Len() < 256*1024 {
		fmt.Fprintf(&bb, "dict sample line %d this is", bb.Len())
	}
	data := bb.Bytes()

	// Serial test.
	if err := testStreamCompressDecompressDictSerial(cd, dd, data); err != nil {
		t.Fatalf("error in serial test: %s", err)
	}

	// Concurrent test.
	ch := make(chan error, 3)
	for i := 0; i < cap(ch); i++ {
		go func() {
			ch <- testStreamCompressDecompressDictSerial(cd, dd, data)
		}()
	}
	for i := 0; i < cap(ch); i++ {
		select {
		case err := <-ch:
			if err != nil {
				t.Fatalf("error in concurrent test: %s", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("timeout in concurrent test")
		}
	}
}

func testStreamCompressDecompressDictSerial(cd *CDict, dd *DDict, data []byte) error {
	var bbCompress bytes.Buffer
	if err := StreamCompressDict(&bbCompress, bytes.NewReader(data), cd); err != nil {
		return fmt.Errorf("cannot compress stream of size %d: %s", len(data), err)
	}

	var bbDecompress bytes.Buffer
	if err := StreamDecompressDict(&bbDecompress, &bbCompress, dd); err != nil {
		return fmt.Errorf("cannot decompress stream of size %d: %s", len(data), err)
	}
	plainData := bbDecompress.Bytes()
	if !bytes.Equal(plainData, data) {
		return fmt.Errorf("unexpected decompressed data; got\n%q; want\n%q", plainData, data)
	}
	return nil
}
