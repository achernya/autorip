package makemkv

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// https://www.makemkv.com/developers/usage.txt contains some
// (possibly out-of-date, under-specified) documentation for the
// `makemkvcon` and the "robot-mode" message formats.
//
// Frequently the concept of `code` shows up, which is a unique,
// language-netrual identifier for the given message. To my knowledge
// there is no index of these messages published anywhere, which
// effectively makes these fields useless.

type MessageFlags int

const (
	// MessageBoxMask indicates which bits are used for
	// box-information. Since it is valued at 0xf0e, that means
	// that the least-significant bit of the lower nibble is
	// excluded, as is the upper nibble entirely. In essence, this
	// means that all box messages will end in (0x2, 0x4,
	// 0x8). For reasons that are not clear to me, 0x2 seems to be
	// unused.
	MessageBoxMask     = 0xf0e
	MessageBoxOk       = 0x104
	MessageBoxError    = 0x204
	MessageBoxWarning  = 0x404
	MessageBoxYesNo    = 0x308
	MessageBoxYesNoErr = 0x508
	MessageYes         = 0x0
	MessageNo          = 0x1
	MessageDebug       = 0x20
	MessageHidden      = 0x40
	MessageEvent       = 0x80
	MessageHaveUrl     = 0x20000
)

type Message struct {
	// Code is a unique, language-neutral message code.
	Code int
	// Flags contains information about how this message should be rendered.
	Flags MessageFlags
	// Count is the number of parameters in Params
	Count int
	// Message is a pre-formatted message, suitable for output.
	Message string
	// Format is a format-string that produced the message.
	Format string
	// Params are the arguments to the format string.
	Params []string
}

type ProgressType int

const (
	// ProgressTotal represents the name of the overall operation
	// for which progress is being reported.
	ProgressTotal = iota
	// ProgressCurrent represents a detailed step for which
	// progress is being reported, within the broader operation
	// named by `ProgressTotal`.
	ProgressCurrent
)

type ProgressTitle struct {
	Type ProgressType
	// Unique message code.
	Code int
	Id   int
	Name string
}

type ProgressUpdate struct {
	Current int
	Total   int
	Max     int
}

// DriveState indicates what the state of the drive is (e.g., empty,
// open/closed, inserted, missing). Despite looking like flags these
// appear to be strict constants.
type DriveState int

const (
	DriveEmptyClosed = 0
	DriveEmptyOpen   = 1
	DriveInserted    = 2
	DriveLoading     = 3
	DriveNoDrive     = 256
	DriveUnmounting  = 257
)

type DiskFlags int

const (
	DiskDvdFilesPresent    = 0x1
	DiskHdvdFilesPresent   = 0x2
	DiskBlurayFilesPresent = 0x4
	DiskAacsFilesPresent   = 0x8
	DiskBdsvmFilesPresent  = 0x10
)

type Drive struct {
	Index     int
	State     DriveState
	Unknown   int // Always 999?
	Flags     DiskFlags
	DriveName string
	DiscName  string
	DrivePath string
}

const tagName = "makemkv"

