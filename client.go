package pipehub

import (
	"context"

	"github.com/pkg/errors"
)

// ClientConfig holds the client configuration.
type ClientConfig struct {
	Core            ClientConfigCore
	HTTP            []ClientConfigHTTP
	Pipe            []ClientConfigPipe
	AsyncErrHandler func(error)
}

func (c *ClientConfig) setDefaultValues() {
	c.Core.setDefaultValues()
	if c.AsyncErrHandler == nil {
		c.AsyncErrHandler = func(error) {}
	}
}

// ClientConfigCore holds the core configuration.
type ClientConfigCore struct {
	HTTP ClientConfigCoreHTTP
}

func (c *ClientConfigCore) setDefaultValues() {
	c.HTTP.setDefaultValues()
}

// ClientConfigCoreHTTP holds the core HTTP configuration.
type ClientConfigCoreHTTP struct {
	Server ClientConfigCoreHTTPServer
}

func (c *ClientConfigCoreHTTP) setDefaultValues() {
	c.Server.setDefaultValues()
}

// ClientConfigCoreHTTPServer holds the core HTTP server configurations.
type ClientConfigCoreHTTPServer struct {
	Action ClientConfigCoreHTTPServerAction
	Listen ClientConfigCoreHTTPServerListen
}

func (c *ClientConfigCoreHTTPServer) setDefaultValues() {
	c.Listen.setDefaultValues()
}

// ClientConfigCoreHTTPServerAction holds the action server configuration.
type ClientConfigCoreHTTPServerAction struct {
	NotFound string
	Panic    string
}

// ClientConfigCoreHTTPServerListen holds the configuration to start a HTTP listener.
type ClientConfigCoreHTTPServerListen struct {
	Port int
}

func (c *ClientConfigCoreHTTPServerListen) setDefaultValues() {
	if c.Port == 0 {
		c.Port = 80
	}
}

// ClientConfigHTTP holds the configuration to direct the request through pipes.
type ClientConfigHTTP struct {
	Endpoint string
	Handler  string
}

// ClientConfigPipe holds the pipe configuration.
type ClientConfigPipe struct {
	Path    string
	Module  string
	Version string
	Config  map[string]interface{}
}

// Client is pipehub entrypoint.
type Client struct {
	cfg         ClientConfig
	server      *server
	pipeManager *pipeManager
}

// Start pipehub.
func (c *Client) Start() error {
	if err := c.server.start(); err != nil {
		return errors.Wrap(err, "server start error")
	}
	return nil
}

// Stop the pipehub.
func (c *Client) Stop(ctx context.Context) error {
	if err := c.server.stop(ctx); err != nil {
		return errors.Wrap(err, "server stop error")
	}

	if err := c.pipeManager.close(ctx); err != nil {
		return errors.Wrap(err, "pipe manager close error")
	}
	return nil
}

// nolint: gocritic
func (c *Client) init(cfg ClientConfig) error {
	cfg.setDefaultValues()
	c.cfg = cfg
	if c.cfg.AsyncErrHandler == nil {
		c.cfg.AsyncErrHandler = func(error) {}
	}

	var err error
	c.pipeManager, err = newPipeManager(c)
	if err != nil {
		return errors.Wrap(err, "pipe manager new instance error")
	}
	c.server = newServer(c)
	return nil
}

// NewClient return a configured pipehub client.
// nolint: gocritic
func NewClient(cfg ClientConfig) (Client, error) {
	var c Client
	if err := c.init(cfg); err != nil {
		return c, errors.Wrap(err, "client new instance error")
	}
	return c, nil
}
