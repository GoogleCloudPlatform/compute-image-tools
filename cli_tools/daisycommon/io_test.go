package daisycommon

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByteCountingReader(t *testing.T) {
	data := []byte{100, 101, 102, 103, 104, 105}
	reader := NewByteCountingReader(bytes.NewReader(data))
	buff := make([]byte, 2)

	// Read one byte at a time, checking that our accounting of
	// BytesRead is correct and that we forward the correct
	// byte to the caller
	for i := 0; i < 6; i += 2 {
		assert.Equal(t, int64(i), reader.BytesRead)
		n, err := reader.Read(buff)
		assert.Equal(t, data[i], buff[0])
		assert.Equal(t, data[i+1], buff[1])
		assert.Equal(t, 2, n)
		assert.Nil(t, err)
	}

	// Once exhausted, the facade should forward the error from the underlying
	// reader, and it should not change increment BytesRead.
	assert.Equal(t, int64(6), reader.BytesRead)
	n, err := reader.Read(buff)
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, int64(6), reader.BytesRead)
}
