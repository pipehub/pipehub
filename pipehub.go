package pipehub

import (
	"context"
	"net/http"
)

type pipeInstance interface {
	Close(ctx context.Context) error
}

type pipe struct {
	name     string
	endpoint string
	instance pipeInstance
	alias    string
	fn       func(http.Handler) http.Handler
}

func (p pipe) valid() bool {
	return p.instance != nil
}
