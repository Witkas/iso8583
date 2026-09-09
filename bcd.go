package iso8583

import (
	"errors"
	"fmt"

	"github.com/Witkas/iso8583/packager"
)

// bcdByteLen returns the number of bytes needed to hold n decimal digits packed
// two per byte. An odd digit count rounds up (the spare nibble is a zero pad).
func bcdByteLen(digits int) int { return (digits + 1) / 2 }

// bcdEncode packs a decimal digit string into BCD, two digits per byte. Values
// with an odd number of digits are left-padded with a zero nibble, matching the
// right-justified convention, so bcdDecode(..., len(digits)) recovers them
// exactly. The input must be all ASCII digits.
func bcdEncode(digits string) ([]byte, error) {
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return nil, fmt.Errorf("value %q contains a non-digit character", digits)
		}
	}
	s := digits
	if len(s)%2 == 1 {
		s = "0" + s
	}
	out := make([]byte, len(s)/2)
	for i := range out {
		out[i] = (s[2*i]-'0')<<4 | (s[2*i+1] - '0')
	}
	return out, nil
}

// bcdDecode expands BCD bytes into a digit string (two digits per byte) and
// returns the last `digits` characters, undoing the left-pad bcdEncode applies
// to odd-length values. It errors if any nibble is not a decimal digit or if
// fewer than `digits` digits are present.
func bcdDecode(b []byte, digits int) (string, error) {
	buf := make([]byte, 0, len(b)*2)
	for _, by := range b {
		hi, lo := by>>4, by&0x0f
		if hi > 9 || lo > 9 {
			return "", fmt.Errorf("byte %#02x is not valid packed decimal", by)
		}
		buf = append(buf, '0'+hi, '0'+lo)
	}
	if digits > len(buf) {
		return "", errors.New("fewer digits decoded than the field requires")
	}
	return string(buf[len(buf)-digits:]), nil
}

// encodeUnit serializes a digit/character string per the field encoding: raw
// bytes for ASCII, packed nibbles for BCD.
func encodeUnit(s string, enc packager.Encoding) ([]byte, error) {
	if enc == packager.BCD {
		return bcdEncode(s)
	}
	return []byte(s), nil
}

// unitByteLen returns how many wire bytes a value of the given digit/character
// length occupies under the encoding.
func unitByteLen(length int, enc packager.Encoding) int {
	if enc == packager.BCD {
		return bcdByteLen(length)
	}
	return length
}

// decodeUnit turns wire bytes back into the value string per the encoding.
// length is the logical length in digits/characters.
func decodeUnit(raw []byte, length int, enc packager.Encoding) (string, error) {
	if enc == packager.BCD {
		return bcdDecode(raw, length)
	}
	return string(raw), nil
}
