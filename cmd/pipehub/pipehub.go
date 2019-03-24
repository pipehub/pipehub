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

	"github.com/pipehub/pipehub"
)

type config struct {
	HTTP   []configHTTP   `mapstructure:"host"`
	Pipe   []configPipe   `mapstructure:"pipe"`
	Server []configServer `mapstructure:"server"`
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

func (c config) toGenerateConfig() pipehub.GenerateConfig {
	var cfg pipehub.GenerateConfig
	for _, pipe := range c.Pipe {
		cfg.Pipe = append(cfg.Pipe, pipehub.GenerateConfigPipe{
			Alias:   pipe.Alias,
			Path:    pipe.Path,
			Module:  pipe.Module,
			Version: pipe.Version,
		})
	}
	return cfg
}

func (c config) toClientConfig() pipehub.ClientConfig {
	cfg := pipehub.ClientConfig{
		AsyncErrHandler: asyncErrHandler,
		HTTP:            make([]pipehub.ClientConfigHTTP, 0, len(c.HTTP)),
	}

	for _, http := range c.HTTP {
		cfg.HTTP = append(cfg.HTTP, pipehub.ClientConfigHTTP{
			Endpoint: http.Endpoint,
			Handler:  http.Handler,
		})
	}

	for _, pipe := range c.Pipe {
		cfg.Pipe = append(cfg.Pipe, pipehub.ClientConfigPipe{
			Path:    pipe.Path,
			Module:  pipe.Module,
			Version: pipe.Version,
			Config:  pipe.Config,
		})
	}

	if len(c.Server) > 0 {
		if len(c.Server[0].Action) > 0 {
			cfg.Server.Action.NotFound = c.Server[0].Action[0].NotFound
			cfg.Server.Action.Panic = c.Server[0].Action[0].Panic
		}

		if len(c.Server[0].HTTP) > 0 {
			cfg.Server.HTTP = pipehub.ClientConfigServerHTTP{
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

type configPipe struct {
	Path    string                 `mapstructure:"path"`
	Version string                 `mapstructure:"version"`
	Alias   string                 `mapstructure:"alias"`
	Module  string                 `mapstructure:"module"`
	Config  map[string]interface{} `mapstructure:"config"`
}

type configHTTP struct {
	Endpoint string `mapstructure:"endpoint"`
	Handler  string `mapstructure:"handler"`
}

type configServer struct {
	GracefulShutdown string               `mapstructure:"graceful-shutdown"`
	HTTP             []configServerHTTP   `mapstructure:"http"`
	Action           []configServerAction `mapstructure:"action"`
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
	Port int `mapstructure:"port"`
}

type configServerAction struct {
	NotFound string `mapstructure:"not-found"`
	Panic    string `mapstructure:"panic"`
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

	cfg.HTTP, err = loadConfigHTTP(rawCfg["http"])
	if err != nil {
		return config{}, errors.Wrap(err, "unmarshal http config error")
	}

	cfg.Pipe, err = loadConfigPipe(rawCfg["pipe"])
	if err != nil {
		return config{}, errors.Wrap(err, "unmarshal pipe config error")
	}

	return cfg, nil
}

// loadConfigPipe expect to receive a interface with this format:
//
//	[]map[string]interface {}{
//		{
//				"github.com/pipehub/pipe": []map[string]interface {}{
//						{
//								"version": "v0.7.0",
//								"alias":   "pipe",
//						},
//				},
//		},
//	}
func loadConfigPipe(raw interface{}) ([]configPipe, error) {
	var result []configPipe

	if raw == nil {
		return nil, nil
	}

	rawSliceMap, ok := raw.([]map[string]interface{})
	if !ok {
		return nil, errors.New("can't type assertion value into []map[string]interface{} on the first assignment")
	}

	for _, rawMap := range rawSliceMap {
		for key, rawMapEntry := range rawMap {
			rawSliceMapInner, ok := rawMapEntry.([]map[string]interface{})
			if !ok {
				return nil, errors.New("can't type assertion value into []map[string]interface{} on the second assignment")
			}

			for _, rawSliceMapInnerEntry := range rawSliceMapInner {
				ch := configPipe{
					Path: key,
				}

				for innerKey, innerEntry := range rawSliceMapInnerEntry {
					switch innerKey {
					case "version", "alias", "module":
						value, ok := innerEntry.(string)
						if !ok {
							return nil, errors.New("can't type assertion value into string")
						}

						switch innerKey {
						case "version":
							ch.Version = value
						case "alias":
							ch.Alias = value
						case "module":
							ch.Module = value
						}
					case "config":
						values, ok := innerEntry.([]map[string]interface{})
						if !ok {
							return nil, errors.New("can't type assertion value into map[string]interface{}")
						}
						if len(values) > 1 {
							return nil, errors.New("more then one 'config' found at a pipe, only one is allowed")
						}
						ch.Config = values[0]
					default:
						return nil, fmt.Errorf("unknow pipe key '%s'", innerKey)
					}
				}

				result = append(result, ch)
			}
		}
	}

	return result, nil
}

// loadConfigHTTP expect to receive a interface with this format:
//
// []map[string]interface {}{
// 	{
// 			"google": []map[string]interface {}{
// 					{
// 							"handler": "base.Default",
// 					},
// 			},
// 	},
// }
func loadConfigHTTP(raw interface{}) ([]configHTTP, error) {
	var result []configHTTP

	if raw == nil {
		return nil, nil
	}

	rawSliceMap, ok := raw.([]map[string]interface{})
	if !ok {
		return nil, errors.New("can't type assertion value into []map[string]interface{} on the first assignment")
	}

	for _, rawMap := range rawSliceMap {
		for key, rawMapEntry := range rawMap {
			rawSliceMapInner, ok := rawMapEntry.([]map[string]interface{})
			if !ok {
				return nil, errors.New("can't type assertion value into []map[string]interface{} on the second assignment")
			}

			for _, rawSliceMapInnerEntry := range rawSliceMapInner {
				ch := configHTTP{
					Endpoint: key,
				}

				for innerKey, innerEntry := range rawSliceMapInnerEntry {
					value, ok := innerEntry.(string)
					if !ok {
						return nil, errors.New("can't type assertion value into string")
					}

					switch innerKey {
					case "handler":
						ch.Handler = value
					default:
						return nil, fmt.Errorf("unknow http key '%s'", innerKey)
					}
				}

				result = append(result, ch)
			}
		}
	}

	return result, nil
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
