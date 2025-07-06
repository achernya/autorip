package cmd

import (
	"github.com/achernya/autorip/db"
	"github.com/achernya/autorip/makemkv"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(analyzeCmd)
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Auto-detect the inserted disc and analyze it",
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := db.OpenDB("autorip.sqlite")
		if err != nil {
			return err
		}
		mkv := makemkv.New(d, makemkvcon)
		_, err = mkv.Analyze()
		if err != nil {
			return err
		}
		return nil
	},
}
