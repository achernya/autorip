package cmd

import (
	"sync"

	"github.com/achernya/autorip/db"
	"github.com/achernya/autorip/makemkv"
	"github.com/achernya/autorip/tui"
	"github.com/spf13/cobra"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	rootCmd.AddCommand(ripCmd)
}

var ripCmd = &cobra.Command{
	Use:   "rip",
	Short: "Auto-detect the inserted disc and rip it",
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := db.OpenDB("autorip.sqlite")
		if err != nil {
			return err
		}
		mkv := makemkv.New(d, makemkvcon, destDir)
		drives, err := scan(mkv)
		if err != nil {
			return err
		}
		analysis, err := analyze(mkv, drives)
		if err != nil {
			return err
		}
		plan, err := makemkv.MakePlan(analysis.DiscInfo)
		if err != nil {
			return err
		}

		t := tui.NewTui()
		p := tea.NewProgram(t)

		cb := func(msg *makemkv.StreamResult, eof bool) {
			if eof {
				p.Send(tui.Eof{})
				return
			}
			p.Send(msg)
		}

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := mkv.Rip(drives[analysis.DriveIndex], plan, cb)
			if err != nil {
				panic(err)
			}

		}()

		if _, err := p.Run(); err != nil {
			return err
		}
		wg.Wait()
		return nil
	},
}
