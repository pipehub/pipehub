package main

import (
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/httpway/httpway"
)

func main() {
	var rootCmd = &cobra.Command{Use: "httpway"}
	rootCmd.AddCommand(cmdGenerate())
	rootCmd.Execute()
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
