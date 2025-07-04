package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/achernya/autorip/makemkv"
	"github.com/spf13/cobra"
)

func init() {
	parseCmd.Flags().BoolVar(&discOnly, "disc-only", false, "Only print DiscInfo")
	rootCmd.AddCommand(parseCmd)
}

var discOnly = false

var parseCmd = &cobra.Command{
	Use:   "parse [filename]",
	Short: "Parse a makemkvcon robot-output and pretty-print it",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		parser := makemkv.NewParser(f)
		stream := parser.Stream()
		for msg := range stream {
			_, isDiscInfo := msg.Parsed.(*makemkv.DiscInfo)
			if discOnly && !isDiscInfo {
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
