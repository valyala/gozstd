package gozstd

import (
	"fmt"
	"io"
)

type Writer struct {
}

// NewWriter returns new zstd writer writing compressed data to w.
//
// The returned writer must be closed with Close call in order
// to finalize the compressed stream.
//
// Call Release when the Writer is no longer needed.
func NewWriter(w io.Writer) *Writer {
	return &Writer{}
}

// NewWriterLevel returns new zstd writer writing compressed data to w
// at the given compression level.
//
// The returned writer must be closed with Close call in order
// to finalize the compressed stream.
//
// Call Release when the Writer is no longer needed.
func NewWriterLevel(w io.Writer, compressionLevel int) *Writer {
	return &Writer{}
}

// NewWriterDict returns new zstd writer writing compressed data to w
// using the given cd.
//
// The returned writer must be closed with Close call in order
// to finalize the compressed stream.
//
// Call Release when the Writer is no longer needed.
func NewWriterDict(w io.Writer, cd *CDict) *Writer {
	return &Writer{}
}

// A WriterParams allows users to specify compression parameters by calling
// NewWriterParams.
//
// Calling NewWriterParams with a nil WriterParams is equivalent to calling
// NewWriter.
type WriterParams struct {
	// Compression level. Special value 0 means 'default compression level'.
	CompressionLevel int

	// WindowLog. Must be clamped between WindowLogMin and WindowLogMin32/64.
	// Special value 0 means 'use default windowLog'.
	//
	// Note: enabling log distance matching increases memory usage for both
	// compressor and decompressor. When set to a value greater than 27, the
	// decompressor requires special treatment.
	WindowLog int

	// Dict is optional dictionary used for compression.
	Dict *CDict
}

// NewWriterParams returns new zstd writer writing compressed data to w
// using the given set of parameters.
//
// The returned writer must be closed with Close call in order
// to finalize the compressed stream.
//
// Call Release when the Writer is no longer needed.
func NewWriterParams(w io.Writer, params *WriterParams) *Writer {
	return &Writer{}
}

// Reset resets zw to write to w using the given dictionary cd and the given
// compressionLevel. Use ResetWriterParams if you wish to change other
// parameters that were set via WriterParams.
func (zw *Writer) Reset(w io.Writer, cd *CDict, compressionLevel int) {
}

// ResetWriterParams resets zw to write to w using the given set of parameters.
func (zw *Writer) ResetWriterParams(w io.Writer, params *WriterParams) {
}

func (zw *Writer) Release() {
}

// ReadFrom reads all the data from r and writes it to zw.
//
// Returns the number of bytes read from r.
//
// ReadFrom may not flush the compressed data to the underlying writer
// due to performance reasons.
// Call Flush or Close when the compressed data must propagate
// to the underlying writer.
func (zw *Writer) ReadFrom(r io.Reader) (int64, error) {
	return -1, fmt.Errorf("zstd not supported without cgo")
}

// Write writes p to zw.
//
// Write doesn't flush the compressed data to the underlying writer
// due to performance reasons.
// Call Flush or Close when the compressed data must propagate
// to the underlying writer.
func (zw *Writer) Write(p []byte) (int, error) {
	return -1, fmt.Errorf("zstd not supported without cgo")
}

// Flush flushes the remaining data from zw to the underlying writer.
func (zw *Writer) Flush() error {
	return fmt.Errorf("zstd not supported without cgo")
}

// Close finalizes the compressed stream and flushes all the compressed data
// to the underlying writer.
//
// It doesn't close the underlying writer passed to New* functions.
func (zw *Writer) Close() error {
	return fmt.Errorf("zstd not supported without cgo")
}
