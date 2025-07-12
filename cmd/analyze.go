package cmd

import (
	"github.com/achernya/autorip/db"
	"github.com/achernya/autorip/makemkv"
	"github.com/spf13/cobra"
)

var (
	driveIndex int
)

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().IntVarP(&driveIndex, "index", "i", -1, "drive to analyze. If set to -1, scan for drives.")
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
		var drives []*makemkv.Drive
		if driveIndex == -1 {
			drives, err = mkv.ScanDrive()
			if err != nil {
				return err
			}
		} else {
			drives = []*makemkv.Drive{
				{
					Index: driveIndex,
					State: makemkv.DriveInserted,
				},
			}
		}
		_, err = mkv.Analyze(drives)
		if err != nil {
			return err
		}
		return nil
	},
}
