// Package generator implements the logic to create and or update the dynamic files needed to
// include pipes at the project.
//
// Go plugins are buggy as they don't support windows and have several flaws like very limiting
// constraint forcing the plugin and the server to be built with the exact same version of Go, not
// mentioning the vendor problem.
// The other approach is to force the user to edit some files and add import paths that usually
// relly on init functions and global state, this solution is not good enough either.
//
// PipeHub acts somehow different but archiving the same results. Our pipes are built using Go
// modules and we relly on it to fetch and built the dependencies. We need to generate two files
// to make everything works: 'go.mod' and 'dynamic.go'.
//
// The 'go.mod' describe the project that should be imported into the project. 'dynamic.go'
// is used to initialize the pipe.
package generator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// Place where the 'dynamic.go' file need to be written.
const templatePath = "internal/application/generator/template"

// ClientConfig has the information needed by the client to execute.
type ClientConfig struct {
	// The filesystem should point to the root folder of the project.
	Filesystem afero.Fs
	Pipes      []Pipe
}

// Client is used to dynamic generate files that allows pipes to be included at the final build.
// Being more specific, we relly on 2 files: 'dynamic.go' and 'go.mod'.
//
// 'dynamic.go' is used to initialize the pipe and 'go.mod' is used to solve the dependencies.
type Client struct {
	config ClientConfig
}

// Do generate dynamic files to import the Pipes.
func (c Client) Do() error {
	// Init the templates.
	tmpl, err := c.initTemplate()
	if err != nil {
		return errors.Wrap(err, "init template error")
	}

	// Generate the content used to render the templates.
	tmplContent := c.genTemplateContent()

	// Generate the 'dynamic.go' file.
	if err := c.execDynamicTemplate(tmpl, tmplContent); err != nil {
		return errors.Wrap(err, "template 'dynamic.go' execution error")
	}

	// Generate the 'go.mod' file.
	if err := c.execGoModTemplate(tmpl, tmplContent); err != nil {
		return errors.Wrap(err, "template 'go.mod' execution error")
	}

	return nil
}

func (c Client) init() error {
	if c.config.Filesystem == nil {
		return errors.New("missing 'Filesystem'")
	}
	return nil
}

func (c Client) initTemplate() (template.Template, error) {
	files, err := afero.ReadDir(c.config.Filesystem, templatePath)
	if err != nil {
		return template.Template{}, errors.Wrap(err, "read dir error")
	}

	var tmpls template.Template
	for _, file := range files {
		if file.IsDir() {
			return template.Template{}, fmt.Errorf("recursive directory template parse not supported '%s'", file.Name())
		}

		path := filepath.Join(templatePath, file.Name())
		if err := c.embeddTemplate(&tmpls, path); err != nil {
			return template.Template{}, errors.Wrapf(err, "error while embedding template '%s'", path)
		}
	}

	return tmpls, nil
}

func (c Client) embeddTemplate(tmpl *template.Template, path string) error {
	f, err := c.config.Filesystem.Open(path)
	if err != nil {
		return errors.Wrap(err, "open file")
	}
	defer f.Close() // nolint: errcheck

	rawContent, err := ioutil.ReadAll(f)
	if err != nil {
		return errors.Wrap(err, "read file error")
	}
	content := string(rawContent)

	_, err = tmpl.Parse(content)
	if err != nil {
		return errors.Wrap(err, "parse template error")
	}

	return nil
}

func (c Client) genTemplateContent() templateContent {
	var (
		pipes       = make(templateContentPipeSlice, 0, len(c.config.Pipes))
		moduleCount int
	)

	for _, pipe := range c.config.Pipes {
		p := templateContentPipe{
			ImportPath: pipe.ImportPath,
			Alias:      pipe.Alias,
			Revision:   pipe.Version,
			Module:     pipe.Module,
		}

		pathFragments := strings.Split(pipe.ImportPath, "/")
		if pipe.Alias == "" {
			p.Alias = pathFragments[len(pathFragments)-1]
		}
		if pipe.Alias != "" && pipe.Alias != pathFragments[len(pathFragments)-1] {
			p.ImportPathAlias = pipe.Alias
		}
		if pipe.Module != "" {
			moduleCount++
		}
		pipes = append(pipes, p)
	}

	// Sort is extremelly important to ensure that the generated files have always the same format.
	sort.Sort(pipes)

	return templateContent{
		Pipe:            pipes,
		PipeModuleCount: moduleCount,
	}
}

