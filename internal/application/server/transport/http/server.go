package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/go-chi/chi"
	"github.com/go-chi/hostrouter"
	"github.com/pkg/errors"

	"github.com/pipehub/pipehub/internal"
)

type serverHandlerFetcher interface {
	Middleware(id string) (func(http.Handler) http.Handler, error)
	Handler(id string) (func(http.ResponseWriter, *http.Request), error)
}

// ServerConfig has all the configuration needed to start a server.
type ServerConfig struct {
	// At the HTTP server a error can occur in a async manner. This function is used track this kind
	// of error and allow actions to be taken.
	AsyncErrorHandler func(error)

	Port           int
	Host           []internal.Host
	DefaultAction  ServerConfigDefaultAction
	HandlerFetcher serverHandlerFetcher
	RoundTripper   http.RoundTripper
}

// ServerConfigDefaultAction has the configuration needed to set the default actions at the server.
type ServerConfigDefaultAction struct {
	Panic    string
	NotFound string
}

// Server expose a HTTP server.
type Server struct {
	config ServerConfig
	base   *http.Server
}

// Start the server.
func (s *Server) Start() error {
	// Initialize the mux with its default handlers.
	mux := chi.NewRouter()

	if err := s.initHandlerNotFound(mux); err != nil {
		return errors.Wrap(err, "not found middleware initialization error")
	}

	if err := s.initHandlerPanic(mux); err != nil {
		return errors.Wrap(err, "panic middleware initialization error")
	}

	// Initialize all needed logic to direct the traffic to a pipe.
	pipeMux, err := s.genPipeMux()
	if err != nil {
		return errors.Wrap(err, "pipe mux initialization error")
	}
	s.initPipeMux(mux, pipeMux)

	// At this step, the mux is ready to receive requests.
	s.base = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: mux,
	}

	// Note that we're using the async error handler to catch any kind of listen errors. This is
	// needed because the listen call blocks, to avoid this issue, the whole call is inside a
	// goroutine. The async error handler is the only way to expose the error a listen may have.
	go func() {
		if err := s.base.ListenAndServe(); err != http.ErrServerClosed {
			err = errors.Wrapf(err, "server listen error at addr '%s'", s.base.Addr)
			s.config.AsyncErrorHandler(err)
		}
	}()

	return nil
}

// Stop the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.base.Shutdown(ctx)
}

func (s *Server) init() error {
	if s.config.AsyncErrorHandler == nil {
		return errors.New("missing 'AsyncErrorHandler'")
	}

	return nil
}

func (s *Server) initHandlerNotFound(mux *chi.Mux) error {
	fnName := s.config.DefaultAction.NotFound
	if fnName == "" {
		return nil
	}

	fn, err := s.config.HandlerFetcher.Handler(fnName)
	if err != nil {
		return errors.Wrapf(err, "fetch handler '%s' error", fnName)
	}
	mux.NotFound(fn)
	return nil
}

func (s *Server) initHandlerPanic(mux *chi.Mux) error {
	fnName := s.config.DefaultAction.Panic
	if fnName == "" {
		return nil
	}

	fn, err := s.config.HandlerFetcher.Middleware(fnName)
	if err != nil {
		return errors.Wrapf(err, "fetch middleware '%s' error", fnName)
	}
	mux.Use(fn)
	return nil
}

func (s *Server) genPipeMux() (map[string]*chi.Mux, error) {
	pipes := make(map[string]*chi.Mux)
	for _, host := range s.config.Host {
		proxy, err := s.initProxy(host.Handler)
		if err != nil {
			return nil, errors.Wrapf(err, "init proxy error for handler '%s'", host.Endpoint)
		}
		pipes[host.Endpoint] = proxy
	}
	return pipes, nil
}

func (Server) initPipeMux(mux *chi.Mux, pipeMux map[string]*chi.Mux) {
	router := hostrouter.New()
	for endpoint, pmux := range pipeMux {
		router.Map(endpoint, pmux)
	}
	mux.Mount("/", router)
}

func (s *Server) initProxy(handlerID string) (*chi.Mux, error) {
	director := func(req *http.Request) {
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	}
	proxy := &httputil.ReverseProxy{
		Director:  director,
		Transport: s.config.RoundTripper,
	}
	proxyHandler := http.HandlerFunc(proxy.ServeHTTP)

	pipeHandler, err := s.config.HandlerFetcher.Middleware(handlerID)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch handler '%s' error", handlerID)
	}

	mux := chi.NewRouter()
	if err := s.initHandlerPanic(mux); err != nil {
		return nil, errors.Wrap(err, "init panic handler error")
	}
	mux.Use(pipeHandler)
	mux.Mount("/", proxyHandler)

	return mux, nil
}

// NewServer return a configured server.
// nolint: gocritic
func NewServer(config ServerConfig) (Server, error) {
	s := Server{
		config: config,
	}
	if err := s.init(); err != nil {
		return s, errors.Wrap(err, "initialization error")
	}
	return s, nil
}
