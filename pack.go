// This file implements encoding: turning a structured Message back into raw
// wire bytes according to a packager definition. It is the inverse of Parse
// (see message.go). The caller supplies the MTI and the field values; Pack
// derives the bitmap and writes each variable-length field's length prefix, so
// the caller never does bitmap bit math or length-prefix arithmetic by hand.
package iso8583

import (
	"fmt"
	"sort"

	"github.com/Witkas/iso8583lens/packager"
)

// Pack serializes the message to wire bytes using the given packager.
//
// The bitmap is computed from the fields actually present in m.Fields, not from
// m.Bitmap — so a Message assembled by hand needs only its MTI and field values
// set. A secondary bitmap is emitted automatically when any field above 64 is
// present. Fields are written in ascending field-number order.
//
// Pack is the inverse of Parse: for any message Parse accepts, packing the
// result reproduces the original bytes.
func (m *Message) Pack(p *packager.Packager) ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("pack: nil packager")
	}
	if len(m.MTI) != p.MTI.Length {
		return nil, fmt.Errorf("pack: MTI %q is %d bytes, but the packager requires %d", m.MTI, len(m.MTI), p.MTI.Length)
	}

	present := m.PresentFields()
	bitmap, err := buildBitmap(present)
	if err != nil {
		return nil, err
	}

	out := make([]byte, 0, 64)
	out = append(out, []byte(m.MTI)...)
	out = append(out, bitmap...)

	for _, n := range present {
		def, ok := p.Field(n)
		if !ok {
			return nil, fmt.Errorf("pack: field %d is present but the packager has no definition for it", n)
		}
		encoded, err := encodeField(n, m.Fields[n], def)
		if err != nil {
			return nil, err
		}
		out = append(out, encoded...)
	}

	return out, nil
}

// buildBitmap builds the 8-byte primary bitmap (plus an 8-byte secondary block
// when any field above 64 is present). Bit 1 — the most significant bit of the
// first byte — is the secondary-bitmap indicator and is set automatically.
func buildBitmap(fields []int) ([]byte, error) {
	hasSecondary := false
	for _, n := range fields {
		if n < 2 || n > 128 {
			return nil, fmt.Errorf("pack: field number %d out of range 2-128", n)
		}
		if n > 64 {
			hasSecondary = true
		}
	}

	size := 8
	if hasSecondary {
		size = 16
	}
	bitmap := make([]byte, size)

	setBit := func(n int) {
		// Field n lives at bit n (1-indexed, bit 1 = MSB of byte 0).
		idx := n - 1
		bitmap[idx/8] |= 0x80 >> uint(idx%8)
	}
	if hasSecondary {
		setBit(1) // secondary-bitmap indicator
	}
	for _, n := range fields {
		setBit(n)
	}
	return bitmap, nil
}

// encodeField serializes one data element: a length prefix (for LLVAR/LLLVAR)
// followed by the value. Only ASCII field encoding is implemented, matching the
// parser.
func encodeField(n int, f Field, def packager.FieldDef) ([]byte, error) {
	if def.Encoding != packager.ASCII {
		return nil, fmt.Errorf("pack: field %d (%s): unsupported encoding %q (only %q is implemented)", n, def.Name, def.Encoding, packager.ASCII)
	}
	value := f.Value

	if prefixDigits := def.Type.LengthPrefixDigits(); prefixDigits > 0 {
		if len(value) > def.MaxLength {
			return nil, fmt.Errorf("pack: field %d (%s): value length %d exceeds the field maximum of %d", n, def.Name, len(value), def.MaxLength)
		}
		if max := pow10(prefixDigits); len(value) >= max {
			return nil, fmt.Errorf("pack: field %d (%s): value length %d does not fit in a %d-digit length prefix", n, def.Name, len(value), prefixDigits)
		}
		prefix := fmt.Sprintf("%0*d", prefixDigits, len(value))
		return []byte(prefix + value), nil
	}

	// Fixed-length: the value must be exactly the declared width. Pack is
	// strict rather than silently padding, because the correct padding
	// (zero-left for numeric, space-right for alphanumeric) depends on field
	// semantics the schema does not carry.
	if len(value) != def.Length {
		return nil, fmt.Errorf("pack: field %d (%s): value %q is %d bytes, but the field is fixed at %d", n, def.Name, value, len(value), def.Length)
	}
	return []byte(value), nil
}

func pow10(n int) int {
	out := 1
	for i := 0; i < n; i++ {
		out *= 10
	}
	return out
}

// NewMessage builds a Message from an MTI and a set of field values, ready to
// Pack. It is a convenience for callers assembling a message by hand: it fills
// in each Field's Number and leaves Name/Raw for Pack and the packager to
// handle. Field names are cosmetic for packing and can be left empty.
func NewMessage(mti string, fields map[int]string) *Message {
	m := &Message{MTI: mti, Fields: make(map[int]Field, len(fields))}
	nums := make([]int, 0, len(fields))
	for n := range fields {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	for _, n := range nums {
		v := fields[n]
		m.Fields[n] = Field{Number: n, Value: v, Raw: []byte(v)}
	}
	return m
}
