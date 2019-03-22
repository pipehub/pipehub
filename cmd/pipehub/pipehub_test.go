package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pipehub/pipehub"
)

func TestConfigValid(t *testing.T) {
	tests := []struct {
		name      string
		config    config
		assertion require.ErrorAssertionFunc
	}{
		{
			"valid #1",
			config{},
			require.NoError,
		},
		{
			"valid #2",
			config{
				Server: []configServer{
					{},
				},
			},
			require.NoError,
		},
		{
			"multiple servers",
			config{
				Server: []configServer{
					{},
					{},
				},
			},
			require.Error,
		},
		{
			"multiple actions inside a server",
			config{
				Server: []configServer{
					{
						Action: []configServerAction{
							{},
							{},
						},
					},
				},
			},
			require.Error,
		},
		{
			"multiple http inside a server",
			config{
				Server: []configServer{
					{
						HTTP: []configServerHTTP{
							{},
							{},
						},
					},
				},
			},
			require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertion(t, tt.config.valid())
		})
	}
}

func TestConfigToGenerateConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   config
		expected []pipehub.GenerateConfigPipe
	}{
		{
			"success #1",
			config{},
			[]pipehub.GenerateConfigPipe{},
		},
		{
			"success #2",
			config{
				Pipe: []configPipe{
					{
						Path:    "path1",
						Version: "version1",
						Alias:   "alias1",
					},
					{
						Path:    "path2",
						Version: "version2",
						Alias:   "alias2",
					},
				},
			},
			[]pipehub.GenerateConfigPipe{
				{
					Path:    "path1",
					Version: "version1",
					Alias:   "alias1",
				},
				{
					Path:    "path2",
					Version: "version2",
					Alias:   "alias2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.config.toGenerateConfig().Pipe
			require.ElementsMatch(t, tt.expected, actual)
		})
	}
}

func TestConfigToClientConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   config
		expected pipehub.ClientConfig
	}{
		{
			"success #1",
			config{
				Host:   []configHost{},
				Server: []configServer{},
			},
			pipehub.ClientConfig{
				Host:   []pipehub.ClientConfigHost{},
				Server: pipehub.ClientConfigServer{},
			},
		},
		{
			"success #2",
			config{
				Host: []configHost{
					{
						Endpoint: "endpoint1",
						Handler:  "handler1",
					},
					{
						Endpoint: "endpoint2",
						Handler:  "handler2",
					},
				},
				Server: []configServer{
					{
						HTTP: []configServerHTTP{
							{
								Port: 80,
							},
						},
						Action: []configServerAction{
							{
								NotFound: "notFound",
								Panic:    "panic",
							},
						},
					},
				},
			},
			pipehub.ClientConfig{
				Host: []pipehub.ClientConfigHost{
					{
						Endpoint: "endpoint1",
						Handler:  "handler1",
					},
					{
						Endpoint: "endpoint2",
						Handler:  "handler2",
					},
				},
				Server: pipehub.ClientConfigServer{
					HTTP: pipehub.ClientConfigServerHTTP{
						Port: 80,
					},
					Action: pipehub.ClientConfigServerAction{
						NotFound: "notFound",
						Panic:    "panic",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.config.toClientConfig()
			actual.AsyncErrHandler = tt.expected.AsyncErrHandler
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestConfigCtxShutdown(t *testing.T) {
	tests := []struct {
		name     string
		config   config
		deadline time.Duration
		exist    require.BoolAssertionFunc
	}{
		{
			"with deadline",
			config{
				Server: []configServer{
					{
						GracefulShutdown: "1s",
					},
				},
			},
			time.Second,
			require.True,
		},
		{
			"without deadline",
			config{},
			0,
			require.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, ctxCancel := tt.config.ctxShutdown()
			defer ctxCancel()

			deadline, ok := ctx.Deadline()
			tt.exist(t, ok)
			if !ok {
				return
			}

			tdiff := time.Now().Add(tt.deadline).Sub(deadline)
			require.True(t, (tdiff >= 0))
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		expected  config
		assertion require.ErrorAssertionFunc
	}{
		{
			"success #1",
			"loadConfig.success.1.hcl",
			config{},
			require.NoError,
		},
		{
			"success #2",
			"loadConfig.success.2.hcl",
			config{
				Host: []configHost{
					{
						Endpoint: "google",
						Handler:  "base.Default",
					},
				},
				Pipe: []configPipe{
					{
						Path:    "github.com/pipehub/handler",
						Version: "v0.7.0",
						Alias:   "base",
					},
				},
				Server: []configServer{
					{
						GracefulShutdown: "10s",
						HTTP: []configServerHTTP{
							{
								Port: 80,
							},
						},
						Action: []configServerAction{
							{
								NotFound: "base.NotFound",
								Panic:    "base.Panic",
							},
						},
					},
				},
			},
			require.NoError,
		},
		{
			"invalid hcl",
			"loadConfig.fail.1.hcl",
			config{},
			require.Error,
		},
		{
			"decode error",
			"loadConfig.fail.2.hcl",
			config{},
			require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := testdataPath(t, tt.path)

			actual, err := loadConfig(path)
			tt.assertion(t, err)
			if err != nil {
				return
			}

			require.Equal(t, tt.expected, actual)
		})
	}
}

func testdataPath(t *testing.T, name string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic("could not get the caller that invoked testdataPath")
	}

	return fmt.Sprintf("%s/testdata/%s", filepath.Dir(file), name)
}
