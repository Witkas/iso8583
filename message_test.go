package iso8583

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/Witkas/iso8583lens/packager"
)

// sampleHex is the hand-verified 0100 authorization request also shipped in
// testdata/auth-0100.hex. Fields present: 2,3,4,7,11,12,13,18,41,42,49.
const sampleHex = "303130307238400000c08000" +
	"3136" + "34353536373337353836383939383535" + // DE2 LLVAR 16-digit PAN
	"303030303030" + // DE3 "000000"
	"303030303030303034353030" + // DE4 "000000004500"
	"30393036313230303030" + // DE7 "0906120000"
	"303030313233" + // DE11 "000123"
	"313230303030" + // DE12 "120000"
	"30393036" + // DE13 "0906"
	"35343131" + // DE18 "5411"
	"5445524d30303031" + // DE41 "TERM0001"
	"4d45524348414e5430303030313233" + // DE42 "MERCHANT0000123"
	"383430" // DE49 "840"

func mustDefault(t *testing.T) *packager.Packager {
	t.Helper()
	p, err := packager.Default()
	if err != nil {
		t.Fatalf("loading default packager: %v", err)
	}
	return p
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex in test: %v", err)
	}
	return b
}

func TestParseSampleMessage(t *testing.T) {
	msg, err := Parse(mustHex(t, sampleHex), mustDefault(t))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if msg.MTI != "0100" {
		t.Errorf("MTI = %q, want %q", msg.MTI, "0100")
	}
	if msg.Bitmap.HasSecondary {
		t.Errorf("HasSecondary = true, want false for a primary-only message")
	}

	wantValues := map[int]string{
		2:  "4556737586899855",
		3:  "000000",
		4:  "000000004500",
		7:  "0906120000",
		11: "000123",
		12: "120000",
		13: "0906",
		18: "5411",
		41: "TERM0001",
		42: "MERCHANT0000123",
		49: "840",
	}
	if got := len(msg.Fields); got != len(wantValues) {
		t.Errorf("parsed %d fields, want %d (%v)", got, len(wantValues), msg.PresentFields())
	}
	for n, want := range wantValues {
		f, ok := msg.Fields[n]
		if !ok {
			t.Errorf("field %d missing", n)
			continue
		}
		if f.Value != want {
			t.Errorf("field %d value = %q, want %q", n, f.Value, want)
		}
	}
}

func TestParseSecondaryBitmap(t *testing.T) {
	// Primary bitmap with bit 1 (secondary present) and bit 3 set.
	// Secondary bitmap with bit 64 set -> data element 128.
	primary := []byte{0xA0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	secondary := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}

	var raw []byte
	raw = append(raw, []byte("0100")...)
	raw = append(raw, primary...)
	raw = append(raw, secondary...)
	raw = append(raw, []byte("000000")...)           // DE3, FIXED 6
	raw = append(raw, []byte("1234567890123456")...) // DE128, FIXED 16

	msg, err := Parse(raw, mustDefault(t))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !msg.Bitmap.HasSecondary {
		t.Errorf("HasSecondary = false, want true")
	}
	if !msg.Bitmap.IsPresent(3) || !msg.Bitmap.IsPresent(128) {
		t.Errorf("expected fields 3 and 128 present, got %v", msg.PresentFields())
	}
	if msg.Fields[128].Value != "1234567890123456" {
		t.Errorf("DE128 = %q", msg.Fields[128].Value)
	}
}

func TestParseMalformed(t *testing.T) {
	p := mustDefault(t)

	tests := []struct {
		name    string
		raw     []byte
		wantSub string
	}{
		{
			name:    "too short for MTI",
			raw:     []byte("01"),
			wantSub: "too short for",
		},
		{
			name:    "too short for primary bitmap",
			raw:     []byte("0100\x72\x38"),
			wantSub: "primary bitmap",
		},
		{
			name: "secondary flagged but missing",
			// bit 1 set -> secondary expected, but none follows
			raw:     append([]byte("0100"), 0x80, 0, 0, 0, 0, 0, 0, 0),
			wantSub: "secondary bitmap",
		},
		{
			name: "fixed field runs out of bytes",
			// bitmap: only DE3 present (FIXED 6), but supply 3 bytes
			raw:     append(append([]byte("0100"), 0x20, 0, 0, 0, 0, 0, 0, 0), []byte("000")...),
			wantSub: "field 3",
		},
		{
			name: "llvar non-numeric length prefix",
			// DE2 present (LLVAR); prefix "XY" is not numeric
			raw:     append(append([]byte("0100"), 0x40, 0, 0, 0, 0, 0, 0, 0), []byte("XY1234")...),
			wantSub: "not numeric",
		},
		{
			name: "llvar declared length exceeds remaining",
			// DE2 present; prefix "19" but only a few data bytes follow
			raw:     append(append([]byte("0100"), 0x40, 0, 0, 0, 0, 0, 0, 0), []byte("19123")...),
			wantSub: "field 2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.raw, p)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			var perr *Error
			if !errors.As(err, &perr) {
				t.Fatalf("error type = %T, want *iso8583.Error", err)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestParseUndefinedFieldInBitmap(t *testing.T) {
	// A minimal packager that defines only DE3, but a bitmap that flags DE4.
	mini := &packager.Packager{
		Name:   "mini",
		MTI:    packager.MTIDef{Length: 4, Encoding: packager.ASCII},
		Bitmap: packager.BitmapDef{Encoding: packager.Binary},
		Fields: map[int]packager.FieldDef{
			3: {Name: "Processing Code", Type: packager.Fixed, Length: 6, Encoding: packager.ASCII},
		},
	}
	// bits 3 and 4 set (0x30 in the first byte).
	raw := append([]byte("0100"), 0x30, 0, 0, 0, 0, 0, 0, 0)
	raw = append(raw, []byte("000000")...)

	_, err := Parse(raw, mini)
	if err == nil {
		t.Fatal("expected error for undefined field, got nil")
	}
	if !strings.Contains(err.Error(), "no definition") {
		t.Errorf("error %q does not mention missing definition", err.Error())
	}
}
