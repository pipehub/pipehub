package server

import (
	"context"

	"github.com/pkg/errors"

	"github.com/pipehub/pipehub/internal"
	"github.com/pipehub/pipehub/internal/application/server/service/pipe"
	transportHTTP "github.com/pipehub/pipehub/internal/application/server/transport/http"
)

// ClientConfig is used to initialize the server.
type ClientConfig struct {
	Pipe      []internal.Pipe
	Service   ClientConfigService
	Transport ClientConfigTransport
}

type ClientConfigService struct {
	Pipe ClientConfigServicePipe
}

type ClientConfigServicePipe struct {
	HTTP pipe.HTTPConfig
}

type ClientConfigTransport struct {
	HTTP transportHTTP.ServerConfig
}

// Client is the core struct that initialize the PipeHub server.
type Client struct {
	service struct {
		manager pipe.Manager
		http    pipe.HTTP
	}

	transport struct {
		http transportHTTP.Server
	}

	config ClientConfig
}

// Start the server.
func (c *Client) Start() error {
	var err error
	c.service.manager, err = pipe.NewManager(c.config.Pipe)
	if err != nil {
		return errors.Wrap(err, "manager service initialization error")
	}

	c.config.Service.Pipe.HTTP.Instance = c.service.manager
	c.service.http, err = pipe.NewHTTP(c.config.Service.Pipe.HTTP)
	if err != nil {
		return errors.Wrap(err, "http service initialization error")
	}

	c.config.Transport.HTTP.HandlerFetcher = &c.service.http
	c.transport.http, err = transportHTTP.NewServer(c.config.Transport.HTTP)
	if err != nil {
		return errors.Wrap(err, "transport http initialization error")
	}

	if err := c.transport.http.Start(); err != nil {
		return errors.Wrap(err, "http transport start error")
	}

	return nil
}

// Stop the server.
func (c *Client) Stop(ctx context.Context) error {
	if err := c.transport.http.Stop(ctx); err != nil {
		return errors.Wrap(err, "transport http stop error")
	}

	if err := c.service.manager.Close(ctx); err != nil {
		return errors.Wrap(err, "manager service stop error")
	}

	return nil
}

// NewClient initialize the client.
// nolint: gocritic
func NewClient(config ClientConfig) Client {
	return Client{config: config}
}
