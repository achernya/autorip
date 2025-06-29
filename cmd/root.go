package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/charmbracelet/fang"
)


var rootCmd = &cobra.Command{
	Use:   "autorip",
	Short: "autorip is a tool to manage ripping media",
}

func Execute() {
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		os.Exit(1)
	}
}
