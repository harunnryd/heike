package main

import (
	"fmt"
	"os"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/logger"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "heike",
	Short: "Heike AI Runtime",
	Long:  `Heike is a deterministic, guarded, and proactive AI runtime.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cmd)
		if err != nil {
			return err
		}

		logger.Setup(cfg.Server.LogLevel)
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.heike/config.yaml)")
	rootCmd.PersistentFlags().String("server.log_level", config.DefaultServerLogLevel, "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().Int("server.port", config.DefaultServerPort, "server port")
}
