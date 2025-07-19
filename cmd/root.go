package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var (
	makemkvcon = ""
	rootCmd    = &cobra.Command{
		Use:   "autorip",
		Short: "autorip is a tool to manage ripping media",
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&makemkvcon, "makemkvcon", "", "path to makemkvcon executable")
	if err := rootCmd.MarkPersistentFlagRequired("makemkvcon"); err != nil {
		panic(err)
	}
}

func Execute() {
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		os.Exit(1)
	}
}
