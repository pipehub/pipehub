package httpway

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

type handlerManager struct {
	client   *Client
	handlers []handler
}

func (h *handlerManager) init() error {
	handlers, err := fetchHandlers()
	if err != nil {
		return errors.Wrap(err, "fetch handlers error")
	}

	for _, host := range h.client.cfg.Host {
		for _, rawHandler := range handlers {
			if !strings.HasPrefix(host.Handler, rawHandler.alias+".") {
				continue
			}

			fragments := strings.Split(host.Handler, ".")
			if len(fragments) != 2 {
				return fmt.Errorf("invalid host handler '%s'", host.Handler)
			}

			rawFn := reflect.ValueOf(rawHandler.instance).MethodByName(fragments[1]).Interface()
			fn, ok := rawFn.(func(http.Handler) http.Handler)
			if !ok {
				return fmt.Errorf("invalid host handler function '%s'", host.Handler)
			}

			hh := handler{
				name:     host.Handler,
				endpoint: host.Endpoint,
				instance: rawHandler.instance,
				alias:    rawHandler.alias,
				fn:       fn,
			}
			h.handlers = append(h.handlers, hh)
		}
	}

	return nil
}

func (h handlerManager) fetch(name string) (handler, error) {
	for _, handler := range h.handlers {
		if handler.name == name {
			return handler, nil
		}
	}
	return handler{}, fmt.Errorf("handler '%s' not found", name)
}

func (h *handlerManager) close(ctx context.Context) error {
	for _, handler := range h.handlers {
		if err := handler.instance.Close(ctx); err != nil {
			return errors.Wrap(err, "error closing handler")
		}
	}
	return nil
}

func newHandlerManager(c *Client) (*handlerManager, error) {
	h := handlerManager{
		client: c,
	}

	if err := h.init(); err != nil {
		return nil, errors.Wrap(err, "initialization error")
	}
	return &h, nil
}
