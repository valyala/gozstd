package gozstd_test

import (
	gozstd "."
	"bytes"
	"fmt"
	"io"
	"log"
)

func ExampleReader() {
	// Compress the data.
	compressedData := gozstd.Compress(nil, []byte("line 0\nline 1\nline 2"))

	// Read it via gozstd.Reader.
	r := bytes.NewReader(compressedData)
	zr := gozstd.NewReader(r)

	var a []int
	for i := 0; i < 3; i++ {
		var n int
		if _, err := fmt.Fscanf(zr, "line %d\n", &n); err != nil {
			log.Fatalf("cannot read line: %s", err)
		}
		a = append(a, n)
	}

	// Make sure there are no data left in zr.
	buf := make([]byte, 1)
	if _, err := zr.Read(buf); err != io.EOF {
		log.Fatalf("unexpected error; got %v; want %v", err, io.EOF)
	}

	fmt.Println(a)

	// Output:
	// [0 1 2]
}
