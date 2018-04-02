package gozstd

import (
	"fmt"
	"log"
)

func ExampleBuildDict() {
	// Collect samples for the dictionary.
	var samples [][]byte
	for i := 0; i < 1000; i++ {
		sample := fmt.Sprintf("this is a sample number %d", i)
		samples = append(samples, []byte(sample))
	}

	// Build a dictionary with the desired size of 8Kb.
	dict := BuildDict(samples, 8*1024)

	// Now the dict may be used for compression/decompression.

	plainData := []byte("this is a data for the compression with dict")

	// Compress data with the dict.
	cd, err := NewCDict(dict)
	if err != nil {
		log.Fatalf("cannot create CDict: %s", err)
	}
	defer cd.Release()
	compressedData := CompressWithDict(nil, plainData, cd)

	// The compressedData must be decompressed with the same dict.
	dd, err := NewDDict(dict)
	if err != nil {
		log.Fatalf("cannot create DDict: %s", err)
	}
	defer dd.Release()
	decompressedData, err := DecompressWithDict(nil, compressedData, dd)
	if err != nil {
		log.Fatalf("cannot decompress data: %s", err)
	}

	fmt.Printf("%s", decompressedData)

	// Output:
	// this is a data for the compression with dict
}