type GenericInfo struct {
	Unknown                      string `makemkv:"0" json:",omitempty"`
	Type                         string `makemkv:"1" json:",omitempty"`
	Name                         string `makemkv:"2" json:",omitempty"`
	LangCode                     string `makemkv:"3" json:",omitempty"`
	LangName                     string `makemkv:"4" json:",omitempty"`
	CodecId                      string `makemkv:"5" json:",omitempty"`
	CodecShort                   string `makemkv:"6" json:",omitempty"`
	CodecLong                    string `makemkv:"7" json:",omitempty"`
	ChapterCount                 string `makemkv:"8" json:",omitempty"`
	Duration                     string `makemkv:"9" json:",omitempty"`
	DiskSize                     string `makemkv:"10" json:",omitempty"`
	DiskSizeBytes                string `makemkv:"11" json:",omitempty"`
	StreamTypeExtension          string `makemkv:"12" json:",omitempty"`
	Bitrate                      string `makemkv:"13" json:",omitempty"`
	AudioChannelsCount           string `makemkv:"14" json:",omitempty"`
	AngleInfo                    string `makemkv:"15" json:",omitempty"`
	SourceFileName               string `makemkv:"16" json:",omitempty"`
	AudioSampleRate              string `makemkv:"17" json:",omitempty"`
	AudioSampleSize              string `makemkv:"18" json:",omitempty"`
	VideoSize                    string `makemkv:"19" json:",omitempty"`
	VideoAspectRatio             string `makemkv:"20" json:",omitempty"`
	VideoFrameRate               string `makemkv:"21" json:",omitempty"`
	StreamFlags                  string `makemkv:"22" json:",omitempty"`
	DateTime                     string `makemkv:"23" json:",omitempty"`
	OriginalTitleId              string `makemkv:"24" json:",omitempty"`
	SegmentsCount                string `makemkv:"25" json:",omitempty"`
	SegmentsMap                  string `makemkv:"26" json:",omitempty"`
	OutputFileName               string `makemkv:"27" json:",omitempty"`
	MetadataLanguageCode         string `makemkv:"28" json:",omitempty"`
	MetadataLanguageName         string `makemkv:"29" json:",omitempty"`
	TreeInfo                     string `makemkv:"30" json:",omitempty"`
	PanelTitle                   string `makemkv:"31" json:",omitempty"`
	VolumeName                   string `makemkv:"32" json:",omitempty"`
	OrderWeight                  string `makemkv:"33" json:",omitempty"`
	OutputFormat                 string `makemkv:"34" json:",omitempty"`
	OutputFormatDescription      string `makemkv:"35" json:",omitempty"`
	SeamlessInfo                 string `makemkv:"36" json:",omitempty"`
	PanelText                    string `makemkv:"37" json:",omitempty"`
	MkvFlags                     string `makemkv:"38" json:",omitempty"`
	MkvFlagsText                 string `makemkv:"39" json:",omitempty"`
	AudioChannelLayoutName       string `makemkv:"40" json:",omitempty"`
	OutputCodecShort             string `makemkv:"41" json:",omitempty"`
	OutputConversionType         string `makemkv:"42" json:",omitempty"`
	OutputAudioSampleRate        string `makemkv:"43" json:",omitempty"`
	OutputAudioSampleSize        string `makemkv:"44" json:",omitempty"`
	OutputAudioChannelsCount     string `makemkv:"45" json:",omitempty"`
	OutputAudioChannelLayoutName string `makemkv:"46" json:",omitempty"`
	OutputAudioChannelLayout     string `makemkv:"47" json:",omitempty"`
	OutputAudioMixDescription    string `makemkv:"48" json:",omitempty"`
	Comment                      string `makemkv:"49" json:",omitempty"`
	OffsetSequenceId             string `makemkv:"50" json:",omitempty"`
}

type StreamInfo GenericInfo
type TitleInfo struct {
	GenericInfo
	Streams []StreamInfo
}
type DiscInfo struct {
	GenericInfo
	Titles []TitleInfo
}

type MakeMkvParser struct {
	scanner *bufio.Scanner
}
func updateGenericInfo(info *GenericInfo, records []string) error {
	id, err := strconv.Atoi(records[0])
	if err != nil {
		return err
	}
	_, err = strconv.Atoi(records[1])
	if err != nil {
		return nil
	}
	v := reflect.ValueOf(info)
	for i := 0; i < reflect.Indirect(v).NumField(); i++ {
		// Get the field tag value
		tag := reflect.Indirect(v).Type().Field(i).Tag.Get(tagName)
		
		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}
		currId, err := strconv.Atoi(tag)
		if err != nil {
			return err
		}
		if id != currId {
			continue
		}
		reflect.Indirect(v).Field(i).SetString(records[2])
	}
	return nil
}

