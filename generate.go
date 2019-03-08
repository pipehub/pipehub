package pipehub

import (
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type generateTemplateContent struct {
	Handler []generateTemplateContentHandler
}

type generateTemplateContentHandler struct {
	Path      string
	PathAlias string // Alias extracted from the path.
	Alias     string // Overwrite the above path alias value.
	Revision  string
}

// GenerateConfig has all the information needed to execute the generate.
type GenerateConfig struct {
	Filesystem afero.Fs
	Handler    []GenerateConfigHandler
}

func (cfg GenerateConfig) toGenerateTemplateContent() generateTemplateContent {
	handler := make([]generateTemplateContentHandler, 0, len(cfg.Handler))
	for _, h := range cfg.Handler {
		pathFragments := strings.Split(h.Path, "/")
		handler = append(handler, generateTemplateContentHandler{
			Path:      h.Path,
			PathAlias: pathFragments[len(pathFragments)-1],
			Alias:     h.Alias,
			Revision:  h.Version,
		})
	}
	return generateTemplateContent{
		Handler: handler,
	}
}

// GenerateConfigHandler has the information needed to represent a handler.
type GenerateConfigHandler struct {
	Path    string
	Version string
	Alias   string
}

// Generate the dynamic files to include custom handlers at the final build.
type Generate struct {
	cfg                GenerateConfig
	goModTmpl          template.Template
	handlerDynamicTmpl template.Template
}

// Do dynamic generate the required files from the configuration file.
func (g *Generate) Do() error {
	content := g.cfg.toGenerateTemplateContent()
	if err := g.doGoMod(content); err != nil {
		return errors.Wrap(err, "go mod generation error")
	}
	if err := g.doHandlerDynamic(content); err != nil {
		return errors.Wrap(err, "handler dynamic generation error")
	}
	return nil
}

func (g *Generate) init() error {
	goModTmpl, err := g.parseTemplate("template/go.mod.tmpl")
	if err != nil {
		return err
	}
	g.goModTmpl = *goModTmpl

	handlerDynamicTmpl, err := g.parseTemplate("template/handler_dynamic.go.tmpl")
	if err != nil {
		return err
	}
	g.handlerDynamicTmpl = *handlerDynamicTmpl

	return nil
}

func (g *Generate) parseTemplate(path string) (*template.Template, error) {
	f, err := g.cfg.Filesystem.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "open file '%s' error", path)
	}
	defer f.Close() // nolint: errcheck

	rawContent, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.Wrapf(err, "read '%s' file error", path)
	}
	content := string(rawContent)

	tmpl, err := template.New(path).Parse(content)
	return tmpl, errors.Wrapf(err, "parse template '%s' error", path)
}

func (g *Generate) doGoMod(content generateTemplateContent) error {
	var (
		goModPath       = "go.mod"
		goModBackupPath = "go.mod.backup"
	)

	// Check if the go mod backup file exists, if true, it should generate a error.
	if _, err := g.cfg.Filesystem.Stat(goModBackupPath); os.IsExist(err) {
		return errors.Wrapf(err, "file '%s' already exists, first this need to be resolved", goModBackupPath)
	}
	if err := g.cfg.Filesystem.Rename(goModPath, goModBackupPath); err != nil {
		return errors.Wrapf(err, "backup '%s' file error", goModPath)
	}

	f, err := g.cfg.Filesystem.Open(goModBackupPath)
	if err != nil {
		return errors.Wrapf(err, "open file '%s' error", goModBackupPath)
	}
	defer f.Close() // nolint: errcheck

	rawPayload, err := ioutil.ReadAll(f)
	if err != nil {
		return errors.Wrapf(err, "read file '%s' error", goModBackupPath)
	}
	payload := g.templateModCleanup(string(rawPayload))
	payload = strings.TrimSpace(payload)

	nf, err := g.cfg.Filesystem.OpenFile(goModPath, os.O_RDWR|os.O_CREATE, 0644) // nolint: gocritic
	if err != nil {
		return errors.Wrapf(err, "create file '%s' error", goModPath)
	}
	defer nf.Close() // nolint: errcheck

	if _, err = nf.WriteString(payload); err != nil {
		return errors.Wrap(err, "payload writer error")
	}
	if _, err = nf.WriteString("\r\n\n"); err != nil { // pular a linha aqui ou no template?
		return errors.Wrap(err, "carriage return and jump line write error")
	}

	if err = g.goModTmpl.Execute(nf, content); err != nil {
		return errors.Wrap(err, "template execution error")
	}

	if err = g.cfg.Filesystem.Remove(goModBackupPath); err != nil {
		return errors.Wrapf(err, "remove '%s' file error", goModBackupPath)
	}

	return nil
}

func (g *Generate) doHandlerDynamic(content generateTemplateContent) error {
	path := "handler_dynamic.go"
	err := g.cfg.Filesystem.Remove(path)
	if err != nil && os.IsExist(err) {
		return errors.Wrapf(err, "remove file '%s' error ", path)
	}

	nf, err := g.cfg.Filesystem.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644) // nolint: gocritic
	if err != nil {
		return errors.Wrapf(err, "create file '%s' error", path)
	}
	defer nf.Close() // nolint: errcheck

	err = g.handlerDynamicTmpl.Execute(nf, content)
	return errors.Wrap(err, "template execution error")
}

func (Generate) templateModCleanup(payload string) string {
	i := strings.Index(payload, "// Code generated by pipehub; DO NOT EDIT.")
	if i == -1 {
		return payload
	}
	return payload[:i]
}

// NewGenerate return the struct that will generate the dynamic code.
func NewGenerate(cfg GenerateConfig) (Generate, error) {
	g := Generate{cfg: cfg}
	return g, errors.Wrap(g.init(), "initialization error")
}
