package httpway

import (
	"context"

	"github.com/pkg/errors"
)

// ClientConfig holds the client configuration.
type ClientConfig struct {
	HTTP            ClientConfigHTTP
	AsyncErrHandler func(error)
}

// ClientConfigHTTP holds the http server configuration.
type ClientConfigHTTP struct {
	Port int
}

// Client is httpway entrypoint.
type Client struct {
	cfg    ClientConfig
	server *server
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
	return nil
}

func (c *Client) init(cfg ClientConfig) {
	c.cfg = cfg
	if c.cfg.AsyncErrHandler == nil {
		c.cfg.AsyncErrHandler = func(error) {}
	}
	c.server = newServer(c)
}

// NewClient return a configured httpway client.
func NewClient(cfg ClientConfig) Client {
	var c Client
	c.init(cfg)
	return c
}
