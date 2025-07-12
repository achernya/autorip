package makemkv

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/achernya/autorip/db"
	"github.com/achernya/autorip/discid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type MakeMkv struct {
	DB         *gorm.DB
	makemkvcon string
	session    *db.Session
}

func New(d *gorm.DB, makemkvcon string) *MakeMkv {
	return &MakeMkv{
		DB:         d,
		makemkvcon: makemkvcon,
	}
}

func (m *MakeMkv) sessionIfNeeded() error {
	if m.session != nil {
		return nil
	}
	m.session = &db.Session{}
	return m.DB.Create(m.session).Error
}

func (m *MakeMkv) run(ctx context.Context, cb func(msg *StreamResult, eof bool), args ...string) error {
	rawLog := db.MakeMkvLog{}
	if err := m.DB.Model(m.session).Association("RawLog").Append(&rawLog); err != nil {
		return err
	}
	process, err := NewProcess(ctx, m.makemkvcon, args)
	if err != nil {
		return err
	}
	parser, err := process.Start()
	if err != nil {
		return err
	}
	rawLog.Args = datatypes.NewJSONSlice(process.Args)
	if err := m.DB.Save(&rawLog).Error; err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	defer func() {
		wg.Wait()
		process.Wait()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		stream := parser.Stream()
		for {
			select {
			case <-ctx.Done():
				cb(nil, true)
				return
			case msg, ok := <-stream:
				cb(msg, !ok)
				if !ok {
					return
				}
				if len(msg.Raw) > 0 {
					m.DB.Model(&rawLog).Association("Entry").Append(&db.MakeMkvLogEntry{Entry: msg.Raw})
				}
			}
		}
	}()
	return nil
}

func discInfoToFingerprint(discInfo *DiscInfo) ([]byte, error) {
	disc := &discid.Disc{
		Name:   discInfo.VolumeName,
		Titles: make([]discid.Title, 0),
	}
	for _, t := range discInfo.Titles {
		title := discid.Title{
			Filename: t.SourceFileName,
			Duration: t.Duration,
		}
		if size, err := strconv.ParseInt(t.DiskSizeBytes, 10, 64); err == nil {
			title.Size = size
		}
		disc.Titles = append(disc.Titles, title)
	}

	return discid.Fingerprint(disc)
}

// ScanDrive will invoke `makemkvcon` to find all attached disc drives
// and their state (e.g., is a disc inserted). Note that calling this
// function may perturb any concurrent accesses other processes are
// doing to their own disc drive.
func (m *MakeMkv) ScanDrive() ([]*Drive, error) {
	if err := m.sessionIfNeeded(); err != nil {
		return nil, err
	}

	log.Println("Looking for disc drives")
	result := make([]*Drive, 0)
	ch := make(chan struct{})
	cb := func(msg *StreamResult, eof bool) {
		if eof {
			close(ch)
			return
		}
		switch msg := msg.Parsed.(type) {
		case *Drive:
			// No point in reporting drives that don't exist.
			if msg.State != DriveNoDrive {
				result = append(result, msg)
			}
		}
	}
	// Passing `invalid` as an argument, is not supported by
	// makemkvcon. But it prints drive statuses anyway!
	if err := m.run(context.Background(), cb, "invalid"); err != nil {
		return nil, err
	}
	<-ch

	return result, nil
}

type Analysis struct {
	DriveIndex int
	New        bool
	DiscInfo   *DiscInfo
}

// Analyze finds the first drive with a disc inserted and analyzes the
// contents of that disc, producing a fingerprint. Usually the input
// `drives` comes from ScanDrive, but can be specified manually. The
// only fields checked in Drive as Index and State, and State bust be
// DriveInserted.
func (m *MakeMkv) Analyze(drives []*Drive) (*Analysis, error) {
	if err := m.sessionIfNeeded(); err != nil {
		return nil, err
	}
	if len(drives) == 0 {
		return nil, fmt.Errorf("no disc drives found")
	}
	targetDrive := -1
	for index, drive := range drives {
		if drive.State == DriveInserted {
			targetDrive = index
			break
		}
	}
	if targetDrive == -1 {
		return nil, fmt.Errorf("no disc inserted in any drive")
	}

	var discInfo *DiscInfo = nil
	ch := make(chan struct{})
	cb := func(msg *StreamResult, eof bool) {
		if eof {
			close(ch)
			return
		}
		switch msg := msg.Parsed.(type) {
		case *DiscInfo:
			discInfo = msg
		}
	}
	log.Printf("Analyzing drive %d\n", drives[targetDrive].Index)
	// We pass --noscan here to avoid accessing any other drives
	// to avoid perturbing any concurrent processes working with
	// them.
	if err := m.run(context.Background(), cb, "--noscan", "info", fmt.Sprintf("disc:%d", drives[targetDrive].Index)); err != nil {
		return nil, err
	}
	<-ch

	if discInfo == nil {
		return nil, fmt.Errorf("internal error occurred, no disc info found")
	}

	fp, err := discInfoToFingerprint(discInfo)
	if err != nil {
		return nil, err
	}

	result := db.DiscFingerprint{}
	insert := db.DiscFingerprint{
		Fingerprint: fp,
		Name:        discInfo.Name,
		VolumeName:  discInfo.VolumeName,
	}
	dbx := m.DB.Where("Fingerprint = ?", fp).Attrs(insert).FirstOrCreate(&result)
	if dbx.Error != nil {
		return nil, dbx.Error
	}
	m.session.DiscFingerprintID = &result.ID
	if err := m.DB.Save(m.session).Error; err != nil {
		return nil, err
	}

	analysis := &Analysis{
		DriveIndex: targetDrive,
		New:        dbx.RowsAffected != 0,
		DiscInfo:   discInfo,
	}

	unique := "new"
	if !analysis.New {
		unique = "seen before"
	}
	log.Printf("Found disc %s (%s) = %s [%s]\n", result.VolumeName, result.Name, hex.EncodeToString(result.Fingerprint), unique)
	return analysis, nil
}
