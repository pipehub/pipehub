package pipehub

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-chi/chi"
	"github.com/go-chi/hostrouter"
	"github.com/pkg/errors"
)

type server struct {
	client *Client
	base   *http.Server
}

func (s *server) init() {
	s.base = &http.Server{
		Addr: fmt.Sprintf(":%d", s.client.cfg.Server.HTTP.Port),
	}
}

func (s *server) start() error {
	r := chi.NewRouter()

	if s.client.handlerManager.action.notFound != nil {
		r.NotFound(s.client.handlerManager.action.notFound)
	}

	if s.client.handlerManager.action.panik != nil {
		r.Use(s.client.handlerManager.action.panik)
	}

	handlers, err := s.startHandlers()
	if err != nil {
		return errors.Wrap(err, "start handlers error")
	}

	hr := hostrouter.New()
	for endpoint, handler := range handlers {
		hr.Map(endpoint, handler)
	}
	r.Mount("/", hr)
	s.base.Handler = r

	go func() {
		if err := s.base.ListenAndServe(); err != nil {
			err = errors.Wrapf(err, "http server listen error at addr '%s'", s.base.Addr)
			s.client.cfg.AsyncErrHandler(err)
		}
	}()
	return nil
}

func (s *server) startHandlers() (map[string]*chi.Mux, error) {
	handlers := make(map[string]*chi.Mux)
	for _, host := range s.client.cfg.Host {
		h, err := s.client.handlerManager.fetch(host.Handler)
		if err != nil {
			return handlers, errors.Wrap(err, "fetch handler error")
		}

		if !h.valid() {
			continue
		}

		origin, err := url.Parse(host.Origin)
		if err != nil {
			return handlers, errors.Wrapf(err, "parse url '%s' error", host.Origin)
		}

		director := func(req *http.Request) {
			req.Host = origin.Host
			req.URL.Host = origin.Host
			req.URL.Scheme = origin.Scheme
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		}
		proxy := &httputil.ReverseProxy{Director: director}
		r := chi.NewRouter()
		if s.client.handlerManager.action.panik != nil {
			r.Use(s.client.handlerManager.action.panik)
		}
		r.Use(h.fn)
		r.Mount("/", http.HandlerFunc(proxy.ServeHTTP))
		handlers[host.Endpoint] = r
	}
	return handlers, nil
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
