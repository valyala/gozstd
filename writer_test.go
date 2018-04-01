package gozstd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"
)

func TestWriter(t *testing.T) {
	testWriter(t, "")
	testWriter(t, "a")
	testWriter(t, "foo bar")
	testWriter(t, "aasdf sdfa dsa fdsaf dsa")

	for size := 1; size <= 4e5; size *= 2 {
		s := newTestString(size, 20)
		testWriter(t, s)
	}
}

func testWriter(t *testing.T, s string) {
	t.Helper()

	// Serial test
	if err := testWriterSerial(s); err != nil {
		t.Fatalf("error in serial writer test: %s", err)
	}

	// Concurrent test
	ch := make(chan error, 10)
	for i := 0; i < cap(ch); i++ {
		go func() {
			ch <- testWriterSerial(s)
		}()
	}

	for i := 0; i < cap(ch); i++ {
		select {
		case err := <-ch:
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}
}

func testWriterSerial(s string) error {
	zw := NewWriter(nil)
	for i := 0; i < 2; i++ {
		var bb bytes.Buffer
		zw.Reset(&bb)
		if err := testWriterExt(zw, s); err != nil {
			return err
		}
		cd := bb.Bytes()

		// Use Decompress.
		dd, err := Decompress(nil, cd)
		if err != nil {
			return fmt.Errorf("unexpected error when decompressing data: %s", err)
		}
		if string(dd) != s {
			return fmt.Errorf("unexpected data after the decompression; got\n%X; want\n%X", dd, s)
		}

		// Use Reader
		zr := NewReader(&bb)
		dd, err = ioutil.ReadAll(zr)
		if err != nil {
			return fmt.Errorf("unexpected error when reading compressed data: %s", err)
		}
		if string(dd) != s {
			return fmt.Errorf("unexpected data after reading compressed data; got\n%X; want\n%X", dd, s)
		}
	}
	return nil
}

func testWriterExt(zw *Writer, s string) error {
	bs := []byte(s)

	// Verify writing zero bytes.
	n, err := zw.Write(bs[:0])
	if err != nil {
		return fmt.Errorf("cannot write zero-byte value: %s", err)
	}
	if n != 0 {
		return fmt.Errorf("unexpected number of bytes written; got %d; want %d", n, 0)
	}

	// Verify writing random number of bytes.
	i := 0
	for i < len(bs) {
		nWant := rand.Intn(len(bs)-i)/7 + 1
		n, err := zw.Write(bs[i : i+nWant])
		if err != nil {
			return fmt.Errorf("unexpected error when writing data: %s", err)
		}
		if n != nWant {
			return fmt.Errorf("unexpected number of bytes written; got %d; want %d", n, nWant)
		}
		i += nWant
	}
	if err := zw.Flush(); err != nil {
		return fmt.Errorf("unexpected error when flushing data: %s", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("unexpected error when closing zw: %s", err)
	}
	return nil
}
