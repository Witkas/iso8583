package annotate

import (
	"strings"
	"testing"

	"github.com/Witkas/iso8583"
)

func mustAnnotator(t *testing.T) *Annotator {
	t.Helper()
	a, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return a
}

func annotateOne(t *testing.T, a *Annotator, number int, value string) AnnotatedField {
	t.Helper()
	msg := &iso8583.Message{
		MTI: "0100",
		Fields: map[int]iso8583.Field{
			number: {Number: number, Name: "field", Value: value},
		},
	}
	res := a.Annotate(msg)
	for _, f := range res.Fields {
		if f.Number == number {
			return f
		}
	}
	t.Fatalf("field %d not in result", number)
	return AnnotatedField{}
}

func TestResponseCodeLookup(t *testing.T) {
	a := mustAnnotator(t)
	tests := map[string]string{
		"00": "Approved",
		"05": "Do not honor",
		"51": "Insufficient funds",
	}
	for code, want := range tests {
		if got := annotateOne(t, a, 39, code).Meaning; got != want {
			t.Errorf("response code %q meaning = %q, want %q", code, got, want)
		}
	}
	if got := annotateOne(t, a, 39, "ZZ").Meaning; !strings.Contains(got, "unknown") {
		t.Errorf("unknown response code meaning = %q, want it to mention 'unknown'", got)
	}
}

func TestProcessingCodeBreakdown(t *testing.T) {
	a := mustAnnotator(t)
	got := annotateOne(t, a, 3, "010000").Meaning
	if !strings.Contains(got, "Cash withdrawal") {
		t.Errorf("processing code meaning = %q, want it to mention 'Cash withdrawal'", got)
	}
	if !strings.Contains(got, "from") || !strings.Contains(got, "to") {
		t.Errorf("processing code meaning = %q, want from/to account parts", got)
	}
}

func TestMCCLookup(t *testing.T) {
	a := mustAnnotator(t)
	if got := annotateOne(t, a, 18, "5411").Meaning; !strings.Contains(got, "Grocery") {
		t.Errorf("MCC 5411 meaning = %q", got)
	}
	if got := annotateOne(t, a, 18, "0000").Meaning; !strings.Contains(got, "unknown") {
		t.Errorf("unknown MCC meaning = %q", got)
	}
}

func TestAmountFormatting(t *testing.T) {
	a := mustAnnotator(t)
	tests := map[string]string{
		"000000004500": "45.00",
		"000000000001": "0.01",
		"000000100000": "1000.00",
	}
	for v, want := range tests {
		if got := annotateOne(t, a, 4, v).Meaning; got != want {
			t.Errorf("amount %q -> %q, want %q", v, got, want)
		}
	}
}

func TestMTIMeaning(t *testing.T) {
	a := mustAnnotator(t)

	// Exact-match table.
	if got := a.mtiMeaning("0100"); got != "Authorization Request" {
		t.Errorf("mti 0100 = %q", got)
	}
	// Composed from digit positions (1200 is not in the names table).
	got := a.mtiMeaning("1200")
	for _, want := range []string{"ISO 8583:1993", "Financial", "Request", "Acquirer"} {
		if !strings.Contains(got, want) {
			t.Errorf("composed mti 1200 = %q, missing %q", got, want)
		}
	}
}
