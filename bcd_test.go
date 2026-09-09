package iso8583

import (
	"testing"

	"github.com/Witkas/iso8583/data"
	"github.com/Witkas/iso8583/packager"
)

func TestBCDUnitRoundTrip(t *testing.T) {
	cases := []string{"", "0", "45", "123", "000000", "4556737586899855", "840"}
	for _, digits := range cases {
		enc, err := bcdEncode(digits)
		if err != nil {
			t.Fatalf("bcdEncode(%q): %v", digits, err)
		}
		if got := bcdByteLen(len(digits)); got != len(enc) {
			t.Errorf("bcdByteLen(%d) = %d, encoded %d bytes", len(digits), got, len(enc))
		}
		back, err := bcdDecode(enc, len(digits))
		if err != nil {
			t.Fatalf("bcdDecode(% x, %d): %v", enc, len(digits), err)
		}
		if back != digits {
			t.Errorf("round trip %q -> % x -> %q", digits, enc, back)
		}
	}
}

func TestBCDEncodeRejectsNonDigit(t *testing.T) {
	if _, err := bcdEncode("12A4"); err == nil {
		t.Fatal("expected error encoding a non-digit value")
	}
}

func TestBCDDecodeRejectsBadNibble(t *testing.T) {
	// 0xAB has nibbles A and B, neither a decimal digit.
	if _, err := bcdDecode([]byte{0xAB}, 2); err == nil {
		t.Fatal("expected error decoding invalid packed decimal")
	}
}

// bcdPackager loads the shipped BCD demo dialect.
func bcdPackager(t *testing.T) *packager.Packager {
	t.Helper()
	b, err := data.FS.ReadFile("packagers/bcd-demo.yaml")
	if err != nil {
		t.Fatalf("read bcd-demo.yaml: %v", err)
	}
	p, err := packager.Load(b)
	if err != nil {
		t.Fatalf("load bcd-demo packager: %v", err)
	}
	return p
}

// TestBCDMessageRoundTrip packs a message through the BCD dialect and parses it
// back, confirming Pack and Parse agree on the packed encoding — including the
// odd-length currency field (3 digits in 2 bytes) and the packed LLVAR prefix.
func TestBCDMessageRoundTrip(t *testing.T) {
	p := bcdPackager(t)

	fields := map[int]string{
		2:  "4556737586899855", // 16-digit PAN, LLVAR
		3:  "000000",
		4:  "000000004500",
		11: "000123",
		49: "840", // odd length -> 2 bytes
	}
	msg := NewMessage("0100", fields)

	packed, err := msg.Pack(p)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	// BCD packs two digits per byte, so the message is markedly shorter than
	// the ASCII equivalent (which would be 57 bytes for these fields).
	if len(packed) >= 57 {
		t.Errorf("expected BCD packing to be compact, got %d bytes", len(packed))
	}

	got, err := Parse(packed, p)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.MTI != "0100" {
		t.Errorf("MTI = %q, want 0100", got.MTI)
	}
	for n, want := range fields {
		if got.Fields[n].Value != want {
			t.Errorf("field %d = %q, want %q", n, got.Fields[n].Value, want)
		}
	}
}
