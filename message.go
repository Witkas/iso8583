// This file implements decoding: turning raw message bytes into a structured
// Message according to a packager definition. It is deliberately free of any
// human-facing lookups (response-code meanings, MCC descriptions, and so on) —
// those belong to the annotate package. Here we only decode the wire format
// into field values. See pack.go for the inverse (Message -> bytes).
package iso8583

import (
	"errors"
	"fmt"
	"sort"

	"github.com/Witkas/iso8583/packager"
)

// Field is a single decoded data element.
type Field struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
	Raw    []byte `json:"-"`
	Value  string `json:"value"`
}

// Bitmap records which data elements a message declares present.
type Bitmap struct {
	// Bytes is the raw bitmap (8 bytes if primary only, 16 if a secondary
	// block is present).
	Bytes []byte `json:"-"`
	// HasSecondary is true when bit 1 of the primary bitmap was set.
	HasSecondary bool `json:"hasSecondary"`
	present      map[int]bool
}

// IsPresent reports whether data element n is flagged present.
func (b Bitmap) IsPresent(n int) bool { return b.present[n] }

// PresentFields returns the present data-element numbers in ascending order.
// The secondary-bitmap indicator (bit 1) is not reported as a field.
func (b Bitmap) PresentFields() []int {
	out := make([]int, 0, len(b.present))
	for n := range b.present {
		if n == 1 {
			continue
		}
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

// Message is a fully parsed ISO 8583 message.
type Message struct {
	MTI    string        `json:"mti"`
	Bitmap Bitmap        `json:"bitmap"`
	Fields map[int]Field `json:"fields"`
}

// PresentFields returns the parsed field numbers in ascending order.
func (m *Message) PresentFields() []int {
	out := make([]int, 0, len(m.Fields))
	for n := range m.Fields {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

// Error is a parse failure carrying the byte offset at which it occurred, so
// malformed input produces a specific, actionable message rather than a panic.
type Error struct {
	Offset int
	Msg    string
}

func (e *Error) Error() string {
	return fmt.Sprintf("parse error at byte %d: %s", e.Offset, e.Msg)
}

func newErr(offset int, format string, args ...any) *Error {
	return &Error{Offset: offset, Msg: fmt.Sprintf(format, args...)}
}

// Parse decodes raw bytes into a Message using the given packager.
func Parse(raw []byte, p *packager.Packager) (*Message, error) {
	if p == nil {
		return nil, errors.New("parse: nil packager")
	}
	r := &reader{buf: raw}

	mtiBytes, err := r.take(p.MTI.Length)
	if err != nil {
		return nil, newErr(r.pos, "message too short for %d-byte MTI (have %d bytes total)", p.MTI.Length, len(raw))
	}

	msg := &Message{
		MTI:    string(mtiBytes),
		Fields: make(map[int]Field),
	}

	bitmap, err := parseBitmap(r)
	if err != nil {
		return nil, err
	}
	msg.Bitmap = bitmap

	for _, n := range bitmap.PresentFields() {
		def, ok := p.Field(n)
		if !ok {
			return nil, newErr(r.pos, "bitmap flags field %d present, but the packager has no definition for it", n)
		}
		f, err := decodeField(r, n, def)
		if err != nil {
			return nil, err
		}
		msg.Fields[n] = f
	}

	return msg, nil
}

// parseBitmap reads the primary bitmap and, if bit 1 is set, the secondary
// bitmap, returning the set of present field numbers.
func parseBitmap(r *reader) (Bitmap, error) {
	primary, err := r.take(8)
	if err != nil {
		return Bitmap{}, newErr(r.pos, "message too short for 8-byte primary bitmap")
	}
	raw := make([]byte, 0, 16)
	raw = append(raw, primary...)

	present := bitsIn(primary, 0)
	hasSecondary := present[1]

	if hasSecondary {
		secondary, err := r.take(8)
		if err != nil {
			return Bitmap{}, newErr(r.pos, "primary bitmap flags a secondary bitmap (bit 1), but the message is too short to contain it")
		}
		raw = append(raw, secondary...)
		for n := range bitsIn(secondary, 64) {
			present[n] = true
		}
	}

	return Bitmap{Bytes: raw, HasSecondary: hasSecondary, present: present}, nil
}

// bitsIn returns the set of field numbers flagged in an 8-byte bitmap block.
// base is 0 for the primary block (fields 1-64) and 64 for the secondary
// block (fields 65-128). Bit 1 is the most significant bit of the first byte.
func bitsIn(block []byte, base int) map[int]bool {
	out := make(map[int]bool)
	for byteIdx := 0; byteIdx < 8; byteIdx++ {
		for bit := 0; bit < 8; bit++ {
			if block[byteIdx]&(0x80>>uint(bit)) != 0 {
				out[base+byteIdx*8+bit+1] = true
			}
		}
	}
	return out
}

// decodeField reads one data element from the reader per its definition.
func decodeField(r *reader, n int, def packager.FieldDef) (Field, error) {
	length := def.Length

	if prefixDigits := def.Type.LengthPrefixDigits(); prefixDigits > 0 {
		prefix, err := r.take(prefixDigits)
		if err != nil {
			return Field{}, newErr(r.pos, "field %d (%s): message ended while reading its %d-digit length prefix", n, def.Name, prefixDigits)
		}
		length, err = atoiStrict(prefix)
		if err != nil {
			return Field{}, newErr(r.pos-prefixDigits, "field %d (%s): length prefix %q is not numeric", n, def.Name, string(prefix))
		}
		if length > def.MaxLength {
			return Field{}, newErr(r.pos-prefixDigits, "field %d (%s): declared length %d exceeds the field maximum of %d", n, def.Name, length, def.MaxLength)
		}
	}

	value, err := r.take(length)
	if err != nil {
		return Field{}, newErr(r.pos, "field %d (%s): needs %d bytes but only %d remain", n, def.Name, length, r.remaining())
	}

	return Field{
		Number: n,
		Name:   def.Name,
		Raw:    value,
		Value:  string(value),
	}, nil
}

// reader is a cursor over the raw message bytes.
type reader struct {
	buf []byte
	pos int
}

// take returns the next n bytes and advances the cursor, or an error if
// fewer than n bytes remain. The returned slice is a copy, so callers may
// retain it safely.
func (r *reader) take(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("negative read length %d", n)
	}
	if r.pos+n > len(r.buf) {
		return nil, errShort
	}
	out := make([]byte, n)
	copy(out, r.buf[r.pos:r.pos+n])
	r.pos += n
	return out, nil
}

func (r *reader) remaining() int { return len(r.buf) - r.pos }

var errShort = errors.New("short read")

// atoiStrict parses an ASCII decimal length prefix. It rejects any byte that
// is not a digit, so a garbled prefix is caught rather than silently
// truncated (unlike strconv.Atoi on a trimmed string).
func atoiStrict(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, errors.New("empty")
	}
	n := 0
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("non-digit byte %q", c)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
