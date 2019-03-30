package pipehub

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/hostrouter"
	"github.com/pkg/errors"
)

type server struct {
	client    *Client
	transport *http.Transport
	base      *http.Server
}

func (s *server) init() {
	s.base = &http.Server{
		Addr: fmt.Sprintf(":%d", s.client.cfg.Core.HTTP.Server.Listen.Port),
	}

	transportConfig := s.client.cfg.Core.HTTP.Client
	s.transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          transportConfig.MaxIdleConns,
		MaxConnsPerHost:       transportConfig.MaxConnsPerHost,
		MaxIdleConnsPerHost:   transportConfig.MaxIdleConnsPerHost,
		IdleConnTimeout:       transportConfig.IdleConnTimeout,
		TLSHandshakeTimeout:   transportConfig.TLSHandshakeTimeout,
		ExpectContinueTimeout: transportConfig.ExpectContinueTimeout,
	}
}

func (s *server) start() error {
	r := chi.NewRouter()

	if s.client.pipeManager.action.notFound != nil {
		r.NotFound(s.client.pipeManager.action.notFound)
	}

	if s.client.pipeManager.action.panik != nil {
		r.Use(s.client.pipeManager.action.panik)
	}

	pipes, err := s.startPipes()
	if err != nil {
		return errors.Wrap(err, "start handlers error")
	}

	hr := hostrouter.New()
	for endpoint, pipe := range pipes {
		hr.Map(endpoint, pipe)
	}
	r.Mount("/", hr)
	s.base.Handler = r

	go func() {
		if err := s.base.ListenAndServe(); err != http.ErrServerClosed {
			err = errors.Wrapf(err, "http server listen error at addr '%s'", s.base.Addr)
			s.client.cfg.AsyncErrHandler(err)
		}
	}()
	return nil
}

func (s *server) startPipes() (map[string]*chi.Mux, error) {
	pipes := make(map[string]*chi.Mux)
	for _, configHTTP := range s.client.cfg.HTTP {
		p, err := s.client.pipeManager.fetch(configHTTP.Handler)
		if err != nil {
			return pipes, errors.Wrap(err, "fetch pipe error")
		}

		if !p.valid() {
			continue
		}

		director := func(req *http.Request) {
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		}
		proxy := &httputil.ReverseProxy{
			Director:  director,
			Transport: s.transport,
		}
		r := chi.NewRouter()
		if s.client.pipeManager.action.panik != nil {
			r.Use(s.client.pipeManager.action.panik)
		}
		r.Use(p.fn)
		r.Mount("/", http.HandlerFunc(proxy.ServeHTTP))
		pipes[configHTTP.Endpoint] = r
	}
	return pipes, nil
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
