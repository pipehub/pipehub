package httpway

import (
	"context"
)

type handlerInstance interface {
	Close(ctx context.Context) error
}

type handler struct {
	instance handlerInstance
}
