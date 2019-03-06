package httpway

import (
	"context"

	"github.com/pkg/errors"
)

// ClientConfig holds the client configuration.
type ClientConfig struct {
	HTTP            ClientConfigHTTP
	Host            []ClientConfigHost
	AsyncErrHandler func(error)
}

func (c *ClientConfig) setDefaultValues() {
	c.HTTP.setDefaultValues()
	if c.AsyncErrHandler == nil {
		c.AsyncErrHandler = func(error) {}
	}
}

// ClientConfigHTTP holds the http server configuration.
type ClientConfigHTTP struct {
	Port int
}

func (c *ClientConfigHTTP) setDefaultValues() {
	if c.Port == 0 {
		c.Port = 80
	}
}

// ClientConfigHost holds the configuration to direct the request from hosts to handlers.
type ClientConfigHost struct {
	Endpoint string
	Origin   string
	Handler  string
}

// Client is httpway entrypoint.
type Client struct {
	cfg            ClientConfig
	server         *server
	handlerManager *handlerManager
}

// Start httpway.
func (c *Client) Start() error {
	if err := c.server.start(); err != nil {
		return errors.Wrap(err, "server start error")
	}
	return nil
}

// Stop the httpway.
func (c *Client) Stop(ctx context.Context) error {
	if err := c.server.stop(ctx); err != nil {
		return errors.Wrap(err, "server stop error")
	}

	if err := c.handlerManager.close(ctx); err != nil {
		return errors.Wrap(err, "handler manager close error")
	}
	return nil
}

func (c *Client) init(cfg ClientConfig) error {
	cfg.setDefaultValues()
	c.cfg = cfg
	if c.cfg.AsyncErrHandler == nil {
		c.cfg.AsyncErrHandler = func(error) {}
	}

	var err error
	c.handlerManager, err = newHandlerManager(c)
	if err != nil {
		return errors.Wrap(err, "handler manager new instance error")
	}
	c.server = newServer(c)
	return nil
}

// NewClient return a configured httpway client.
func NewClient(cfg ClientConfig) (Client, error) {
	var c Client
	if err := c.init(cfg); err != nil {
		return c, errors.Wrap(err, "client new instance error")
	}
	return c, nil
}
