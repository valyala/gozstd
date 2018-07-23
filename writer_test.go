package gozstd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestWriterReadFrom(t *testing.T) {
	var bb bytes.Buffer
	zw := NewWriter(&bb)
	defer zw.Release()

	data := newTestString(132*1024, 3)
	n, err := zw.ReadFrom(bytes.NewBufferString(data))
	if err != nil {
		t.Fatalf("cannot read data to zw: %s", err)
	}
	if n != int64(len(data)) {
		t.Fatalf("unexpected number of bytes read; got %d; want %d", n, len(data))
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("cannot close zw: %s", err)
	}

	plainData, err := Decompress(nil, bb.Bytes())
	if err != nil {
		t.Fatalf("cannot decompress data: %s", err)
	}
	if string(plainData) != data {
		t.Fatalf("unexpected data decompressed; got\n%X; want\n%X", plainData, data)
	}
}

func TestNewWriterLevel(t *testing.T) {
	src := []byte(newTestString(512, 3))
	for level := 0; level < 23; level++ {
		var bb bytes.Buffer
		zw := NewWriterLevel(&bb, level)
		_, err := io.Copy(zw, bytes.NewReader(src))
		if err != nil {
			t.Fatalf("error when compressing on level %d: %s", level, err)
		}
		if err := zw.Close(); err != nil {
			t.Fatalf("error when closing zw on level %d: %s", level, err)
		}
		zw.Release()

		plainData, err := Decompress(nil, bb.Bytes())
		if err != nil {
			t.Fatalf("cannot decompress data on level %d: %s", level, err)
		}
		if !bytes.Equal(plainData, src) {
			t.Fatalf("unexpected data obtained after decompression on level %d; got\n%X; want\n%X", level, plainData, src)
		}
	}
}

