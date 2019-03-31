package pipe

import (
	"context"
	"fmt"
	"strings"
)

// nolint: unused, structcheck
type instance struct {
	// If the pipe has a module, then the id should be it's module, otherwise, the version.
	id string

	importPath string
	instancer
}

type instancer interface {
	Close(ctx context.Context) error
}

// extractPipeHandler return the pipe import path alias and the function.
// Example: base.handler, will return 'base' as the import path alias and 'handler' as the function.
func extractPipeHandler(id string) (importPathAlias, handler string, err error) {
	fragments := strings.Split(id, ".")
	if len(fragments) != 2 {
		return "", "", fmt.Errorf("invalid handler '%s'", id)
	}
	return fragments[0], fragments[1], nil
}
