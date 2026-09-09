package iso8583

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/Witkas/iso8583/packager"
)

// TestPackRoundTripsParse is the central guarantee: for a message Parse
// accepts, packing the result reproduces the original bytes exactly.
func TestPackRoundTripsParse(t *testing.T) {
	p := mustDefault(t)
	raw, err := hex.DecodeString(sampleHex)
	if err != nil {
		t.Fatalf("decode sampleHex: %v", err)
	}

	msg, err := Parse(raw, p)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	packed, err := msg.Pack(p)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if !bytes.Equal(packed, raw) {
		t.Fatalf("round-trip mismatch:\n got %x\nwant %x", packed, raw)
	}
}

// TestNewMessagePack builds a message from field values by hand and checks that
// Pack computes a bitmap and length prefixes that Parse reads back identically.
func TestNewMessagePack(t *testing.T) {
	p := mustDefault(t)

	msg := NewMessage("0100", map[int]string{
		2:  "4556737586899855", // LLVAR, prefix computed by Pack
		3:  "000000",
		4:  "000000004500",
		49: "840",
	})
	packed, err := msg.Pack(p)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	// The LLVAR length prefix "16" must precede the 16-digit PAN.
	if !strings.Contains(string(packed), "164556737586899855") {
		t.Errorf("expected LLVAR prefix 16 before PAN, got %q", packed)
	}

	// And it must parse back to the same values.
	got, err := Parse(packed, p)
	if err != nil {
		t.Fatalf("re-Parse: %v", err)
	}
	if got.MTI != "0100" {
		t.Errorf("MTI = %q, want 0100", got.MTI)
	}
	for n, want := range map[int]string{2: "4556737586899855", 3: "000000", 4: "000000004500", 49: "840"} {
		if got.Fields[n].Value != want {
			t.Errorf("field %d = %q, want %q", n, got.Fields[n].Value, want)
		}
	}
}

// TestPackSecondaryBitmap checks that a field above 64 triggers a 16-byte
// bitmap with the secondary indicator set, and round-trips.
func TestPackSecondaryBitmap(t *testing.T) {
	p := mustDefault(t)
	msg := NewMessage("0800", map[int]string{
		70: valueFor(t, p, 70),
	})
	packed, err := msg.Pack(p)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	// MTI(4) + primary(8) + secondary(8) = 20 bytes minimum.
	if len(packed) < 20 {
		t.Fatalf("expected secondary bitmap, packed only %d bytes", len(packed))
	}
	got, err := Parse(packed, p)
	if err != nil {
		t.Fatalf("re-Parse: %v", err)
	}
	if !got.Bitmap.HasSecondary {
		t.Errorf("expected HasSecondary=true")
	}
	if !got.Bitmap.IsPresent(70) {
		t.Errorf("expected field 70 present")
	}
}

// valueFor produces a value of the correct width for a fixed field, so the
// secondary-bitmap test does not depend on a specific field's length.
func valueFor(t *testing.T, p *packager.Packager, n int) string {
	t.Helper()
	def, _ := p.Field(n)
	if def.Type == packager.Fixed {
		return strings.Repeat("0", def.Length)
	}
	return "0"
}

func TestPackRejectsBadInput(t *testing.T) {
	p := mustDefault(t)

	cases := []struct {
		name    string
		msg     *Message
		wantSub string
	}{
		{
			name:    "wrong MTI length",
			msg:     NewMessage("010", map[int]string{4: "000000004500"}),
			wantSub: "MTI",
		},
		{
			name:    "fixed field wrong width",
			msg:     NewMessage("0100", map[int]string{4: "4500"}),
			wantSub: "fixed at",
		},
		{
			name:    "LLVAR value over maximum",
			msg:     NewMessage("0100", map[int]string{2: strings.Repeat("9", 99)}),
			wantSub: "exceeds the field maximum",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.msg.Pack(p)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not mention %q", err, tc.wantSub)
			}
		})
	}
}

func TestPackNilPackager(t *testing.T) {
	msg := NewMessage("0100", map[int]string{4: "000000004500"})
	if _, err := msg.Pack(nil); err == nil {
		t.Fatal("expected error packing with nil packager")
	}
}
