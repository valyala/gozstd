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

func TestReaderReadCompressBomb(t *testing.T) {
	// Compress easily compressible string with size greater
	// than the dstreamOutBufSize.
	// This string shuld be compressed into a short byte slice,
	// which should be then decompressed into a big buffer
	// with size greater than dstreamOutBufSize.
	// This means the Reader.outBuf capacity isn't enough to hold
	// all the decompressed data.

	var bb bytes.Buffer
	zw := NewWriter(&bb)
	s := newTestString(int(2*dstreamOutBufSize), 1)
	n, err := zw.Write([]byte(s))
	if err != nil {
		t.Fatalf("unexpected error in Writer.Write: %s", err)
	}
	if n != len(s) {
		t.Fatalf("unexpected number of bytes written; got %d; want %d", n, len(s))
	}
	if err := zw.Flush(); err != nil {
		t.Fatalf("cannot flush data: %s", err)
	}

	zr := NewReader(&bb)
	buf := make([]byte, len(s))
	n, err = io.ReadFull(zr, buf)
	if err != nil {
		t.Fatalf("unexpected error in io.ReadFull: %s", err)
	}
	if n != len(s) {
		t.Fatalf("unexpected number of bytes read; got %d; want %d", n, len(s))
	}
	if string(buf) != s {
		t.Fatalf("unexpected data read;\ngot\n%X\nwant\n%X", buf, s)
	}

	// Free resources.
	zw.Close()
	zw.Release()
	zr.Release()
}

func TestReaderWriteTo(t *testing.T) {
	data := newTestString(130*1024, 3)
	compressedData := Compress(nil, []byte(data))
	zr := NewReader(bytes.NewReader(compressedData))
	defer zr.Release()

	var bb bytes.Buffer
	n, err := zr.WriteTo(&bb)
	if err != nil {
		t.Fatalf("cannot write data from zr to bb: %s", err)
	}
	if n != int64(bb.Len()) {
		t.Fatalf("unexpected number of bytes written; got %d; want %d", n, bb.Len())
	}
	plainData := bb.Bytes()
	if string(plainData) != data {
		t.Fatalf("unexpected data decompressed; got\n%X; want\n%X", plainData, data)
	}
}

func TestReaderDict(t *testing.T) {
	var samples [][]byte
	for i := 0; i < 1e4; i++ {
		sample := []byte(fmt.Sprintf("this is a sample number %d", i))
		samples = append(samples, sample)
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

	// Run serial test.
	if err := testReaderDictSerial(cd, dd); err != nil {
		t.Fatalf("error in serial test: %s", err)
	}

	// Run concurrent test.
	ch := make(chan error, 3)
	for i := 0; i < cap(ch); i++ {
		go func() {
			ch <- testReaderDictSerial(cd, dd)
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

func testReaderDictSerial(cd *CDict, dd *DDict) error {
	var bb bytes.Buffer
	for i := 0; i < 8000; i++ {
		fmt.Fprintf(&bb, "This is number %d ", i)
	}
	origData := bb.Bytes()
	compressedData := CompressDict(nil, origData, cd)

	// Decompress via Reader.
	zr := NewReaderDict(bytes.NewReader(compressedData), dd)
	defer zr.Release()

	plainData, err := ioutil.ReadAll(zr)
	if err != nil {
		return fmt.Errorf("cannot stream decompress data with dict: %s", err)
	}
	if !bytes.Equal(plainData, origData) {
		return fmt.Errorf("unexpected stream uncompressed data; got\n%q; want\n%q\nlen(plainData)=%d, len(origData)=%d",
			plainData, origData, len(plainData), len(origData))
	}

	// Try decompressing without dict.
	zrNoDict := NewReader(bytes.NewReader(compressedData))
	defer zrNoDict.Release()

	_, err = ioutil.ReadAll(zrNoDict)
	if err == nil {
		return fmt.Errorf("expecting non-nil error when stream decompressing without dict")
	}
	if !strings.Contains(err.Error(), "Dictionary mismatch") {
		return fmt.Errorf("unexpected error when stream decompressing without dict; got %q; want %q", err, "Dictionary mismatch")
	}
	return nil
}

func TestReaderMultiFrames(t *testing.T) {
	var bb bytes.Buffer
	for bb.Len() < 3*128*1024 {
		fmt.Fprintf(&bb, "reader big data %d, ", bb.Len())
	}
	origData := append([]byte{}, bb.Bytes()...)

	cd := Compress(nil, bb.Bytes())

	r := bytes.NewReader(cd)
	zr := NewReader(r)
	defer zr.Release()
	plainData, err := ioutil.ReadAll(zr)
	if err != nil {
		t.Fatalf("cannot read big data: %s", err)
	}
	if !bytes.Equal(plainData, origData) {
		t.Fatalf("unexpected data read: got\n%q; want\n%q\nlen(data)=%d, len(orig)=%d",
			plainData, origData, len(plainData), len(origData))
	}
}

func TestReaderBadUnderlyingReader(t *testing.T) {
	r := &badReader{
		b: Compress(nil, []byte(newTestString(64*1024, 30))),
	}
	zr := NewReader(r)
	defer zr.Release()

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
	defer zr.Release()

	if _, err := ioutil.ReadAll(zr); err == nil {
		t.Fatalf("expecting error when decompressing invalid data")
	}

	// Try decompressing corrupted data.
	s := newTestString(64*1024, 15)
	cd := Compress(nil, []byte(s))
	cd[len(cd)-1]++

	r = bytes.NewReader(cd)
	zr.Reset(r, nil)

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
	defer zr.Release()
	for i := 0; i < 2; i++ {
		r := bytes.NewReader(cd)
		zr.Reset(r, nil)
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
