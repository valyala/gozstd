package gozstd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
)

func ExampleReader() {
	// Compress the data.
	compressedData := Compress(nil, []byte("line 0\nline 1\nline 2"))

	// Read it via Reader.
	r := bytes.NewReader(compressedData)
	zr := NewReader(r)
	defer zr.Release()

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

func ExampleReader_Reset() {
	zr := NewReader(nil)
	defer zr.Release()

	// Read from different sources using the same Reader.
	for i := 0; i < 3; i++ {
		compressedData := Compress(nil, []byte(fmt.Sprintf("line %d", i)))
		r := bytes.NewReader(compressedData)
		zr.Reset(r, nil)

		data, err := ioutil.ReadAll(zr)
		if err != nil {
			log.Fatalf("unexpected error when reading compressed data: %s", err)
		}
		fmt.Printf("%s\n", data)
	}

	// Output:
	// line 0
	// line 1
	// line 2
}
