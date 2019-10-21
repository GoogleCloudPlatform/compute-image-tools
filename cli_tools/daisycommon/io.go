package daisycommon

import "io"

// ByteCountingReader forwards calls to a delegate reader, keeping
// track of the number bytes that have been read in `BytesRead`.
// Errors are propagated unchanged.
type ByteCountingReader struct {
	r         io.Reader
	BytesRead int64
}

// NewByteCountingReader is a contructor for ByteCountingReader.
func NewByteCountingReader(r io.Reader) *ByteCountingReader {
	return &ByteCountingReader{r, 0}
}

func (l *ByteCountingReader) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	l.BytesRead += int64(n)
	return n, err
}
