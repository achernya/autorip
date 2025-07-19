package discid

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestSerializeNullDisc(t *testing.T) {
	var d *Disc = nil
	_, err := Serialize(d)
	if err == nil {
		t.Errorf("Passed in nil disc but got no error")
	}
}

func TestSerialize(t *testing.T) {
	tests := map[string]struct {
		input    *Disc
		expected []byte
	}{
		"empty disc": {
			input: &Disc{},
			// SEQUENCE { OCTET_STRING {} SEQUENCE {} }
			expected: []byte{0x30, 0x04, 0x04, 0x00, 0x30, 0x00},
		},
		"just name": {
			input: &Disc{
				Name: "Name",
			},
			// SEQUENCE { OCTET_STRING { "Name" } SEQUENCE {} }
			expected: []byte{0x30, 0x08, 0x04, 0x04, 'N', 'a', 'm', 'e', 0x30, 0x00},
		},
		"just title": {
			input: &Disc{
				Titles: []*Title{{}},
			},
			// SEQUENCE { OCTET_STRING {} SEQUENCE { SEQUENCE { OCTET_STRING {} INTEGER { 0 } OCTET_STRING {} } } }
			expected: []byte{0x30, 0x0d, 0x04, 0x00, 0x30, 0x09, 0x30, 0x07, 0x04, 0x00, 0x02, 0x01, 0x00, 0x04, 0x00},
		},
		"full disc": {
			input: &Disc{
				Name: "Disc 1",
				Titles: []*Title{{
					Filename: "00000.mpls",
					Size:     1234,
					Duration: "00:00:01",
				}, {
					Filename: "00001.mpls",
					Size:     5678,
					Duration: "01:00:00",
				}},
			},
			// SEQUENCE {
			//   OCTET_STRING { "Disc 1" }
			//   SEQUENCE {
			//     SEQUENCE { OCTET_STRING { "00000.mpls" } INTEGER { 1234 } OCTET_STRING { "00:00:01" } }
			//     SEQUENCE { OCTET_STRING { "00001.mpls" } INTEGER { 5678 } OCTET_STRING { "01:00:00" } }
			//   }
			// }
			expected: []byte{
				0x30, 0x42, 0x04, 0x06, 0x44, 0x69, 0x73, 0x63, 0x20, 0x31, 0x30, 0x38, 0x30, 0x1a, 0x04, 0x0a,
				0x30, 0x30, 0x30, 0x30, 0x30, 0x2e, 0x6d, 0x70, 0x6c, 0x73, 0x02, 0x02, 0x04, 0xd2, 0x04, 0x08,
				0x30, 0x30, 0x3a, 0x30, 0x30, 0x3a, 0x30, 0x31, 0x30, 0x1a, 0x04, 0x0a, 0x30, 0x30, 0x30, 0x30,
				0x31, 0x2e, 0x6d, 0x70, 0x6c, 0x73, 0x02, 0x02, 0x16, 0x2e, 0x04, 0x08, 0x30, 0x31, 0x3a, 0x30,
				0x30, 0x3a, 0x30, 0x30,
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := Serialize(tt.input)
			if err != nil {
				t.Errorf("unexpected error when serializing %+v", err.Error())
				return
			}
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("got %x; want %x", result, tt.expected)
			}
		})
	}
}
func TestFingerprintNullDisc(t *testing.T) {
	var d *Disc = nil
	_, err := Fingerprint(d)
	if err == nil {
		t.Errorf("Passed in nil disc but got no error")
	}
}

func TestFingerprint(t *testing.T) {
	tests := map[string]struct {
		input    *Disc
		expected string
	}{
		"full disc": {
			input: &Disc{
				Name: "Disc 1",
				Titles: []*Title{{
					Filename: "00000.mpls",
					Size:     1234,
					Duration: "00:00:01",
				}, {
					Filename: "00001.mpls",
					Size:     5678,
					Duration: "01:00:00",
				}},
			},
			expected: "b6075493dee08c318ef7a90d9c252af288e94ee6b72e4a27ccb9245854a421a1",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := Fingerprint(tt.input)
			if err != nil {
				t.Errorf("unexpected error when serializing %+v", err.Error())
				return
			}
			enc := hex.EncodeToString(result)
			if enc != tt.expected {
				t.Errorf("got %x; want %x", enc, tt.expected)
			}
		})
	}
}

func TestAlternateOrdersSameFingerprint(t *testing.T) {
	d1 := &Disc{
		Titles: []*Title{{
			Filename: "00000.mpls",
			Size:     1234,
			Duration: "00:00:01",
		}, {
			Filename: "00001.mpls",
			Size:     5678,
			Duration: "01:00:00",
		}},
	}
	d2 := &Disc{
		Titles: []*Title{{
			Filename: "00001.mpls",
			Size:     5678,
			Duration: "01:00:00",
		}, {
			Filename: "00000.mpls",
			Size:     1234,
			Duration: "00:00:01",
		}},
	}
	f1, err := Fingerprint(d1)
	if err != nil {
		t.Errorf("unexpected error when fingerprinting %+v", err.Error())
		return
	}
	f2, err := Fingerprint(d2)
	if err != nil {
		t.Errorf("unexpected error when fingerprinting %+v", err.Error())
		return
	}
	if !bytes.Equal(f1, f2) {
		t.Errorf("serialization was not order-agnostic")
	}
}
