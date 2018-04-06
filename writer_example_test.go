package gozstd

import (
	"bytes"
	"fmt"
	"io"
	"log"
)

func ExampleWriter() {

	// Compress data to bb.
	var bb bytes.Buffer
	zw := NewWriter(&bb)
	defer zw.Release()

	for i := 0; i < 3; i++ {
		fmt.Fprintf(zw, "line %d\n", i)
	}
	if err := zw.Close(); err != nil {
		log.Fatalf("cannot close writer: %s", err)
	}

	// Decompress the data and verify it is valid.
	plainData, err := Decompress(nil, bb.Bytes())
	fmt.Printf("err: %v\n%s", err, plainData)

	// Output:
	// err: <nil>
	// line 0
	// line 1
	// line 2
}

func ExampleWriter_Flush() {

	var bb bytes.Buffer
	zw := NewWriter(&bb)
	defer zw.Release()

	// Write some data to zw.
	data := []byte("some data\nto compress")
	if _, err := zw.Write(data); err != nil {
		log.Fatalf("cannot write data to zw: %s", err)
	}

	// Verify the data is cached in zw and isn't propagated to bb.
	if bb.Len() > 0 {
		log.Fatalf("%d bytes unexpectedly propagated to bb", bb.Len())
	}

	// Flush the compressed data to bb.
	if err := zw.Flush(); err != nil {
		log.Fatalf("cannot flush compressed data: %s", err)
	}

	// Verify the compressed data is propagated to bb.
	if bb.Len() == 0 {
		log.Fatalf("the compressed data isn't propagated to bb")
	}

	// Try reading the compressed data with reader.
	zr := NewReader(&bb)
	defer zr.Release()
	buf := make([]byte, len(data))
	if _, err := io.ReadFull(zr, buf); err != nil {
		log.Fatalf("cannot read the compressed data: %s", err)
	}
	fmt.Printf("%s", buf)

	// Output:
	// some data
	// to compress
}

func ExampleWriter_Reset() {
	zw := NewWriter(nil)
	defer zw.Release()

	// Write to different destinations using the same Writer.
	for i := 0; i < 3; i++ {
		var bb bytes.Buffer
		zw.Reset(&bb, nil, DefaultCompressionLevel)
		if _, err := zw.Write([]byte(fmt.Sprintf("line %d", i))); err != nil {
			log.Fatalf("unexpected error when writing data: %s", err)
		}
		if err := zw.Close(); err != nil {
			log.Fatalf("unexpected error when closing zw: %s", err)
		}

		// Decompress the compressed data.
		plainData, err := Decompress(nil, bb.Bytes())
		if err != nil {
			log.Fatalf("unexpected error when decompressing data: %s", err)
		}
		fmt.Printf("%s\n", plainData)
	}

	// Output:
	// line 0
	// line 1
	// line 2
}
