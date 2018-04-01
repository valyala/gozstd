package gozstd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestReaderBadUnderlyingReader(t *testing.T) {
	r := &badReader{
		b: Compress(nil, []byte(newTestString(64*1024, 30))),
	}
	zr := NewReader(r)

	buf := make([]byte, 123)
	for {
		if _, err := zr.Read(buf); err != nil {
			if !strings.Contains(err.Error(), "badReader failed") {
				t.Fatalf("unexpected error: %s", err)
			}
			break
		}
	}
}

type badReader struct {
	b []byte
}

func (br *badReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if rand.Intn(5) == 0 || len(br.b) < 2 {
		return 0, fmt.Errorf("badReader failed")
	}
	n := copy(p[:1], br.b)
	br.b = br.b[n:]
	return n, nil
}

func TestReaderInvalidData(t *testing.T) {
	// Try decompressing invalid data.
	src := []byte("invalid compressed data")

	r := bytes.NewReader(src)
	zr := NewReader(r)

	if _, err := ioutil.ReadAll(zr); err == nil {
		t.Fatalf("expecting error when decompressing invalid data")
	}

	// Try decompressing corrupted data.
	s := newTestString(64*1024, 15)
	cd := Compress(nil, []byte(s))
	cd[len(cd)-1]++

	r = bytes.NewReader(cd)
	zr.Reset(r)

	if _, err := ioutil.ReadAll(zr); err == nil {
		t.Fatalf("expecting error when decompressing corrupted data")
	}
}

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

	// Verify reading zero bytes
	n, err := zr.Read(buf[:0])
	if err != nil {
		return fmt.Errorf("cannot read zero bytes: %s", err)
	}
	if n != 0 {
		return fmt.Errorf("unexpected number of bytes read; got %d; want %d", n, 0)
	}

	// Verify reading random number of bytes.
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