func (m *MakeMkvParser) Stream() <- chan interface{} {
	out := make(chan interface{})
	go func() {
		info := GenericInfo{}
		currInfo := ""
		currIndex := "0"
		for m.scanner.Scan() {
			// In case this is a progress message
			progressType := ProgressTotal
			// For parsing disc info messages
			offset := 0
			newIndex := "0"
			
			msg := m.scanner.Text()
			// First, split on `:` to get the message type.
			msgType, rest, found := strings.Cut(msg, ":")
			if !found {
				panic("Invalid line encountered")
			}
			// makemkvcon doesn't produce a valid CSV, since it escapes `"` as `\"` rather than `""`.
			rest = strings.ReplaceAll(rest, "\\\"", "\"\"")
			r := csv.NewReader(strings.NewReader(rest))
			records, err := r.Read()
			if err != nil {
				panic(fmt.Errorf("Unable to parse line %+v: %w", rest, err))
			}
			// Special case for the `[CTS]INFO`, the
			// struct needs to get emitted when the next
			// message changes.
			switch msgType {
			case "SINFO":
				offset++
				fallthrough
			case "TINFO":
				offset++
				fallthrough
			case "CINFO":
				if offset > 0 {
					newIndex = records[offset-1]
				}

			}
			if currInfo != msgType  || currIndex != newIndex {
				// Only actually emit the current info object if we are in an "INFO" context.
				if strings.HasSuffix(currInfo, "INFO") {
					out <- info
					info = GenericInfo{}
				}
				currInfo = msgType
				currIndex = newIndex
			}
			switch msgType {
			case "MSG":
				code, err := strconv.Atoi(records[0])
				if err != nil {
					panic(err)
				}
				flags, err := strconv.Atoi(records[1])
				if err != nil {
					panic(err)
				}
				count, err := strconv.Atoi(records[2])
				if err != nil {
					panic(err)
				}
				out <- Message{
					Code: code,
					Flags: MessageFlags(flags),
					Count: count,
					Message: records[3],
					Format: records[4],
					Params: records[5:],
				}
			case "PRGC":
				progressType = ProgressCurrent
				fallthrough		
			case "PRGT":
				code, err := strconv.Atoi(records[0])
				if err != nil {
					panic(err)
				}
				id, err := strconv.Atoi(records[1])
				if err != nil {
					panic(err)
				}
				out <- ProgressTitle{
					Type: ProgressType(progressType),
					Code: code,
					Id: id,
					Name: records[2],
				}
			case "PRGV":
				current, err := strconv.Atoi(records[0])
				if err != nil {
					panic(err)
				}
				total, err := strconv.Atoi(records[1])
				if err != nil {
					panic(err)
				}
				max, err := strconv.Atoi(records[2])
				if err != nil {
					panic(err)
				}
				out <- ProgressUpdate{
					Current: current,
					Total: total,
					Max: max,
				}
			case "DRV":
				index, err := strconv.Atoi(records[0])
				if err != nil {
					panic(err)
				}
				state, err := strconv.Atoi(records[1])
				if err != nil {
					panic(err)
				}
				unknown, err := strconv.Atoi(records[2])
				if err != nil {
					panic(err)
				}
				flags, err := strconv.Atoi(records[3])
				if err != nil {
					panic(err)
				}
				out <- Drive{
					Index: index,
					State: DriveState(state),
					Unknown: unknown,
					Flags: DiskFlags(flags),
					DriveName: records[4],
					DiscName: records[5],
					DrivePath: records[6],
				}
			case "SINFO":
				fallthrough
			case "TINFO":
				fallthrough
			case "CINFO":
				updateGenericInfo(&info, records[offset:])
			case "TCOUNT":
				// We don't actually care about `TCOUNT`, so we can just ignore it.
			default:
				panic("Unknown message type")
			}

		}
		close(out)
	}()
	return out
}

func NewParser(r io.Reader) *MakeMkvParser {
	return &MakeMkvParser{
		scanner: bufio.NewScanner(r),
	}
}
