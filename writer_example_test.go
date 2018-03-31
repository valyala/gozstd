package gozstd_test

import (
	gozstd "."
	"bytes"
	"fmt"
	"io"
	"log"
)

func ExampleWriter() {

	// Compress data to bb.
	var bb bytes.Buffer
	w := gozstd.NewWriter(&bb)
	for i := 0; i < 3; i++ {
		fmt.Fprintf(w, "line %d\n", i)
	}
	if err := w.Close(); err != nil {
		log.Fatalf("cannot close writer: %s", err)
	}

	// Decompress the data and verify it is valid.
	plainData, err := gozstd.Decompress(nil, bb.Bytes())
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
	zr := gozstd.NewReader(r)
	zw := gozstd.NewWriter(w)

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
