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
	action   struct {
		notFound http.HandlerFunc
		panik    func(http.Handler) http.Handler
	}
}

func (h *handlerManager) init() error {
	if err := h.initAction(); err != nil {
		return errors.Wrap(err, "init action error")
	}

	if err := h.initHandlers(); err != nil {
		return errors.Wrap(err, "init handlers error")
	}

	return nil
}

// essa função tem que ser genérica o suficiente pra podermos usar pro not found e pro panic.
func (h *handlerManager) initAction() error {
	notFoundAction, err := h.initFunc(h.client.cfg.Server.Action.NotFound)
	if err != nil {
		return errors.Wrap(err, "not found action initialization error")
	}
	h.action.notFound = notFoundAction

	panik, _, err := h.initMiddleware(h.client.cfg.Server.Action.Panic)
	if err != nil {
		return errors.Wrap(err, "panic initialization error")
	}
	h.action.panik = panik
	return nil
}

func (h *handlerManager) initHandlers() error {
	for _, host := range h.client.cfg.Host {
		middleware, rawHandler, err := h.initMiddleware(host.Handler)
		if err != nil {
			return errors.Wrap(err, "middleware initialization error")
		}

		hh := handler{
			name:     host.Handler,
			endpoint: host.Endpoint,
			instance: rawHandler.instance,
			alias:    rawHandler.alias,
			fn:       middleware,
		}
		h.handlers = append(h.handlers, hh)
	}

	return nil
}

func (h handlerManager) initGeneric(id string) (interface{}, handler, error) {
	if id == "" {
		return nil, handler{}, nil
	}

	handlers, err := fetchHandlers()
	if err != nil {
		return nil, handler{}, errors.Wrap(err, "fetch handlers error")
	}

	for _, rawHandler := range handlers {
		if !strings.HasPrefix(id, rawHandler.alias+".") {
			continue
		}

		fragments := strings.Split(id, ".")
		if len(fragments) != 2 {
			return nil, handler{}, fmt.Errorf("invalid func '%s'", id)
		}

		value := reflect.ValueOf(rawHandler.instance).MethodByName(fragments[1])
		if !value.IsValid() {
			continue
		}

		rawFn := value.Interface()
		return rawFn, rawHandler, nil
	}

	return nil, handler{}, nil
}

func (h handlerManager) initFunc(id string) (http.HandlerFunc, error) {
	rawFn, _, err := h.initGeneric(id)
	if err != nil {
		return nil, err
	}
	if rawFn == nil {
		return nil, nil
	}

	fn, ok := rawFn.(func(http.ResponseWriter, *http.Request))
	if !ok {
		return nil, fmt.Errorf("invalid func '%s'", id)
	}
	return fn, nil
}

func (h handlerManager) initMiddleware(id string) (func(http.Handler) http.Handler, handler, error) {
	rawFn, rawHandler, err := h.initGeneric(id)
	if err != nil {
		return nil, handler{}, err
	}
	if rawFn == nil {
		return nil, handler{}, nil
	}

	fn, ok := rawFn.(func(http.Handler) http.Handler)
	if !ok {
		return nil, handler{}, fmt.Errorf("invalid func '%s'", id)
	}
	return fn, rawHandler, nil
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
	h := handlerManager{client: c}
	if err := h.init(); err != nil {
		return nil, errors.Wrap(err, "initialization error")
	}
	return &h, nil
}
