package generator

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"

	"github.com/stretchr/testify/require"
)

func TestClientDo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pipes     []Pipe
		templates []string
		goMod     string
		output    map[string]string
		err       require.ErrorAssertionFunc
	}{
		{
			name: "single pipe with no module",
			pipes: []Pipe{
				{
					ImportPath: "github.com/pipehub/pipehub",
					Version:    "0.1.0",
					Alias:      "base",
				},
			},
			templates: []string{
				"template/go.mod.tmpl",
				"template/dynamic.go.tmpl",
			},
			goMod: "testdata/go.mod.1.input",
			output: map[string]string{
				"go.mod":     "testdata/go.mod.1.output",
				"dynamic.go": "testdata/dynamic.go.1.output",
			},
			err: require.NoError,
		},
		{
			name: "no pipe",
			templates: []string{
				"template/go.mod.tmpl",
				"template/dynamic.go.tmpl",
			},
			goMod: "testdata/go.mod.1.input",
			output: map[string]string{
				"go.mod":     "testdata/go.mod.2.output",
				"dynamic.go": "testdata/dynamic.go.2.output",
			},
			err: require.NoError,
		},
		{
			name: "multiple pipes with no module",
			pipes: []Pipe{
				{
					ImportPath: "github.com/pipehub/pipehub",
					Version:    "0.1.0",
					Alias:      "base",
				},
				{
					ImportPath: "github.com/diegobernardes/pipehub",
					Version:    "0.4.0",
				},
			},
			templates: []string{
				"template/go.mod.tmpl",
				"template/dynamic.go.tmpl",
			},
			goMod: "testdata/go.mod.1.input",
			output: map[string]string{
				"go.mod":     "testdata/go.mod.3.output",
				"dynamic.go": "testdata/dynamic.go.3.output",
			},
			err: require.NoError,
		},
		{
			name: "single pipe with module",
			pipes: []Pipe{
				{
					ImportPath: "github.com/diegobernardes/pipehub",
					Version:    "0.4.0",
					Module:     "diegobernardes/pipehub",
				},
			},
			templates: []string{
				"template/go.mod.tmpl",
				"template/dynamic.go.tmpl",
			},
			goMod: "testdata/go.mod.1.input",
			output: map[string]string{
				"go.mod":     "testdata/go.mod.4.output",
				"dynamic.go": "testdata/dynamic.go.4.output",
			},
			err: require.NoError,
		},
		{
			name: "multiple pipes with module",
			pipes: []Pipe{
				{
					ImportPath: "github.com/pipehub/pipehub",
					Version:    "0.3.0",
					Module:     "pipehub/pipehub",
					Alias:      "base",
				},
				{
					ImportPath: "github.com/diegobernardes/pipehub",
					Version:    "0.4.0",
					Module:     "diegobernardes/pipehub",
				},
			},
			templates: []string{
				"template/go.mod.tmpl",
				"template/dynamic.go.tmpl",
			},
			goMod: "testdata/go.mod.1.input",
			output: map[string]string{
				"go.mod":     "testdata/go.mod.5.output",
				"dynamic.go": "testdata/dynamic.go.5.output",
			},
			err: require.NoError,
		},
		{
			name: "mixed cases",
			pipes: []Pipe{
				{
					ImportPath: "github.com/pipehub/pipehub",
					Version:    "0.3.0",
					Module:     "pipehub/pipehub",
				},
				{
					ImportPath: "github.com/pipehub/sample",
					Version:    "1.0.0",
					Module:     "pipehub/sample",
					Alias:      "newpipe",
				},
				{
					ImportPath: "github.com/diegobernardes/loadbalancer",
					Version:    "0.5.0",
				},
				{
					ImportPath: "github.com/diegobernardes/ratelimit",
					Version:    "0.6.0",
				},
				{
					ImportPath: "github.com/diegobernardes/proxy",
					Version:    "0.7.0",
				},
			},
			templates: []string{
				"template/go.mod.tmpl",
				"template/dynamic.go.tmpl",
			},
			goMod: "testdata/go.mod.1.input",
			output: map[string]string{
				"go.mod":     "testdata/go.mod.6.output",
				"dynamic.go": "testdata/dynamic.go.6.output",
			},
			err: require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create the temporary dir where the filesystem gonna point to.
			path, err := ioutil.TempDir("", "")
			require.NoError(t, err, "should not have a error during temp file creation")
			defer os.RemoveAll(path) // nolint: errcheck

			// Initialize the filesystem using the temporary file directory.
			fs := afero.NewBasePathFs(afero.NewOsFs(), path)
			initFilesystem(t, fs)
			initFilesystemGoMod(t, fs, tt.goMod)
			initFilesystemTemplate(t, fs, tt.templates)

			// Initialize the client and execute the template generation.
			config := ClientConfig{
				Filesystem: fs,
				Pipes:      tt.pipes,
			}
			c, err := NewClient(config)
			require.NoError(t, err, "initialization should not generate an error")
			tt.err(t, c.Do())

			// Check if the generated files are correct.
			checkFilesystem(t, fs, "go.mod", tt.output["go.mod"])
			checkFilesystem(t, fs, "internal/application/server/service/pipe/dynamic.go", tt.output["dynamic.go"])
		})
	}
}

