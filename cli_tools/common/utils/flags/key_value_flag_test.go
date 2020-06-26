package flags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyValueSetReturnsMap(t *testing.T) {
	input := "KEY1=AB,KEY2=CD"
	expected := map[string]string{"KEY1": "AB", "KEY2": "CD"}
	var s KeyValueString = nil
	err := s.Set(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, map[string]string(s))
}

func TestKeyValueSetReturnsEmptyMap(t *testing.T) {
	input := ""
	expected := map[string]string{}
	var s KeyValueString = nil
	err := s.Set(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, map[string]string(s))
}

func TestKeyValueSetReturnsErrorIfNotNil(t *testing.T) {
	input := "KEY1=AB,KEY2=CD"
	var s KeyValueString = map[string]string{}
	err := s.Set(input)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "only one instance of this flag is allowed")
}

func TestKeyValueSetReturnsErrorIfWrongStringFormat(t *testing.T) {
	input := "KEY1->AB,KEY2->CD"
	var s KeyValueString = nil
	err := s.Set(input)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to parse key-value pair. "+
		"key-value should be in the following format: KEY=VALUE")
}

func TestKeyValueStringReturnsFormattedString(t *testing.T) {
	var s KeyValueString = map[string]string{"KEY1": "AB", "KEY2": "CD"}
	output := s.String()
	expected := "KEY1=AB,KEY2=CD"
	assert.Equal(t, expected, output)
}
