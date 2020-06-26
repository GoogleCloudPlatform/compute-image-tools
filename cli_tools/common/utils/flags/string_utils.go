package flags

import "strings"

// TrimmedString is an implementation of flag.Value that trims whitespace
// from the incoming argument prior to storing it.
type TrimmedString string

// String returns string representation of TrimmedString.
func (s TrimmedString) String() string { return (string)(s) }

// Set trims whitespace from input string and stores in TrimmedString.
func (s *TrimmedString) Set(input string) error {
	*s = TrimmedString(strings.TrimSpace(input))
	return nil
}

// LowerTrimmedString is an implementation of flag.Value that trims whitespace
// and converts to lowercase the incoming argument prior to storing it.
type LowerTrimmedString string

// String returns string representation of LowerTrimmedString.
func (s LowerTrimmedString) String() string { return (string)(s) }

// Set trims whitespace from input string and converts the string to lowercase,
// then stores in LowerTrimmedString.
func (s *LowerTrimmedString) Set(input string) error {
	*s = LowerTrimmedString(strings.ToLower(strings.TrimSpace(input)))
	return nil
}
