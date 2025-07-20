package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	makemkvcon = "makemkvcon"
	destdir    = "destdir"
	dbdir      = "dbdir"
)

var (
	cfgFile = ""
	rootCmd = &cobra.Command{
		Use:   "autorip",
		Short: "autorip is a tool to manage ripping media",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			missing := []string{}
			if viper.GetString(makemkvcon) == "" {
				missing = append(missing, makemkvcon)
			}
			if viper.GetString(destdir) == "" {
				missing = append(missing, destdir)
			}
			if viper.GetString(dbdir) == "" {
				missing = append(missing, dbdir)
			}
			if len(missing) > 0 {
				return fmt.Errorf("Required flag(s) %+v not set", missing)
			}
			return nil
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.autorip.yaml)")
	rootCmd.PersistentFlags().String(makemkvcon, "", "path to makemkvcon executable")
	rootCmd.PersistentFlags().String(destdir, "", "path to destination directory")
	rootCmd.PersistentFlags().String(dbdir, "", "path to directory storing databases")
	viper.BindPFlag(makemkvcon, rootCmd.PersistentFlags().Lookup(makemkvcon))
	viper.BindPFlag(destdir, rootCmd.PersistentFlags().Lookup(destdir))
	viper.BindPFlag(dbdir, rootCmd.PersistentFlags().Lookup(dbdir))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".autorip")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func Execute() {
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		os.Exit(1)
	}
}
