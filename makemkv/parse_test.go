package makemkv

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const sampleLog = `MSG:1005,0,1,"FakeMKV v0.0.1 mock(cpu-release) started","%1 started","FakeMKV v0.0.1 mock(cpu-release)"
PRGT:5018,0,"Scanning CD-ROM devices"
PRGC:5018,0,"Scanning CD-ROM devices"
PRGV:0,0,65536
DRV:0,0,999,0,"BD-RE HL-DT-ST BD-RE FAKE 1.00 SERIAL","",""
TCOUNT:1
CINFO:2,0,"Volume Name"
CINFO:32,0,"VOLUME_ID"
TINFO:0,8,0,"4"
TINFO:0,9,0,"1:00:00"
TINFO:0,10,0,"10.0 GB"
TINFO:0,11,0,"10737418240"
TINFO:0,16,0,"00000.mpls"
SINFO:0,0,1,6201,"Video"
SINFO:0,0,5,0,"V_MPEG2"
SINFO:0,0,6,0,"Mpeg2"
SINFO:0,1,1,6202,"Audio"
SINFO:0,1,2,0,"Surround 5.1"
SINFO:0,1,5,0,"A_DTS"
SINFO:0,1,6,0,"DTS-HD MA"
SINFO:0,2,1,6203,"Subtitles"
SINFO:0,2,3,0,"eng"
SINFO:0,2,4,0,"English"
SINFO:0,2,5,0,"S_HDMV/PGS"
SINFO:0,2,6,0,"PGS"
`

func compareStreamResult(a, b *StreamResult) bool {
	if a.Type != b.Type {
		return false
	}
	return reflect.DeepEqual(a.Parsed, b.Parsed)
}

func TestSuccessWithSampleLog(t *testing.T) {
	want := []*StreamResult{
		// Empty entry to deal with off-by-one due to preincrement below.
		{},
		{
			Type: MessageTag,
			Parsed: &Message{
				Code:    1005,
				Flags:   0,
				Count:   1,
				Message: "FakeMKV v0.0.1 mock(cpu-release) started",
				Format:  "%1 started",
				Params:  []string{"FakeMKV v0.0.1 mock(cpu-release)"},
			},
		},
		{
			Type: ProgressTitleTag,
			Parsed: &ProgressTitle{
				Type: ProgressTotal,
				Code: 5018,
				Id:   0,
				Name: "Scanning CD-ROM devices",
			},
		},
		{
			Type: ProgressCurrentTag,
			Parsed: &ProgressTitle{
				Type: ProgressCurrent,
				Code: 5018,
				Id:   0,
				Name: "Scanning CD-ROM devices",
			},
		},
		{
			Type: ProgressUpdateTag,
			Parsed: &ProgressUpdate{
				Current: 0,
				Total:   0,
				Max:     65536,
			},
		},
		{
			Type: DriveTag,
			Parsed: &Drive{
				Index:     0,
				State:     DriveEmptyClosed,
				Unknown:   999,
				Flags:     0,
				DriveName: "BD-RE HL-DT-ST BD-RE FAKE 1.00 SERIAL",
				DiscName:  "",
				DrivePath: "",
			},
		},
		{
			Parsed: &DiscInfo{
				GenericInfo: GenericInfo{
					Name:       "Volume Name",
					VolumeName: "VOLUME_ID",
				},
				Titles: []TitleInfo{
					{
						GenericInfo: GenericInfo{
							ChapterCount:   "4",
							Duration:       "1:00:00",
							DiskSize:       "10.0 GB",
							DiskSizeBytes:  "10737418240",
							SourceFileName: "00000.mpls",
						},
						Streams: []StreamInfo{
							{
								GenericInfo: GenericInfo{
									Type:       "Video",
									CodecId:    "V_MPEG2",
									CodecShort: "Mpeg2",
								},
							},
							{
								GenericInfo: GenericInfo{
									Type:       "Audio",
									Name:       "Surround 5.1",
									CodecId:    "A_DTS",
									CodecShort: "DTS-HD MA",
								},
							},
							{
								GenericInfo: GenericInfo{
									Type:       "Subtitles",
									LangCode:   "eng",
									LangName:   "English",
									CodecId:    "S_HDMV/PGS",
									CodecShort: "PGS",
								},
							},
						},
					},
				},
			},
		},
	}
	r := strings.NewReader(sampleLog)
	p := NewParser(r)
	count := 0
	for result := range p.Stream() {
		if result.Parsed == nil {
			continue
		}
		count++
		if count >= len(want) {
			gotJson, err := json.Marshal(result)
			if err != nil {
				t.Errorf("error while formatting mismatched struct: %+v", err)
				continue
			}
			t.Errorf("unexpected log line: %s", gotJson)
			continue
		}
		if !compareStreamResult(result, want[count]) {
			gotJson, err := json.Marshal(result)
			if err != nil {
				t.Errorf("error while formatting mismatched struct: %+v", err)
				continue
			}
			wantJson, err := json.Marshal(want[count])
			if err != nil {
				t.Errorf("error while formatting mismatched struct: %+v", err)
				continue
			}
			t.Errorf("got %s, want %s", gotJson, wantJson)
		}
	}
}
