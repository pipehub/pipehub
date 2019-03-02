package httpway

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"
)

type server struct {
	client *Client
	base   *http.Server
}

func (s *server) init() {
	s.base = &http.Server{
		Addr: fmt.Sprintf(":%d", s.client.cfg.HTTP.Port),
	}
}

func (s *server) start() error {
	r := chi.NewRouter()
	s.base.Handler = r

	go func() {
		if err := s.base.ListenAndServe(); err != nil {
			err = errors.Wrapf(err, "http server listen error at addr '%s'", s.base.Addr)
			s.client.cfg.AsyncErrHandler(err)
		}
	}()

	return nil
}

func (s *server) stop(ctx context.Context) error {
	err := s.base.Shutdown(ctx)
	return errors.Wrap(err, "base http server shutdown error")
}

func newServer(c *Client) *server {
	s := server{client: c}
	s.init()
	return &s
}
