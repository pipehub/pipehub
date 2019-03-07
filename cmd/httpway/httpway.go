package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/hcl"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/httpway/httpway"
)

type config struct {
	Host    []configHost    `hcl:"host" mapstructure:"host"`
	Handler []configHandler `hcl:"handler" mapstructure:"handler"`
	Server  []configServer  `hcl:"server" mapstructure:"server"`
}

func (c config) valid() error {
	if len(c.Server) > 1 {
		return errors.New("more then one 'server' config block found, only one is allowed")
	}

	for _, s := range c.Server {
		if err := s.valid(); err != nil {
			return err
		}
	}
	return nil
}

func (c config) toGenerateConfig() httpway.GenerateConfig {
	var cfg httpway.GenerateConfig
	for _, handler := range c.Handler {
		cfg.Handler = append(cfg.Handler, httpway.GenerateConfigHandler{
			Alias:   handler.Alias,
			Path:    handler.Path,
			Version: handler.Version,
		})
	}
	return cfg
}

func (c config) toClientConfig() httpway.ClientConfig {
	cfg := httpway.ClientConfig{
		AsyncErrHandler: asyncErrHandler,
		Host:            make([]httpway.ClientConfigHost, 0, len(c.Host)),
	}

	for _, host := range c.Host {
		cfg.Host = append(cfg.Host, httpway.ClientConfigHost{
			Endpoint: host.Endpoint,
			Origin:   host.Origin,
			Handler:  host.Handler,
		})
	}

	if len(c.Server) > 0 {
		if len(c.Server[0].Action) > 0 {
			cfg.Server.Action.NotFound = c.Server[0].Action[0].NotFound
			cfg.Server.Action.Panic = c.Server[0].Action[0].Panic
		}

		if len(c.Server[0].HTTP) > 0 {
			cfg.Server.HTTP = httpway.ClientConfigServerHTTP{
				Port: c.Server[0].HTTP[0].Port,
			}
		}
	}

	return cfg
}

func (c config) ctxShutdown() (ctx context.Context, ctxCancel func()) {
	if (len(c.Server) == 0) || (c.Server[0].GracefulShutdown == "") {
		return context.Background(), func() {}
	}

	raw := c.Server[0].GracefulShutdown
	duration, err := time.ParseDuration(raw)
	if err != nil {
		err = errors.Wrapf(err, "parse duration '%s' error", raw)
		fatal(err)
	}
	return context.WithTimeout(context.Background(), duration)
}

type configHandler struct {
	Path    string `hcl:"path" mapstructure:"path"`
	Version string `hcl:"version" mapstructure:"version"`
	Alias   string `hcl:"alias" mapstructure:"alias"`
}

type configHost struct {
	Endpoint string `hcl:"endpoint" mapstructure:"endpoint"`
	Origin   string `hcl:"origin" mapstructure:"origin"`
	Handler  string `hcl:"handler" mapstructure:"handler"`
}

type configServer struct {
	GracefulShutdown string               `hcl:"graceful-shutdown" mapstructure:"graceful-shutdown"`
	HTTP             []configServerHTTP   `hcl:"http" mapstructure:"http"`
	Action           []configServerAction `hcl:"action" mapstructure:"action"`
}

func (c configServer) valid() error {
	if len(c.HTTP) > 1 {
		return errors.New("more then one 'server.http' config block found, only one is allowed")
	}

	if len(c.Action) > 1 {
		return errors.New("more then one 'server.action' config block found, only one is allowed")
	}
	return nil
}

type configServerHTTP struct {
	Port int `hcl:"port" mapstructure:"port"`
}

type configServerAction struct {
	NotFound string `hcl:"not-found" mapstructure:"not-found"`
	Panic    string `hcl:"panic" mapstructure:"panic"`
}

func loadConfig(path string) (config, error) {
	payload, err := ioutil.ReadFile(path)
	if err != nil {
		return config{}, errors.Wrap(err, "load file error")
	}

	// For some reason I can't unmarshal direct from the HCL to a struct, the array values get messed up.
	// Unmarshalling to a map works fine, so we do this and later transform the map into the desired struct.
	rawCfg := make(map[string]interface{})
	if err = hcl.Unmarshal(payload, &rawCfg); err != nil {
		return config{}, errors.Wrap(err, "unmarshal payload error")
	}

	var cfg config
	if err := mapstructure.Decode(rawCfg, &cfg); err != nil {
		return config{}, errors.Wrap(err, "unmarshal error")
	}
	return cfg, nil
}

func fatal(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
}

func wait() {
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
}

func asyncErrHandler(err error) {
	fmt.Println(errors.Wrap(err, "async error occurred").Error())
	done <- syscall.SIGTERM
}
