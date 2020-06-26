package flags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const expected = "test string"

func TestTrimmedStringSetTrimsSpace(t *testing.T) {
	input := "     test string        "
	var s = TrimmedString("")
	err := s.Set(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, string(s))
}

func TestTrimmedStringStringReturnsString(t *testing.T) {
	var s = TrimmedString(expected)
	output := s.String()
	assert.Equal(t, expected, output)
}

func TestLowerTrimmedStringSetTrimsSpace(t *testing.T) {
	input := "     TeSt StRiNg        "
	var s = LowerTrimmedString("")
	err := s.Set(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, string(s))
}

func TestLowerTrimmedStringStringReturnsString(t *testing.T) {
	var s = LowerTrimmedString(expected)
	output := s.String()
	assert.Equal(t, expected, output)
}
