package gozstd

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func TestDecompressInvalidData(t *testing.T) {
	// Try decompressing invalid data.
	src := []byte("invalid compressed data")
	buf := make([]byte, len(src))
	if _, err := Decompress(nil, src); err == nil {
		t.Fatalf("expecting error when decompressing invalid data")
	}
	if _, err := Decompress(buf[:0], src); err == nil {
		t.Fatalf("expecting error when decompressing invalid data into existing buffer")
	}

	// Try decompressing corrupted data.
	s := newTestString(64*1024, 15)
	cd := Compress(nil, []byte(s))
	cd[len(cd)-1]++

	if _, err := Decompress(nil, cd); err == nil {
		t.Fatalf("expecting error when decompressing corrupted data")
	}
	if _, err := Decompress(buf[:0], cd); err == nil {
		t.Fatalf("expecting error when decompressing corrupdate data into existing buffer")
	}
}

func TestCompressLevel(t *testing.T) {
	src := []byte("foobar baz")

	for compressLevel := 1; compressLevel < 22; compressLevel++ {
		testCompressLevel(t, src, compressLevel)
	}

	// Test invalid compression levels - they should clamp
	// to the closest valid levels.
	testCompressLevel(t, src, -123)
	testCompressLevel(t, src, 234324)
}

func testCompressLevel(t *testing.T, src []byte, compressionLevel int) {
	t.Helper()

	cd := CompressLevel(nil, src, compressionLevel)
	dd, err := Decompress(nil, cd)
	if err != nil {
		t.Fatalf("unexpected error during decompression: %s", err)
	}
	if string(dd) != string(src) {
		t.Fatalf("unexpected dd\n%X; want\n%X", dd, src)
	}
}

func TestCompressDecompress(t *testing.T) {
	testCompressDecompress(t, "")
	testCompressDecompress(t, "a")
	testCompressDecompress(t, "foo bar")

	for size := 1; size <= 1e6; size *= 10 {
		s := newTestString(size, 20)
		testCompressDecompress(t, s)
	}
}

func testCompressDecompress(t *testing.T, s string) {
	t.Helper()

	if err := testCompressDecompressSerial(s); err != nil {
		t.Fatalf("error in serial test: %s", err)
	}

	ch := make(chan error, runtime.GOMAXPROCS(-1)+2)
	for i := 0; i < cap(ch); i++ {
		go func() {
			ch <- testCompressDecompressSerial(s)
		}()
	}
	for i := 0; i < cap(ch); i++ {
		select {
		case err := <-ch:
			if err != nil {
				t.Fatalf("unexpected error in parallel test: %s", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("timeout in parallel test")
		}
	}
}

func testCompressDecompressSerial(s string) error {
	cs := Compress(nil, []byte(s))
	ds, err := Decompress(nil, cs)
	if err != nil {
		return fmt.Errorf("cannot decompress: %s\ns=%X\ncs=%X", err, s, cs)
	}
	if string(ds) != s {
		return fmt.Errorf("unexpected ds (len=%d, sLen=%d, cslen=%d)\n%X; want\n%X", len(ds), len(s), len(cs), ds, s)
	}

	// Verify prefixed decompression.
	prefix := []byte("foobaraaa")
	ds, err = Decompress(prefix, cs)
	if err != nil {
		return fmt.Errorf("cannot decompress prefixed cs: %s\ns=%X\ncs=%X", err, s, cs)
	}
	if string(ds[:len(prefix)]) != string(prefix) {
		return fmt.Errorf("unexpected prefix in the decompressed result: %X; want %X", ds[:len(prefix)], prefix)
	}
	ds = ds[len(prefix):]
	if string(ds) != s {
		return fmt.Errorf("unexpected prefixed ds\n%X; want\n%X", ds, s)
	}

	// Verify prefixed compression.
	csp := Compress(prefix, []byte(s))
	if string(csp[:len(prefix)]) != string(prefix) {
		return fmt.Errorf("unexpected prefix in the compressed result: %X; want %X", csp[:len(prefix)], prefix)
	}
	csp = csp[len(prefix):]
	if string(csp) != string(cs) {
		return fmt.Errorf("unexpected prefixed cs\n%X; want\n%X", csp, cs)
	}
	return nil
}

func newTestString(size, randomness int) string {
	s := make([]byte, size)
	for i := 0; i < size; i++ {
		s[i] = byte(rand.Intn(randomness))
	}
	return string(s)
}
