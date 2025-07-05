package cmd

import (
	"context"
	"fmt"

	"github.com/achernya/autorip/imdb"
	"github.com/spf13/cobra"
)

func init() {
	imdbCmd.AddCommand(indexCmd)
	imdbCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(imdbCmd)
}

var (
	imdbCmd = &cobra.Command{
		Use: "imdb",
		Short: "Collection of subcommands for IMDb",
	}
	indexCmd = &cobra.Command{
		Use:   "index",
		Short: "Build an index of IMDb data",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := imdb.Fetch(context.Background(), ".")
			if err != nil {
				return err
			}
			return imdb.MakeIndex(".")
		},
	}
	searchCmd = &cobra.Command{
		Use: "search",
		Short: "Look up a given IMDb entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := imdb.Search(args[0])
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
)
