package gozstd

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestCDictEmpty(t *testing.T) {
	cd, err := NewCDict(nil)
	if err == nil {
		t.Fatalf("expecting non-nil error")
	}
	if cd != nil {
		t.Fatalf("expecting nil cd")
	}
}

func TestDDictEmpty(t *testing.T) {
	dd, err := NewDDict(nil)
	if err == nil {
		t.Fatalf("expecting non-nil error")
	}
	if dd != nil {
		t.Fatalf("expecting nil dd")
	}
}

func TestCDictCreateRelease(t *testing.T) {
	var samples [][]byte
	for i := 0; i < 1000; i++ {
		samples = append(samples, []byte(fmt.Sprintf("sample %d", i)))
	}
	dict := BuildDict(samples, 64*1024)

	for i := 0; i < 10; i++ {
		cd, err := NewCDict(dict)
		if err != nil {
			t.Fatalf("cannot create dict: %s", err)
		}
		cd.Release()
	}
}

func TestDDictCreateRelease(t *testing.T) {
	var samples [][]byte
	for i := 0; i < 1000; i++ {
		samples = append(samples, []byte(fmt.Sprintf("sample %d", i)))
	}
	dict := BuildDict(samples, 64*1024)

	for i := 0; i < 10; i++ {
		dd, err := NewDDict(dict)
		if err != nil {
			t.Fatalf("cannot create dict: %s", err)
		}
		dd.Release()
	}
}

func TestBuildDict(t *testing.T) {
	for _, samplesCount := range []int{0, 1, 10, 100, 1000} {
		t.Run(fmt.Sprintf("samples_%d", samplesCount), func(t *testing.T) {
			var samples [][]byte
			for i := 0; i < samplesCount; i++ {
				sample := []byte(fmt.Sprintf("sample %d, rand num %d, other num %X", i, rand.Intn(100), rand.Intn(100000)))
				samples = append(samples, sample)
				samples = append(samples, nil) // add empty sample
			}
			for _, desiredDictLen := range []int{20, 256, 1000, 10000} {
				t.Run(fmt.Sprintf("desiredDictLen_%d", desiredDictLen), func(t *testing.T) {
					testBuildDict(t, samples, desiredDictLen)
				})
			}
		})
	}
}

func testBuildDict(t *testing.T, samples [][]byte, desiredDictLen int) {
	t.Helper()

	// Serial test.
	dictOrig := BuildDict(samples, desiredDictLen)

	// Concurrent test.
	ch := make(chan error, 3)
	for i := 0; i < cap(ch); i++ {
		go func() {
			dict := BuildDict(samples, desiredDictLen)
			if string(dict) != string(dictOrig) {
				ch <- fmt.Errorf("unexpected dict; got\n%X; want\n%X", dict, dictOrig)
			}
			ch <- nil
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
