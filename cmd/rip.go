package cmd

import (
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
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t := tui.NewTui()
		p := tea.NewProgram(t)
		process, err := makemkv.NewProcess(makemkvcon, args)
		if err != nil {
			return err
		}
		parser, err := process.Start()
		if err != nil {
			return err
		}
		go func() {
			stream := parser.Stream()
			for msg := range stream {
				p.Send(msg)
			}
			process.Wait()
			p.Send(tui.Eof{})
		}()
		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	},
}
