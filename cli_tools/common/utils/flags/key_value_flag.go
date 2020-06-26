package flags

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
)

// KeyValueString is an implementation of flag.Value that creates a map
// from the user's argument prior to storing it.
type KeyValueString map[string]string

// String returns string representation of KeyValueString.
// The format of the return value is "KEY1=AB,KEY2=CD"
func (s KeyValueString) String() string {
	var parts []string
	for k, v := range s {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ",")
}

// Set creates a key-value map of the input string.
// The input string must be in the format of KEY1=AB,KEY2=CD
func (s *KeyValueString) Set(input string) error {
	if *s != nil {
		return fmt.Errorf("only one instance of this flag is allowed")
	}

	*s = make(map[string]string)
	if input != "" {
		var err error
		*s, err = param.ParseKeyValues(input)
		if err != nil {
			return err
		}
	}
	return nil
}
