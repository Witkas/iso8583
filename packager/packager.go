// Package packager loads ISO 8583 field-layout definitions ("packagers")
// from YAML. A packager describes how the MTI, bitmap, and each data element
// are encoded on the wire so the parser can decode a message without any
// field knowledge hardcoded in Go.
package packager

import (
	"fmt"

	"github.com/Witkas/iso8583/data"
	"gopkg.in/yaml.v3"
)

// Encoding names the wire representation of a message part.
//
// ASCII and BCD are both implemented for data elements; Binary is used for the
// bitmap. In BCD ("binary-coded decimal", also called "packed"), two decimal
// digits share one byte — one per nibble — so an N-digit value occupies
// ceil(N/2) bytes. A BCD variable-length field also carries its length prefix
// in packed form.
type Encoding string

const (
	ASCII  Encoding = "ascii"
	BCD    Encoding = "bcd"
	Binary Encoding = "binary"
)

// FieldType is the length discipline of a data element.
type FieldType string

const (
	// Fixed is a field of a known, constant length.
	Fixed FieldType = "FIXED"
	// LLVAR is a variable-length field with a 2-digit length prefix.
	LLVAR FieldType = "LLVAR"
	// LLLVAR is a variable-length field with a 3-digit length prefix.
	LLLVAR FieldType = "LLLVAR"
)

// LengthPrefixDigits returns the number of length-prefix digits for a
// variable-length field type, or 0 for fixed-length fields.
func (t FieldType) LengthPrefixDigits() int {
	switch t {
	case LLVAR:
		return 2
	case LLLVAR:
		return 3
	default:
		return 0
	}
}

// MTIDef describes the message type indicator.
type MTIDef struct {
	Length   int      `yaml:"length"`
	Encoding Encoding `yaml:"encoding"`
}

// BitmapDef describes the bitmap encoding. Each bitmap block is 8 bytes.
type BitmapDef struct {
	Encoding Encoding `yaml:"encoding"`
}

// FieldDef is the layout of a single data element.
type FieldDef struct {
	Name      string    `yaml:"name"`
	Type      FieldType `yaml:"type"`
	Length    int       `yaml:"length"`    // used by FIXED
	MaxLength int       `yaml:"maxLength"` // used by LLVAR/LLLVAR
	Encoding  Encoding  `yaml:"encoding"`
}

// Packager is a complete dialect definition.
type Packager struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	MTI         MTIDef           `yaml:"mti"`
	Bitmap      BitmapDef        `yaml:"bitmap"`
	Fields      map[int]FieldDef `yaml:"fields"`
}

// Field returns the definition for a data element number, or false if the
// packager does not define it.
func (p *Packager) Field(n int) (FieldDef, bool) {
	f, ok := p.Fields[n]
	return f, ok
}

// Load parses a packager from YAML bytes and validates it.
func Load(b []byte) (*Packager, error) {
	var p Packager
	if err := yaml.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("packager: parsing YAML: %w", err)
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// Default returns the packager shipped and embedded with the binary.
func Default() (*Packager, error) {
	b, err := data.FS.ReadFile("packagers/default.yaml")
	if err != nil {
		return nil, fmt.Errorf("packager: reading embedded default: %w", err)
	}
	return Load(b)
}

func (p *Packager) validate() error {
	if p.MTI.Length <= 0 {
		return fmt.Errorf("packager %q: mti.length must be positive", p.Name)
	}
	if p.Bitmap.Encoding != Binary {
		// ASCII-hex bitmaps are a plausible future dialect, but only binary
		// is implemented by the parser today. Fail loudly rather than
		// silently mis-parsing.
		return fmt.Errorf("packager %q: unsupported bitmap encoding %q (only %q is implemented)",
			p.Name, p.Bitmap.Encoding, Binary)
	}
	for n, f := range p.Fields {
		if n < 2 || n > 128 {
			return fmt.Errorf("packager %q: field %d out of range 2-128", p.Name, n)
		}
		switch f.Type {
		case Fixed:
			if f.Length <= 0 {
				return fmt.Errorf("packager %q: field %d (%s) is FIXED but has no positive length", p.Name, n, f.Name)
			}
		case LLVAR, LLLVAR:
			if f.MaxLength <= 0 {
				return fmt.Errorf("packager %q: field %d (%s) is %s but has no positive maxLength", p.Name, n, f.Name, f.Type)
			}
		default:
			return fmt.Errorf("packager %q: field %d (%s) has unknown type %q", p.Name, n, f.Name, f.Type)
		}
		if f.Encoding != ASCII && f.Encoding != BCD {
			return fmt.Errorf("packager %q: field %d (%s) has unsupported encoding %q (only %q and %q are implemented)",
				p.Name, n, f.Name, f.Encoding, ASCII, BCD)
		}
	}
	return nil
}
