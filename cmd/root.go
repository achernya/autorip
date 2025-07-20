package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var (
	makemkvcon = ""
	destDir    = ""
	rootCmd    = &cobra.Command{
		Use:   "autorip",
		Short: "autorip is a tool to manage ripping media",
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&makemkvcon, "makemkvcon", "", "path to makemkvcon executable")
	rootCmd.PersistentFlags().StringVar(&destDir, "destdir", "", "path to destination directory")
	if err := rootCmd.MarkPersistentFlagRequired("makemkvcon"); err != nil {
		panic(err)
	}
	if err := rootCmd.MarkPersistentFlagRequired("destdir"); err != nil {
		panic(err)
	}
}

func Execute() {
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		os.Exit(1)
	}
}
