package pipehub

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

type pipeManager struct {
	client *Client
	pipes  []pipe
	action struct {
		notFound http.HandlerFunc
		panik    func(http.Handler) http.Handler
	}
}

func (pm *pipeManager) init() error {
	if err := pm.initAction(); err != nil {
		return errors.Wrap(err, "init action error")
	}

	if err := pm.initPipes(); err != nil {
		return errors.Wrap(err, "init pipes error")
	}

	return nil
}

func (pm *pipeManager) initAction() error {
	notFoundAction, err := pm.initFunc(pm.client.cfg.Server.Action.NotFound)
	if err != nil {
		return errors.Wrap(err, "not found action initialization error")
	}
	pm.action.notFound = notFoundAction

	panik, _, err := pm.initMiddleware(pm.client.cfg.Server.Action.Panic)
	if err != nil {
		return errors.Wrap(err, "panic initialization error")
	}
	pm.action.panik = panik
	return nil
}

func (pm *pipeManager) initPipes() error {
	for _, host := range pm.client.cfg.Host {
		middleware, rawPipe, err := pm.initMiddleware(host.Handler)
		if err != nil {
			return errors.Wrap(err, "middleware initialization error")
		}

		p := pipe{
			name:     host.Handler,
			endpoint: host.Endpoint,
			instance: rawPipe.instance,
			alias:    rawPipe.alias,
			fn:       middleware,
		}
		pm.pipes = append(pm.pipes, p)
	}

	return nil
}

func (pm pipeManager) initGeneric(id string) (interface{}, pipe, error) {
	if id == "" {
		return nil, pipe{}, nil
	}

	pipes, err := pm.fetchPipes()
	if err != nil {
		return nil, pipe{}, errors.Wrap(err, "fetch pipes error")
	}

	for _, rawPipe := range pipes {
		if !strings.HasPrefix(id, rawPipe.alias+".") {
			continue
		}

		fragments := strings.Split(id, ".")
		if len(fragments) != 2 {
			return nil, pipe{}, fmt.Errorf("invalid func '%s'", id)
		}

		value := reflect.ValueOf(rawPipe.instance).MethodByName(fragments[1])
		if !value.IsValid() {
			continue
		}

		rawFn := value.Interface()
		return rawFn, rawPipe, nil
	}

	return nil, pipe{}, nil
}

func (pm pipeManager) initFunc(id string) (http.HandlerFunc, error) {
	rawFn, _, err := pm.initGeneric(id)
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

func (pm pipeManager) initMiddleware(id string) (func(http.Handler) http.Handler, pipe, error) {
	rawFn, rawHandler, err := pm.initGeneric(id)
	if err != nil {
		return nil, pipe{}, err
	}
	if rawFn == nil {
		return nil, pipe{}, nil
	}

	fn, ok := rawFn.(func(http.Handler) http.Handler)
	if !ok {
		return nil, pipe{}, fmt.Errorf("invalid func '%s'", id)
	}
	return fn, rawHandler, nil
}

func (pm pipeManager) fetch(name string) (pipe, error) {
	for _, pipe := range pm.pipes {
		if pipe.name == name {
			return pipe, nil
		}
	}
	return pipe{}, fmt.Errorf("pipe '%s' not found", name)
}

func (pm *pipeManager) close(ctx context.Context) error {
	for _, pipe := range pm.pipes {
		if err := pipe.instance.Close(ctx); err != nil {
			return errors.Wrap(err, "pipe closing error")
		}
	}
	return nil
}

func newPipeManager(c *Client) (*pipeManager, error) {
	pm := pipeManager{client: c}
	if err := pm.init(); err != nil {
		return nil, errors.Wrap(err, "initialization error")
	}
	return &pm, nil
}
