package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pipehub/pipehub/internal"
	"github.com/pipehub/pipehub/internal/application/generator"
	"github.com/pipehub/pipehub/internal/application/server"
	"github.com/pipehub/pipehub/internal/application/server/service/pipe"
	"github.com/pipehub/pipehub/internal/application/server/transport/http"
)

func TestConfigValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    Config
		assertion require.ErrorAssertionFunc
	}{
		{
			"valid #1",
			Config{},
			require.NoError,
		},
		{
			"valid #2",
			Config{
				Core: []configCore{
					{},
				},
			},
			require.NoError,
		},
		{
			"multiple servers",
			Config{
				Core: []configCore{
					{},
					{},
				},
			},
			require.Error,
		},
		{
			"multiple actions inside a server",
			Config{
				Core: []configCore{
					{
						HTTP: []configCoreHTTP{
							{
								Server: []configCoreHTTPServer{
									{
										Action: []configServerHTTPAction{
											{},
											{},
										},
									},
								},
							},
						},
					},
				},
			},
			require.Error,
		},
		{
			"multiple http inside a server",
			Config{
				Core: []configCore{
					{
						HTTP: []configCoreHTTP{
							{
								Server: []configCoreHTTPServer{
									{},
									{},
								},
							},
						},
					},
				},
			},
			require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.assertion(t, tt.config.valid())
		})
	}
}

func TestConfigToGenerator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   Config
		expected generator.ClientConfig
	}{
		{
			"success #1",
			Config{},
			generator.ClientConfig{},
		},
		{
			"success #2",
			Config{
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
			generator.ClientConfig{
				Pipes: []generator.Pipe{
					{
						ImportPath: "path1",
						Version:    "version1",
						Alias:      "alias1",
					},
					{
						ImportPath: "path2",
						Version:    "version2",
						Alias:      "alias2",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := tt.config.ToGenerator().Pipes
			require.ElementsMatch(t, tt.expected.Pipes, actual)
		})
	}
}

func TestConfigToServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   Config
		expected server.ClientConfig
	}{
		{
			"success",
			Config{
				HTTP: []configHTTP{
					{
						Endpoint: "endpoint1",
						Handler:  "handler1",
					},
					{
						Endpoint: "endpoint2",
						Handler:  "handler2",
					},
				},
				Core: []configCore{
					{
						HTTP: []configCoreHTTP{
							{
								Server: []configCoreHTTPServer{
									{
										Listen: []configServerHTTPListen{
											{
												Port: 80,
											},
										},
										Action: []configServerHTTPAction{
											{
												NotFound: "notFound",
												Panic:    "panic",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			server.ClientConfig{
				Pipe: []internal.Pipe{},
				Service: server.ClientConfigService{
					Pipe: server.ClientConfigServicePipe{
						HTTP: pipe.HTTPConfig{
							Entry: []pipe.HTTPConfigEntry{
								{Endpoint: "endpoint1", Handler: "handler1"},
								{Endpoint: "endpoint2", Handler: "handler2"},
							},
							DefaultAction: pipe.HTTPConfigDefaultAction{
								NotFound: "notFound",
								Panic:    "panic",
							},
							Instance: nil,
						},
					},
				},
				Transport: server.ClientConfigTransport{
					HTTP: http.ServerConfig{
						Port: 80,
						Host: []internal.Host{
							{Endpoint: "endpoint1", Handler: "handler1"},
							{Endpoint: "endpoint2", Handler: "handler2"},
						},
						DefaultAction: http.ServerConfigDefaultAction{
							Panic:    "panic",
							NotFound: "notFound",
						},
						HandlerFetcher: nil,
						RoundTripper:   nil,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := tt.config.ToServer()
			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestConfigCtxShutdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   Config
		deadline time.Duration
		exist    require.BoolAssertionFunc
	}{
		{
			"with deadline",
			Config{
				Core: []configCore{
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
			Config{},
			0,
			require.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, ctxCancel, err := tt.config.CtxShutdown()
			require.NoError(t, err, "should not have a error parsing the context with timeout")
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

func TestNewConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		path      string
		expected  Config
		assertion require.ErrorAssertionFunc
	}{
		{
			"success #1",
			"newConfig.success.1.hcl",
			Config{},
			require.NoError,
		},
		{
			"success #2",
			"newConfig.success.2.hcl",
			Config{
				HTTP: []configHTTP{
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
				Core: []configCore{
					{
						GracefulShutdown: "10s",
						HTTP: []configCoreHTTP{
							{
								Server: []configCoreHTTPServer{
									{
										Listen: []configServerHTTPListen{
											{
												Port: 80,
											},
										},
										Action: []configServerHTTPAction{
											{
												NotFound: "base.NotFound",
												Panic:    "base.Panic",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			require.NoError,
		},
		{
			"invalid hcl",
			"newConfig.fail.1.hcl",
			Config{},
			require.Error,
		},
		{
			"decode error",
			"newConfig.fail.2.hcl",
			Config{},
			require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := testdataPath(t, tt.path)
			payload, err := ioutil.ReadFile(path)
			require.NoError(t, err, "should not have an error loading the config file")

			actual, err := NewConfig(payload)
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