func initFilesystem(t *testing.T, fs afero.Fs) {
	t.Helper()

	// Create the required directories.
	paths := []string{
		templatePath,
		"internal/application/server/service/pipe",
	}
	for _, path := range paths {
		err := fs.MkdirAll(path, os.ModePerm)
		require.NoErrorf(t, err, "should not have a error creating the folder '%s'", path)
	}
}

func initFilesystemFiles(t *testing.T, fs afero.Fs, origin, destination []string) {
	t.Helper()

	for i, o := range origin {
		// Read the origin file.
		of, err := os.Open(o) // nolint: gocritic
		require.NoErrorf(t, err, "should not have an error creating '%s' file", o)
		defer of.Close() // nolint: errcheck

		// Create the destination file.
		df, err := fs.OpenFile(destination[i], os.O_RDWR|os.O_CREATE, 0644) // nolint: gocritic
		require.NoErrorf(t, err, "should not have an error creating '%s' file", destination[i])
		defer df.Close() // nolint: errcheck

		// Copy the file.
		_, err = io.Copy(df, of)
		require.NoError(t, err, "should not have a error copying the content from '%s' to '%s'", o, destination[i])
	}
}

func initFilesystemGoMod(t *testing.T, fs afero.Fs, path string) {
	t.Helper()

	// Copy the go.mod file to the filesystem.
	origin := []string{path}
	destination := []string{"go.mod"}
	initFilesystemFiles(t, fs, origin, destination)
}

func initFilesystemTemplate(t *testing.T, fs afero.Fs, origin []string) {
	t.Helper()

	// Generate the destination path.
	destination := make([]string, 0, len(origin))
	for _, o := range origin {
		path := filepath.Join(templatePath, filepath.Base(o))
		destination = append(destination, path)
	}

	// Copy the files to the filesystem.
	initFilesystemFiles(t, fs, origin, destination)
}

func checkFilesystem(t *testing.T, fs afero.Fs, actual, expected string) {
	t.Helper()

	// Open actual file to read it.
	af, err := fs.Open(actual)
	require.NoErrorf(t, err, "should not have a error opening file '%s'", actual)

	// Read actual file content.
	rawAfContent, err := ioutil.ReadAll(af)
	require.NoErrorf(t, err, "should not have a error reading file '%s'", actual)

	// Open expected file to read it.
	ef, err := os.Open(expected)
	require.NoErrorf(t, err, "should not have an error creating '%s' file", expected)

	// Read expected file content.
	rawEfContent, err := ioutil.ReadAll(ef)
	require.NoErrorf(t, err, "should not have a error reading file '%s'", actual)

	// Compare file content.
	require.Equal(t, string(rawEfContent), string(rawAfContent), "expected files to be equal")
}
