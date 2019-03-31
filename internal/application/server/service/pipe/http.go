package pipe

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/pkg/errors"
)

type httpInstance interface {
	Fetch(importPathAlias string) (instance, error)
}

// HTTPConfig holds the configuration to direct the request through pipes.
type HTTPConfig struct {
	Entry         []HTTPConfigEntry
	DefaultAction HTTPConfigDefaultAction
	Instance      httpInstance
}

// HTTPConfigDefaultAction set the HTTP default actions.
type HTTPConfigDefaultAction struct {
	NotFound string
	Panic    string
}

// HTTPConfigEntry set the entries PipeHub gonna proxy.
type HTTPConfigEntry struct {
	Endpoint string
	Handler  string
}

// HTTP is used to extract all the missing information from the HTTP transport.
type HTTP struct {
	config    HTTPConfig
	instances map[string]instance
}

// Middleware returns a middleware entry.
func (h *HTTP) Middleware(id string) (func(http.Handler) http.Handler, error) {
	rawFn, err := h.extractFn(id)
	if err != nil {
		return nil, err
	}

	fn, ok := rawFn.(func(http.Handler) http.Handler)
	if !ok {
		return nil, errors.New("could not cast the function into 'func(http.Handler) http.Handler'")
	}

	return fn, nil
}

// Handler return a handler entry.
func (h *HTTP) Handler(id string) (func(http.ResponseWriter, *http.Request), error) {
	rawFn, err := h.extractFn(id)
	if err != nil {
		return nil, err
	}

	fn, ok := rawFn.(func(http.ResponseWriter, *http.Request))
	if !ok {
		return nil, errors.New("could not cast the function into 'func(http.Handler) http.Handler'")
	}

	return fn, nil
}

// init fetch all the pipe instances using the path import alias.
func (h *HTTP) init() error {
	for _, entry := range h.config.Entry {
		importPathAlias, _, err := extractPipeHandler(entry.Handler)
		if err != nil {
			return errors.Wrap(err, "could not extract import path alias and function entry")
		}

		instance, err := h.config.Instance.Fetch(importPathAlias)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch instance of '%s'", importPathAlias)
		}
		h.instances[importPathAlias] = instance
	}
	return nil
}

func (h *HTTP) extractFn(id string) (interface{}, error) {
	importPathAlias, fnName, err := extractPipeHandler(id)
	if err != nil {
		return nil, errors.Wrap(err, "could not extract import path alias and function entry")
	}

	instance, ok := h.instances[importPathAlias]
	if !ok {
		return nil, fmt.Errorf("no instance is associated with '%s'", importPathAlias)
	}

	rawFn := reflect.ValueOf(instance.instancer).MethodByName(fnName)
	if !rawFn.IsValid() {
		return nil, fmt.Errorf("invalid method '%s'", fnName)
	}

	return rawFn.Interface(), nil
}

// NewHTTP return a configured HTTP struct.
func NewHTTP(config HTTPConfig) (HTTP, error) {
	h := HTTP{
		config:    config,
		instances: make(map[string]instance),
	}
	if err := h.init(); err != nil {
		return h, errors.Wrap(err, "initialization error")
	}
	return h, nil
}
