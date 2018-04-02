package gozstd

import (
	"fmt"
	"log"
)

func ExampleCompress_simple() {
	data := []byte("foo bar baz")

	// Compress and decompress data into new buffers.
	compressedData := Compress(nil, data)
	decompressedData, err := Decompress(nil, compressedData)
	if err != nil {
		log.Fatalf("cannot decompress data: %s", err)
	}

	fmt.Printf("%s", decompressedData)
	// Output:
	// foo bar baz
}

func ExampleDecompress_simple() {
	data := []byte("foo bar baz")

	// Compress and decompress data into new buffers.
	compressedData := Compress(nil, data)
	decompressedData, err := Decompress(nil, compressedData)
	if err != nil {
		log.Fatalf("cannot decompress data: %s", err)
	}

	fmt.Printf("%s", decompressedData)
	// Output:
	// foo bar baz
}

func ExampleCompress_noAllocs() {
	data := []byte("foo bar baz")

	// Allocate a buffer for compressed results.
	cbuf := make([]byte, 100)

	for i := 0; i < 3; i++ {
		// If the size of compressed data fits cap(cbuf), then Compress
		// will put compressed data into cbuf without additional allocations.
		compressedData := Compress(cbuf[:0], data)

		// Verify Compress returned cbuf instead of allocating new one.
		if &compressedData[0] != &cbuf[0] {
			log.Fatalf("Compress returned new cbuf; got %p; want %p", &compressedData[0], &cbuf[0])
		}

		decompressedData, err := Decompress(nil, compressedData)
		if err != nil {
			log.Fatalf("cannot decompress data: %s", err)
		}

		fmt.Printf("%d. %s\n", i, decompressedData)
	}

	// Output:
	// 0. foo bar baz
	// 1. foo bar baz
	// 2. foo bar baz
}

func ExampleDecompress_noAllocs() {
	data := []byte("foo bar baz")

	compressedData := Compress(nil, data)

	// Allocate a buffer for decompressed results.
	dbuf := make([]byte, 100)

	for i := 0; i < 3; i++ {
		// If the size of decompressed data fits cap(dbuf), then Decompress
		// will put decompressed data into dbuf without additional allocations.
		decompressedData, err := Decompress(dbuf[:0], compressedData)
		if err != nil {
			log.Fatalf("cannot decompress data: %s", err)
		}

		// Verify Decompress returned dbuf instead of allocating new one.
		if &decompressedData[0] != &dbuf[0] {
			log.Fatalf("Decompress returned new dbuf; got %p; want %p", &decompressedData[0], &dbuf[0])
		}

		fmt.Printf("%d. %s\n", i, decompressedData)
	}

	// Output:
	// 0. foo bar baz
	// 1. foo bar baz
	// 2. foo bar baz
}