func TestWriterDict(t *testing.T) {
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
	if err := testWriterDictSerial(cd, dd); err != nil {
		t.Fatalf("error in serial test: %s", err)
	}

	// Run concurrent test.
	ch := make(chan error, 3)
	for i := 0; i < cap(ch); i++ {
		go func() {
			ch <- testWriterDictSerial(cd, dd)
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

func testWriterDictSerial(cd *CDict, dd *DDict) error {
	var bb bytes.Buffer
	var bbOrig bytes.Buffer
	zw := NewWriterDict(&bb, cd)
	defer zw.Release()
	w := io.MultiWriter(zw, &bbOrig)
	for i := 0; i < 8000; i++ {
		if _, err := fmt.Fprintf(w, "This is number %d ", i); err != nil {
			return fmt.Errorf("error when writing data to zw: %s", err)
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("cannot close zw: %s", err)
	}

	// Decompress via Decompress.
	compressedData := bb.Bytes()
	plainData, err := DecompressDict(nil, compressedData, dd)
	if err != nil {
		return fmt.Errorf("cannot decompress data with dict: %s", err)
	}
	if !bytes.Equal(plainData, bbOrig.Bytes()) {
		return fmt.Errorf("unexpected uncompressed data; got\n%q; want\n%q\nlen(plainData)=%d, len(origData)=%d",
			plainData, bbOrig.Bytes(), len(plainData), bbOrig.Len())
	}

	// Decompress via Reader.
	zr := NewReaderDict(&bb, dd)
	defer zr.Release()

	plainData, err = ioutil.ReadAll(zr)
	if err != nil {
		return fmt.Errorf("cannot stream decompress data with dict: %s", err)
	}
	if !bytes.Equal(plainData, bbOrig.Bytes()) {
		return fmt.Errorf("unexpected stream uncompressed data; got\n%q; want\n%q\nlen(plainData)=%d, len(origData)=%d",
			plainData, bbOrig.Bytes(), len(plainData), bbOrig.Len())
	}

	// Try decompressing without dict.
	_, err = Decompress(nil, compressedData)
	if err == nil {
		return fmt.Errorf("expecting non-nil error when decompressing without dict")
	}
	if !strings.Contains(err.Error(), "Dictionary mismatch") {
		return fmt.Errorf("unexpected error when decompressing without dict; got %q; want %q", err, "Dictionary mismatch")
	}

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

func TestWriterMultiFrames(t *testing.T) {
	var bb bytes.Buffer
	var bbOrig bytes.Buffer
	zw := NewWriter(&bb)
	defer zw.Release()

	w := io.MultiWriter(zw, &bbOrig)
	for bbOrig.Len() < 3*128*1024 {
		if _, err := fmt.Fprintf(w, "writer big data %d, ", bbOrig.Len()); err != nil {
			t.Fatalf("unexpected error when writing to zw: %s", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("unexpected error when closing zw: %s", err)
	}

	plainData, err := Decompress(nil, bb.Bytes())
	if err != nil {
		t.Fatalf("cannot decompress big data: %s", err)
	}
	origData := bbOrig.Bytes()
	if !bytes.Equal(plainData, origData) {
		t.Fatalf("unexpected data decompressed: got\n%q; want\n%q\nlen(data)=%d, len(orig)=%d",
			plainData, origData, len(plainData), len(origData))
	}
}

func TestWriterBadUnderlyingWriter(t *testing.T) {
	zw := NewWriter(&badWriter{})
	defer zw.Release()
	data := []byte(newTestString(123, 20))
	for {
		if _, err := zw.Write(data); err != nil {
			if !strings.Contains(err.Error(), "badWriter failed") {
				t.Fatalf("unexpected error: %s", err)
			}
			break
		}
	}
}

type badWriter struct{}

func (*badWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if rand.Intn(10) == 0 {
		return 0, fmt.Errorf("badWriter failed")
	}
	return len(p), nil
}

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
	defer zw.Release()
	for i := 0; i < 2; i++ {
		var bb bytes.Buffer
		zw.Reset(&bb, nil, DefaultCompressionLevel)
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

func TestWriterBig(t *testing.T) {
	pr, pw := io.Pipe()
	zw := NewWriter(pw)
	zr := NewReader(pr)

	doneCh := make(chan error)
	var writtenBB bytes.Buffer
	go func() {
		sizeBuf := make([]byte, 8)
		for writtenBB.Len() < 3e6 {
			packetSize := rand.Intn(1000) + 1
			binary.BigEndian.PutUint64(sizeBuf, uint64(packetSize))
			if _, err := zw.Write(sizeBuf); err != nil {
				panic(fmt.Errorf("cannot write sizeBuf: %s", err))
			}
			s := newTestString(packetSize, 10)
			if _, err := zw.Write([]byte(s)); err != nil {
				panic(fmt.Errorf("cannot write packet with size %d: %s", packetSize, err))
			}
			writtenBB.WriteString(s)
		}
		binary.BigEndian.PutUint64(sizeBuf, 0)
		if _, err := zw.Write(sizeBuf); err != nil {
			panic(fmt.Errorf("cannot write `end of stream` packet: %s", err))
		}
		if err := zw.Flush(); err != nil {
			panic(fmt.Errorf("cannot flush data: %s", err))
		}
		doneCh <- nil
	}()

	var readBB bytes.Buffer
	sizeBuf := make([]byte, 8)
	for {
		if _, err := io.ReadFull(zr, sizeBuf); err != nil {
			t.Fatalf("cannot read sizeBuf: %s", err)
		}
		packetSize := binary.BigEndian.Uint64(sizeBuf)
		if packetSize == 0 {
			// end of stream.
			break
		}
		packetBuf := make([]byte, packetSize)
		if _, err := io.ReadFull(zr, packetBuf); err != nil {
			t.Fatalf("cannot read packetBuf: %s", err)
		}
		readBB.Write(packetBuf)
	}

	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout")
	}

	if writtenBB.Len() != readBB.Len() {
		t.Fatalf("non-equal lens for writtenBB and readBB: %d vs %d", writtenBB.Len(), readBB.Len())
	}
	if !bytes.Equal(writtenBB.Bytes(), readBB.Bytes()) {
		t.Fatalf("unequal writtenBB and readBB\nwrittenBB=\n%X\nreadBB=\n%X", writtenBB.Bytes(), readBB.Bytes())
	}
}
