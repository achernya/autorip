package cmd

import (
	"path"
	"sync"

	"github.com/achernya/autorip/db"
	"github.com/achernya/autorip/imdb"
	"github.com/achernya/autorip/makemkv"
	"github.com/achernya/autorip/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	rootCmd.AddCommand(ripCmd)
}

var ripCmd = &cobra.Command{
	Use:   "rip",
	Short: "Auto-detect the inserted disc and rip it",
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := db.OpenDB(path.Join(viper.GetString(dbdir), "autorip.sqlite"))
		if err != nil {
			return err
		}
		mkv := makemkv.New(d, viper.GetString(makemkvcon), viper.GetString(destdir))
		drives, err := scan(mkv)
		if err != nil {
			return err
		}
		analysis, err := analyze(mkv, drives)
		if err != nil {
			return err
		}
		index, err := imdb.OpenIndex(viper.GetString(dbdir))
		if err != nil {
			return err
		}
		defer index.Close()
		i := makemkv.NewIdentifier(index)
		plan, err := i.MakePlan(analysis.DiscInfo)
		if err != nil {
			return err
		}

		t := tui.NewTui()
		p := tea.NewProgram(t)

		cb := func(msg *makemkv.StreamResult, eof bool) {
			if eof {
				// If we send tui.Eof here, then the
				// TUI will end even if there are
				// multiple entries being ripped.
				return
			}
			p.Send(msg)
		}

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer p.Send(tui.Eof{})
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
