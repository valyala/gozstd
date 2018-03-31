package gozstd

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"
)

func TestReader(t *testing.T) {
	testReader(t, "")
	testReader(t, "a")
	testReader(t, "foo bar")
	testReader(t, "aasdf sdfa dsa fdsaf dsa")

	for size := 1; size <= 4e5; size *= 2 {
		s := newTestString(size, 20)
		testReader(t, s)
	}
}

func testReader(t *testing.T, s string) {
	t.Helper()

	cd := Compress(nil, []byte(s))

	// Serial test
	if err := testReaderSerial(s, cd); err != nil {
		t.Fatalf("error in serial reader test: %s", err)
	}

	// Concurrent test
	ch := make(chan error, 10)
	for i := 0; i < cap(ch); i++ {
		go func() {
			ch <- testReaderSerial(s, cd)
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

func testReaderSerial(s string, cd []byte) error {
	zr := NewReader(nil)
	for i := 0; i < 2; i++ {
		r := bytes.NewReader(cd)
		zr.Reset(r)
		if err := testReaderExt(zr, s); err != nil {
			return err
		}
	}
	return nil
}

func testReaderExt(zr *Reader, s string) error {
	buf := make([]byte, len(s))
	for len(s) > 0 {
		nWant := rand.Intn(len(s))/7 + 1
		n, err := io.ReadFull(zr, buf[:nWant])
		if err != nil {
			return fmt.Errorf("unexpected error when reading data: %s", err)
		}
		if n != nWant {
			return fmt.Errorf("unexpected number of bytes read; got %d; want %d", n, nWant)
		}
		if string(buf[:n]) != s[:n] {
			return fmt.Errorf("unexpected data read: got\n%X; want\n%X", buf[:n], s[:n])
		}
		s = s[n:]
	}
	return nil
}
