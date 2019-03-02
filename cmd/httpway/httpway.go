package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/hashicorp/hcl"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/httpway/httpway"
)

type config struct {
	Handler []configHandler `hcl:"handler" mapstructure:"handler"`
	Server  []configServer  `hcl:"server" mapstructure:"server"`
}

func (c config) server() configServer {
	if len(c.Server) == 0 {
		return configServer{
			GracefulShutdown: "30s",
			HTTP: []configServerHTTP{
				{
					Port: 80,
				},
			},
		}
	}
	return c.Server[0]
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
	s := c.server()
	return httpway.ClientConfig{
		HTTP: httpway.ClientConfigHTTP{
			Port: s.HTTP[0].Port,
		},
		AsyncErrHandler: asyncErrHandler,
	}
}

func (c config) ctxShutdown() (ctx context.Context, ctxCancel func()) {
	s := c.server()
	duration, err := time.ParseDuration(s.GracefulShutdown)
	if err != nil {
		err = errors.Wrapf(err, "parse duration '%s' error", s.GracefulShutdown)
		fatal(err)
	}
	return context.WithTimeout(context.Background(), duration)
}

type configHandler struct {
	Path    string `hcl:"path" mapstructure:"path"`
	Version string `hcl:"version" mapstructure:"version"`
	Alias   string `hcl:"alias" mapstructure:"alias"`
}

type configServer struct {
	GracefulShutdown string             `hcl:"graceful-shutdown" mapstructure:"graceful-shutdown"`
	HTTP             []configServerHTTP `hcl:"http" mapstructure:"http"`
}

type configServerHTTP struct {
	Port int `hcl:"port" mapstructure:"port"`
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
