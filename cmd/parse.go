package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/achernya/autorip/db"
	"github.com/achernya/autorip/makemkv"
	"github.com/spf13/cobra"
)

var (
	discOnly bool
	logId    int
	filename string
)

func init() {
	parseCmd.Flags().BoolVar(&discOnly, "disc-only", false, "Only print DiscInfo")
	parseCmd.Flags().IntVarP(&logId, "log-id", "l", -1, "If set, parse a previous log instead of a filename")
	parseCmd.Flags().StringVarP(&filename, "filename", "f", "", "filename to parse")
	parseCmd.MarkFlagsMutuallyExclusive("log-id", "filename")
	rootCmd.AddCommand(parseCmd)
}

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse a makemkvcon robot-output and pretty-print it",
	RunE: func(cmd *cobra.Command, args []string) error {
		var r io.Reader
		if logId > 0 {
			d, err := db.OpenDB("autorip.sqlite")
			if err != nil {
				return err
			}
			r, err = db.NewLogReader(d, uint(logId))
			if err != nil {
				return err
			}
		} else {
			var err error
			r, err = os.Open(filename)
			if err != nil {
				return err
			}
		}
		parser := makemkv.NewParser(r)
		stream := parser.Stream()
		for msg := range stream {
			_, isDiscInfo := msg.Parsed.(*makemkv.DiscInfo)
			if discOnly && !isDiscInfo {
				continue
			}
			if msg.Parsed == nil {
				continue
			}
			result, err := json.Marshal(msg.Parsed)
			if err != nil {
				continue
			}
			fmt.Println(string(result))
		}
		return nil
	},
}
