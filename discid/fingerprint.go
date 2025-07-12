package discid

import (
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/cryptobyte"
	"golang.org/x/crypto/cryptobyte/asn1"
)

// Disc represents a single disc, be it Vidoe CD, DVD, or
// Blu-ray. This structure is only used for creatinga SHA-256
// fingerprint.
//
// Disc is represented in ASN.1 as
//
// Disc ::= SEQUENCE {
//   name OctetString,
//   titles SEQUENCE OF Title }

type Disc struct {
	// Name is the UDF Volume label of the disc.
	Name string
	// Titles represents the contents of the disc. In the case of
	// a blu-ray, it's .mpls playlists.
	Titles []Title
}

// Title represents a single Disc title (or track).
//
//	Title ::= SEQUENCE {
//	  filename OctetString,
//	  size INTEGER,
//	  duration OctetString }
type Title struct {
	// Filename is the name of the file on disc.
	Filename string
	// Size is the size of the file in bytes.
	Size int64
	// Duration is a hh:mm:ss string of the file's duration.
	Duration string
}

func Serialize(d *Disc) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("input must not be nil")
	}
	b := cryptobyte.NewBuilder(make([]byte, 0, 16))
	b.AddASN1(asn1.SEQUENCE, func(outer *cryptobyte.Builder) {
		outer.AddASN1OctetString([]byte(d.Name))
		outer.AddASN1(asn1.SEQUENCE, func(inner *cryptobyte.Builder) {
			for _, t := range d.Titles {
				inner.AddASN1(asn1.SEQUENCE, func(child *cryptobyte.Builder) {
					child.AddASN1OctetString([]byte(t.Filename))
					child.AddASN1Int64(t.Size)
					child.AddASN1OctetString([]byte(t.Duration))
				})
			}
		})
	})
	return b.Bytes()
}

// Fingerprint returns a SHA-256 hash of the given Disc.
func Fingerprint(d *Disc) ([]byte, error) {
	b, err := Serialize(d)
	if err != nil {
		return []byte{}, err
	}
	result := sha256.Sum256(b)
	return result[:], nil

}
