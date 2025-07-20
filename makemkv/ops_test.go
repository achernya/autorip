package makemkv

import (
	"encoding/json"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/achernya/autorip/db"
	"google.golang.org/protobuf/proto"

	pb "github.com/achernya/autorip/proto"
)

func TestScanDrive(t *testing.T) {
	d, err := db.OpenDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	mkv := New(d, path.Join("testdata", "fakemkv.sh"), ".")
	got, err := mkv.ScanDrive()
	if err != nil {
		t.Fatal(err)
	}
	want := []*Drive{
		{
			Index:     0,
			State:     2,
			Unknown:   999,
			Flags:     4,
			DriveName: "BD-RE HL-DT-ST BD-RE FAKE 1.00 SERIAL",
			DiscName:  "SomeDisc",
			DrivePath: "/dev/rdisk4",
		},
	}
	if !reflect.DeepEqual(got, want) {
		gotJson, err := json.Marshal(got)
		if err != nil {
			t.Errorf("error while formatting mismatched struct: %+v", err)
		}
		wantJson, err := json.Marshal(want)
		if err != nil {
			t.Errorf("error while formatting mismatched struct: %+v", err)
		}
		t.Errorf("drives mismatch: got %s, want %s", gotJson, wantJson)
	}
}

func TestAnalyze(t *testing.T) {
	d, err := db.OpenDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	mkv := New(d, path.Join("testdata", "fakemkv.sh"), ".")
	drives := []*Drive{
		{
			Index: 0,
			State: 2,
		},
	}
	analysis, err := mkv.Analyze(drives)
	if err != nil {
		t.Fatal(err)
	}
	if !analysis.New {
		t.Error("analysis somehow saw the disc before on an empty db")
	}
	analysis, err = mkv.Analyze(drives)
	if err != nil {
		t.Fatal(err)
	}
	if analysis.New {
		t.Error("analysis incorrectly thinks same disc is new")
	}
}

func TestRip(t *testing.T) {
	d, err := db.OpenDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp("", "autorip")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) //nolint:errcheck
	mkv := New(d, path.Join("testdata", "fakemkv.sh"), dir)
	drive := &Drive{
		Index: 0,
		State: 2,
	}
	plan := &Plan{
		Identity: pb.Title_builder{
			PrimaryTitle: proto.String("Film"),
			StartYear:    proto.Int32(2025),
		}.Build(),
		DiscInfo: &DiscInfo{
			Titles: []TitleInfo{
				{
					GenericInfo: GenericInfo{
						OutputFileName: "title_t0.mkv",
					},
				},
			},
		},
		RipTitles: []*Score{
			{
				TitleIndex: 0,
			},
		},
	}

	// Rip is going to try to move the file, which will fail unless we pre-create it.
	f, err := os.Create(path.Join(dir, "title_t0.mkv"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close() //nolint:errcheck

	cb := func(msg *StreamResult, eof bool) {}
	err = mkv.Rip(drive, plan, cb)
	if err != nil {
		t.Fatal(err)
	}
	f, err = os.Open(path.Join(dir, "Film (2025).mkv"))
	if err != nil {
		t.Error("file that Rip created could not be found")
	}
	f.Close() //nolint:errcheck
}
