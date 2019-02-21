// +build public

package ospackage

import (
	"context"
)

// RunOSConfig does nothing.
func RunOsConfig(_ context.Context, _ string, _ bool) error {
	return nil
}
