package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/hcl"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/httpway/httpway"
)

type config struct {
	Handler []configHandler `hcl:"handler"`
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

type configHandler struct {
	Path    string `hcl:"path"`
	Version string `hcl:"version"`
	Alias   string `hcl:"alias"`
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