func (c Client) execDynamicTemplate(tmpl template.Template, content templateContent) error {
	// Delete the old file if it exists.
	var (
		path = "internal/application/server/service/pipe/dynamic.go"
		err  = c.config.Filesystem.Remove(path)
	)
	if err != nil && os.IsExist(err) {
		return errors.Wrapf(err, "remove file '%s' error ", path)
	}

	// Create the new file.
	nf, err := c.config.Filesystem.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644) // nolint: gocritic
	if err != nil {
		return errors.Wrapf(err, "create file '%s' error", path)
	}
	defer nf.Close() // nolint: errcheck

	// Fetch the template to be rendered by the file path.
	pipeDynamicTmpl := tmpl.Lookup(filepath.Base(path))
	if pipeDynamicTmpl == nil {
		return fmt.Errorf("couldn't find the template '%s'", path)
	}

	// Execute the template and send the output to the new file.
	err = pipeDynamicTmpl.Execute(nf, content)
	if err != nil {
		return errors.Wrap(err, "template execution error")
	}

	// Force the write on disk.
	if err := nf.Sync(); err != nil {
		return errors.Wrap(err, "sync error")
	}

	return nil
}

func (c Client) execGoModTemplate(tmpl template.Template, content templateContent) error {
	var (
		goModPath       = "go.mod"
		goModBackupPath = "go.mod.backup"
	)

	// Check if the go mod backup file exists, if true, it should generate a error.
	if _, err := c.config.Filesystem.Stat(goModBackupPath); os.IsExist(err) {
		return errors.Wrapf(err, "backup file '%s' already exists", goModBackupPath)
	}

	// Backup the current file.
	if err := c.config.Filesystem.Rename(goModPath, goModBackupPath); err != nil {
		return errors.Wrapf(err, "backup '%s' file error", goModPath)
	}

	// Read the content of the backup file.
	f, err := c.config.Filesystem.Open(goModBackupPath)
	if err != nil {
		return errors.Wrapf(err, "open file '%s' error", goModBackupPath)
	}
	defer f.Close() // nolint: errcheck

	rawPayload, err := ioutil.ReadAll(f)
	if err != nil {
		return errors.Wrapf(err, "read file '%s' error", goModBackupPath)
	}

	// Clean up the file content. First, is needed to remove the generated code from
	// previous executions and then we trim the space to ensure that the file will
	// be correct formatted.
	payload := c.templateModCleanup(string(rawPayload))
	payload = strings.TrimSpace(payload)

	// Create the new file that will have the content from the backup file minus the old
	// generated code, plus the new generated code.
	nf, err := c.config.Filesystem.OpenFile(goModPath, os.O_RDWR|os.O_CREATE, 0644) // nolint: gocritic
	if err != nil {
		return errors.Wrapf(err, "create file '%s' error", goModPath)
	}
	defer nf.Close() // nolint: errcheck

	// First, we write the cleaned payload from the backup file. Then, we add te carriage return
	// to ensure the formatting.
	if _, err = nf.WriteString(payload); err != nil {
		return errors.Wrap(err, "payload writer error")
	}

	// Fetch the template to be rendered by the file path.
	goModTmpl := tmpl.Lookup(goModPath)
	if goModTmpl == nil {
		return fmt.Errorf("couldn't find the template '%s'", goModPath)
	}

	// Execute the template and send the output to the new file.
	if err = goModTmpl.Execute(nf, content); err != nil {
		return errors.Wrap(err, "template execution error")
	}

	// Force the write on disk.
	if err := nf.Sync(); err != nil {
		return errors.Wrap(err, "sync error")
	}

	// Remove the backup file.
	if err = c.config.Filesystem.Remove(goModBackupPath); err != nil {
		return errors.Wrapf(err, "remove '%s' file error", goModBackupPath)
	}

	return nil
}

func (Client) templateModCleanup(payload string) string {
	i := strings.Index(payload, "// Code generated by PipeHub; DO NOT EDIT.")
	if i == -1 {
		return payload
	}
	return payload[:i]
}

// NewClient return a initialized client.
func NewClient(config ClientConfig) (Client, error) {
	c := Client{config: config}
	if err := c.init(); err != nil {
		return c, errors.Wrap(err, "fields validation error")
	}
	return c, nil
}
