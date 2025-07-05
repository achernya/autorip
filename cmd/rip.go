package cmd

import (
	"context"
	"sync"

	"github.com/achernya/autorip/db"
	"github.com/achernya/autorip/makemkv"
	"github.com/achernya/autorip/tui"
	"github.com/spf13/cobra"
	"gorm.io/datatypes"

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
		d, err := db.OpenDB("autorip.sqlite")
		if err != nil {
			return err
		}
		t := tui.NewTui()
		p := tea.NewProgram(t)

		ctx, cancel := context.WithCancel(context.Background())
		process, err := makemkv.NewProcess(ctx, makemkvcon, args)
		if err != nil {
			return err
		}

		session := db.Session{}
		result := d.Create(&session)
		if result.Error != nil {
			return err
		}
		rawLog := db.MakeMkvLog{}
		d.Model(&session).Association("RawLog").Append(&rawLog)

		parser, err := process.Start()
		if err != nil {
			return err
		}
		rawLog.Args = datatypes.NewJSONSlice(process.Args)
		d.Save(&rawLog)

		wg := sync.WaitGroup{}
		defer func() {
			cancel()
			process.Wait()
			wg.Wait()
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			stream := parser.Stream()
			for {
				select {
				case <-ctx.Done():
					p.Send(tui.Eof{})
					return
				case msg, ok := <-stream:
					if !ok {
						p.Send(tui.Eof{})
						return
					}
					p.Send(msg)
					if len(msg.Raw) > 0 {
						d.Model(&rawLog).Association("Entry").Append(&db.MakeMkvLogEntry{Entry: msg.Raw})
					}
				}
			}
		}()
		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	},
}
