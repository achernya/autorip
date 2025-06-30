package cmd

import (
	"os"
	"time"
	
	"github.com/achernya/autorip/tui"
	"github.com/achernya/autorip/makemkv"
	"github.com/spf13/cobra"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	rootCmd.AddCommand(ripCmd)
}

var ripCmd = &cobra.Command{
	Use:   "rip [temporary filename]",
	Short: "Auto-detect the inserted disc and rip it",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		t := tui.NewTui()
		p := tea.NewProgram(t)
		go func() {
			parser := makemkv.NewParser(f)
			stream := parser.Stream()
			for msg := range stream {
				p.Send(msg)
				// Temporary, for now, to be able to see things
				time.Sleep(100 * time.Millisecond)
			}
			p.Send(tui.Eof{})
		}()
		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	},
}
