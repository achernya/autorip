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

var (
	driveIndex int
	logid2     int
)

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().IntVarP(&driveIndex, "index", "i", -1, "drive to analyze. If set to -1, scan for drives")
	analyzeCmd.Flags().IntVarP(&logid2, "log-id", "s", -1, "if set, load a previous log-id instead of reading a real disc")
	analyzeCmd.MarkFlagsMutuallyExclusive("index", "log-id")
}

func scan(mkv *makemkv.MakeMkv) ([]*makemkv.Drive, error) {
	if driveIndex == -1 && logid2 == -1 {
		return mkv.ScanDrive()
	}
	return []*makemkv.Drive{
		{
			Index: driveIndex,
			State: makemkv.DriveInserted,
		},
	}, nil
}

func analyze(mkv *makemkv.MakeMkv, drives []*makemkv.Drive) (*makemkv.Analysis, error) {
	if logid2 == -1 {
		t := tui.NewTui()
		p := tea.NewProgram(t)
		cb := func(msg *makemkv.StreamResult, eof bool) {
			if eof {
				return
			}
			// Regardless of what the message was, send it to the TUI
			p.Send(msg)
		}
		var result *makemkv.Analysis
		var err error
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer p.Send(tui.Eof{})
			result, err = mkv.Analyze(drives, cb)
		}()
		if _, err := p.Run(); err != nil {
			return nil, err
		}
		wg.Wait()
		return result, err
	}
	log, err := db.NewLogReader(mkv.DB, uint(logid2))
	if err != nil {
		return nil, err
	}

	parser := makemkv.NewParser(log)
	var discInfo *makemkv.DiscInfo

	for msg := range parser.Stream() {
		switch msg := msg.Parsed.(type) {
		case *makemkv.DiscInfo:
			discInfo = msg
		}
	}

	return &makemkv.Analysis{
		DiscInfo: discInfo,
	}, nil
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Auto-detect the inserted disc and analyze it",
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
		_, err = i.MakePlan(analysis.DiscInfo)
		if err != nil {
			return err
		}
		return nil
	},
}
