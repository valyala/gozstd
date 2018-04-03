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

	// Compressed data will be put into cbuf.
	var cbuf []byte

	for i := 0; i < 5; i++ {
		// Compress re-uses cbuf for the compressed data.
		cbuf = Compress(cbuf[:0], data)

		decompressedData, err := Decompress(nil, cbuf)
		if err != nil {
			log.Fatalf("cannot decompress data: %s", err)
		}

		fmt.Printf("%d. %s\n", i, decompressedData)
	}

	// Output:
	// 0. foo bar baz
	// 1. foo bar baz
	// 2. foo bar baz
	// 3. foo bar baz
	// 4. foo bar baz
}

func ExampleDecompress_noAllocs() {
	data := []byte("foo bar baz")

	compressedData := Compress(nil, data)

	// Decompressed data will be put into dbuf.
	var dbuf []byte

	for i := 0; i < 5; i++ {
		// Decompress re-uses dbuf for the decompressed data.
		var err error
		dbuf, err = Decompress(dbuf[:0], compressedData)
		if err != nil {
			log.Fatalf("cannot decompress data: %s", err)
		}

		fmt.Printf("%d. %s\n", i, dbuf)
	}

	// Output:
	// 0. foo bar baz
	// 1. foo bar baz
	// 2. foo bar baz
	// 3. foo bar baz
	// 4. foo bar baz
}
