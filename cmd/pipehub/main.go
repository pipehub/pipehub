package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/pipehub/pipehub/internal/application/generator"
	"github.com/pipehub/pipehub/internal/application/server"
	"github.com/pipehub/pipehub/internal/infra/config"
)

var (
	version string
	builtAt string
	done    = make(chan os.Signal, 1)
)

func main() {
	var rootCmd = &cobra.Command{Use: "pipehub"}
	rootCmd.AddCommand(cmdStart(), cmdGenerate(), cmdVersion())
	if err := rootCmd.Execute(); err != nil {
		err = errors.Wrap(err, "pipehub cli initialization error")
		fatal(err)
	}
}

func cmdStart() *cobra.Command {
	var configPath string
	cmd := cobra.Command{
		Use:   "start",
		Short: "Start the application",
		Long:  `Start the application server.`,
		Run:   cmdStartRun(&configPath),
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "./pipehub.hcl", "config file path")
	return &cmd
}

func cmdStartRun(configPath *string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		payload, err := ioutil.ReadFile(*configPath)
		if err != nil {
			err = errors.Wrap(err, "load file error")
			fatal(err)
		}

		ccfg, err := config.NewConfig(payload)
		if err != nil {
			err = errors.Wrap(err, "config initialization error")
			fatal(err)
		}

		cfg, err := ccfg.ToServer()
		if err != nil {
			err = errors.Wrap(err, "invalid config load")
			fatal(err)
		}
		cfg.Transport.HTTP.AsyncErrorHandler = asyncErrHandler

		c := server.NewClient(cfg)
		if err := c.Start(); err != nil {
			err = errors.Wrap(err, "pipehub start error")
			fatal(err)
		}

		wait()

		ctxShutdown, ctxShutdownCancel, err := ccfg.CtxShutdown()
		if err != nil {
			err = errors.Wrap(err, "context initialization error")
			fatal(err)
		}
		defer ctxShutdownCancel()

		go func() {
			<-ctxShutdown.Done()
			if ctxShutdown.Err() == context.Canceled {
				return
			}
			fmt.Println("pipehub did not gracefuly stopped")
			os.Exit(1)
		}()

		if err := c.Stop(ctxShutdown); err != nil {
			err = errors.Wrap(err, "pipehub stop error")
			fatal(err)
		}
		fmt.Println("pipehub stopped")
	}
}

func cmdGenerate() *cobra.Command {
	var configPath, workspacePath string
	cmd := cobra.Command{
		Use:   "generate",
		Short: "Generate the required code to use the custom pipes",
		Long: `generate is used to create the code to use the custom
	pipes defined at the configuration file.`,
		Run: cmdGenerateRun(&configPath, &workspacePath),
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "./pipehub.hcl", "config file path")
	cmd.Flags().StringVarP(&workspacePath, "workspace", "w", "", "workspace path")
	return &cmd
}

func cmdGenerateRun(configPath, workspacePath *string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		payload, err := ioutil.ReadFile(*configPath)
		if err != nil {
			err = errors.Wrap(err, "load file error")
			fatal(err)
		}

		ccfg, err := config.NewConfig(payload)
		if err != nil {
			err = errors.Wrap(err, "config initialization error")
			fatal(err)
		}

		cfg := ccfg.ToGenerator()
		fs := afero.NewBasePathFs(afero.NewOsFs(), *workspacePath)
		cfg.Filesystem = fs

		g, err := generator.NewClient(cfg)
		if err != nil {
			err = errors.Wrap(err, "pipehub generator initialization error")
			fatal(err)
		}

		if err = g.Do(); err != nil {
			err = errors.Wrap(err, "pipehub generator execute error")
			fatal(err)
		}
	}
}

func cmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Version the application",
		Long:  `Version the application server.`,
		Run:   cmdVersionRun,
	}
}

func cmdVersionRun(cmd *cobra.Command, _ []string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "PipeHub:")
	if version != "" {
		fmt.Fprintln(w, "\t  Version:\t", version)
	}
	fmt.Fprintln(w, "\t  Go version:\t", runtime.Version())
	if builtAt != "" {
		fmt.Fprintln(w, "\t  Built At:\t", builtAt)
	}
	w.Flush()
}
