package httpway

import (
	"context"

	"github.com/pkg/errors"
)

type handlerManager struct {
	handlers []handler
}

func (h *handlerManager) close(ctx context.Context) error {
	for _, handler := range h.handlers {
		if err := handler.instance.Close(ctx); err != nil {
			return errors.Wrap(err, "error closing handler")
		}
	}
	return nil
}

func newHandlerManager(c *Client) *handlerManager {
	return &handlerManager{
		handlers: fetchHandlers(),
	}
}
