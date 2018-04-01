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

	// Create in-memory compressed pipe.
	r, w := io.Pipe()
	zr := NewReader(r)
	defer zr.Release()
	zw := NewWriter(w)
	defer zw.Release()

	// Start writer goroutine.
	readerReadyCh := make(chan int)
	writerDoneCh := make(chan struct{})
	go func() {
		for n := range readerReadyCh {
			fmt.Fprintf(zw, "line %d\n", n)

			// Flush the written line, so it may be read by the reader.
			if err := zw.Flush(); err != nil {
				log.Fatalf("unexpected error when flushing data: %s", err)
			}

		}
		if err := zw.Close(); err != nil {
			log.Fatalf("unexpected error when closing zw: %s", err)
		}
		if err := w.Close(); err != nil {
			log.Fatalf("unexpected error when closing w: %s", err)
		}
		close(writerDoneCh)
	}()

	// Read data from writer goroutine.
	var a []int
	for i := 0; i < 5; i++ {
		// Notify the writer we are ready for reading the next line.
		readerReadyCh <- i

		var n int
		_, err := fmt.Fscanf(zr, "line %d\n", &n)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("unexpected error when reading data: %s", err)
		}
		a = append(a, n)
	}

	// Notify the writer gorotine it must be finished.
	close(readerReadyCh)

	// Make sure the writer is closed.
	buf := make([]byte, 1)
	n, err := zr.Read(buf)
	if err != io.EOF {
		log.Fatalf("unexpected error: got %v; want %v; n=%d", err, io.EOF, n)
	}

	// Wait for writer goroutine to finish.
	<-writerDoneCh

	fmt.Println(a)

	// Output:
	// [0 1 2 3 4]
}

func ExampleWriter_Reset() {
	zw := NewWriter(nil)
	defer zw.Release()

	// Write to different destinations using the same Writer.
	for i := 0; i < 3; i++ {
		var bb bytes.Buffer
		zw.Reset(&bb)
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
