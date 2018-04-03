package gozstd

import (
	"fmt"
	"log"
)

func ExampleBuildDict() {
	// Collect samples for the dictionary.
	var samples [][]byte
	for i := 0; i < 1000; i++ {
		sample := fmt.Sprintf("this is a dict sample number %d", i)
		samples = append(samples, []byte(sample))
	}

	// Build a dictionary with the desired size of 8Kb.
	dict := BuildDict(samples, 8*1024)

	// Now the dict may be used for compression/decompression.

	// Create CDict from the dict.
	cd, err := NewCDict(dict)
	if err != nil {
		log.Fatalf("cannot create CDict: %s", err)
	}
	defer cd.Release()

	// Compress multiple blocks with the same CDict.
	var compressedBlocks [][]byte
	for i := 0; i < 3; i++ {
		plainData := fmt.Sprintf("this is line %d for dict compression", i)
		compressedData := CompressDict(nil, []byte(plainData), cd)
		compressedBlocks = append(compressedBlocks, compressedData)
	}

	// The compressedData must be decompressed with the same dict.

	// Create DDict from the dict.
	dd, err := NewDDict(dict)
	if err != nil {
		log.Fatalf("cannot create DDict: %s", err)
	}
	defer dd.Release()

	// Decompress multiple blocks with the same DDict.
	for _, compressedData := range compressedBlocks {
		decompressedData, err := DecompressDict(nil, compressedData, dd)
		if err != nil {
			log.Fatalf("cannot decompress data: %s", err)
		}
		fmt.Printf("%s\n", decompressedData)
	}

	// Output:
	// this is line 0 for dict compression
	// this is line 1 for dict compression
	// this is line 2 for dict compression
}
