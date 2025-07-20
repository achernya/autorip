package cmd

import (
	"context"
	"fmt"

	"github.com/achernya/autorip/imdb"
	"github.com/spf13/cobra"
)

var (
	maxResults int
)

func init() {
	searchCmd.Flags().IntVarP(&maxResults, "max-results", "m", 10, "maximum number of results to show")

	imdbCmd.AddCommand(indexCmd)
	imdbCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(imdbCmd)
}

var (
	imdbCmd = &cobra.Command{
		Use:   "imdb",
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
			// TODO(achernya): fix dir
			index, err := imdb.NewIndex(".")
			if err != nil {
				return err
			}
			defer index.Close()
			return index.Build()
		},
	}
	searchCmd = &cobra.Command{
		Use:   "search",
		Short: "Look up a given IMDb entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index, err := imdb.OpenIndex(".")
			if err != nil {
				return err
			}
			result, err := index.SearchJSON(args[0], maxResults)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
)
