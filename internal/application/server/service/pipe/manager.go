package pipe

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"

	"github.com/pipehub/pipehub/internal"
)

// Manager is the responsible to initialize the pipes.
type Manager struct {
	pipes     []internal.Pipe
	instances map[string]instance
}

// Close the initialized pipes.
func (m *Manager) Close(ctx context.Context) error {
	var errs []string
	for _, instance := range m.instances {
		if err := instance.Close(ctx); err != nil {
			errs = append(errs, err.Error())
		}
	}

	switch len(errs) {
	case 0:
		return nil
	case 1:
		err := errors.New(errs[0])
		return errors.Wrap(err, "close error")
	default:
		value := strings.Join(errs, "|")
		return fmt.Errorf("multiple errors detected during close: (%s)", value)
	}
}

// Fetch the instance of a pipe.
// nolint: golint
func (m Manager) Fetch(importPathAlias string) (instance, error) {
	instance, ok := m.instances[importPathAlias]
	if !ok {
		return instance, errors.New("instance not found")
	}
	return instance, nil
}

func (m *Manager) init() error {
	add := reflect.ValueOf(m).MethodByName("InitPipes")
	if !add.IsValid() {
		return errors.New("missing 'InitPipes' method")
	}

	value := add.Call(nil)[0]
	if value.IsNil() {
		return nil
	}
	err := value.Interface().(error)
	return errors.Wrap(err, "init pipes error")
}

// nolint: unused
func (m Manager) config(importPath, id string) map[string]interface{} {
	for _, pipe := range m.pipes {
		if (pipe.Module != "") && (pipe.ImportPath == importPath) && (pipe.Module == id) {
			return pipe.Config
		}

		if (pipe.ImportPath == importPath) && (pipe.Version == id) {
			return pipe.Config
		}
	}

	return nil
}

// NewManager start the pipes.
func NewManager(pipes []internal.Pipe) (Manager, error) {
	m := Manager{
		pipes:     pipes,
		instances: make(map[string]instance),
	}

	if err := m.init(); err != nil {
		return m, errors.Wrap(err, "initialization error")
	}

	return m, nil
}
