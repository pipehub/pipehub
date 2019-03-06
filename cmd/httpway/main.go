package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/httpway/httpway"
)

var done = make(chan os.Signal, 1)

func main() {
	var rootCmd = &cobra.Command{Use: "httpway"}
	rootCmd.AddCommand(cmdStart(), cmdGenerate())
	if err := rootCmd.Execute(); err != nil {
		err = errors.Wrap(err, "httpway cli initialization error")
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
	cmd.Flags().StringVarP(&configPath, "config", "c", "./httpway.hcl", "config file path")
	return &cmd
}

func cmdStartRun(configPath *string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		rawCfg, err := loadConfig(*configPath)
		if err != nil {
			err = errors.Wrap(err, "load config error")
			fatal(err)
		}

		if err := rawCfg.valid(); err != nil {
			err = errors.Wrap(err, "invalid config")
			fatal(err)
		}
		cfg := rawCfg.toClientConfig()

		ctxShutdown, ctxShutdownCancel := rawCfg.ctxShutdown()
		defer ctxShutdownCancel()

		c, err := httpway.NewClient(cfg)
		if err != nil {
			err = errors.Wrap(err, "httpway new client error")
			fatal(err)
		}

		if err := c.Start(); err != nil {
			err = errors.Wrap(err, "httpway start error")
			fatal(err)
		}

		wait()

		go func() {
			<-ctxShutdown.Done()
			fmt.Println("httpway did not gracefuly stopped")
			os.Exit(1)
		}()

		if err := c.Stop(ctxShutdown); err != nil {
			err = errors.Wrap(err, "httpway stop error")
			fatal(err)
		}
		fmt.Println("httpway stopped")
	}
}

func cmdGenerate() *cobra.Command {
	var configPath, workspacePath string
	cmd := cobra.Command{
		Use:   "generate",
		Short: "Generate the required code to use the custom handlers",
		Long: `generate is used to create the code to use the custom
	handlers defined at the configuration file.`,
		Run: cmdGenerateRun(&configPath, &workspacePath),
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "./httpway.hcl", "config file path")
	cmd.Flags().StringVarP(&workspacePath, "workspace", "w", "", "workspace path")
	return &cmd
}

func cmdGenerateRun(configPath, workspacePath *string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		rawCfg, err := loadConfig(*configPath)
		if err != nil {
			err = errors.Wrap(err, "load config error")
			fatal(err)
		}
		cfg := rawCfg.toGenerateConfig()

		fs := afero.NewBasePathFs(afero.NewOsFs(), *workspacePath)
		cfg.Filesystem = fs

		g, err := httpway.NewGenerate(cfg)
		if err != nil {
			err = errors.Wrap(err, "httpway generate initialization error")
			fatal(err)
		}

		if err = g.Do(); err != nil {
			err = errors.Wrap(err, "httpway generate execute error")
			fatal(err)
		}
	}
}

func wait() {
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
}

func asyncErrHandler(err error) {
	fmt.Println(errors.Wrap(err, "async error occurred").Error())
	done <- syscall.SIGTERM
}
