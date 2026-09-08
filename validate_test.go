package iso8583

import "testing"

func TestLuhn(t *testing.T) {
	valid := []string{"4111111111111111", "4556737586899855", "79927398713"}
	for _, s := range valid {
		if !luhnValid(s) {
			t.Errorf("luhnValid(%q) = false, want true", s)
		}
	}
	invalid := []string{"4111111111111112", "79927398710", ""}
	for _, s := range invalid {
		if luhnValid(s) {
			t.Errorf("luhnValid(%q) = true, want false", s)
		}
	}
}

func TestValidateClean(t *testing.T) {
	m := NewMessage("0100", map[int]string{
		2:  "4111111111111111", // Luhn-valid
		4:  "000000004500",
		49: "840",
	})
	if got := Validate(m); len(got) != 0 {
		t.Fatalf("expected no findings, got %v", got)
	}
}

func TestValidateFindings(t *testing.T) {
	m := NewMessage("010", map[int]string{ // bad MTI (3 chars)
		2:  "4111111111111112", // Luhn-invalid
		4:  "0000ABCD4500",     // non-numeric amount
		49: "84",               // bad currency (2 digits)
	})
	findings := Validate(m)

	// Errors must sort ahead of warnings.
	if len(findings) == 0 || findings[len(findings)-1].Severity != Warning {
		t.Fatalf("expected the sole warning last; got %v", findings)
	}

	got := map[int]Severity{}
	for _, f := range findings {
		got[f.Field] = f.Severity
	}
	want := map[int]Severity{
		0:  SeverityError, // MTI
		4:  SeverityError, // amount
		49: SeverityError, // currency
		2:  Warning,       // Luhn
	}
	for field, sev := range want {
		if got[field] != sev {
			t.Errorf("field %d: severity %q, want %q (findings: %v)", field, got[field], sev, findings)
		}
	}
}

func TestValidateNonNumericPANIsError(t *testing.T) {
	m := NewMessage("0100", map[int]string{2: "4111-1111-1111"})
	findings := Validate(m)
	if len(findings) != 1 || findings[0].Field != 2 || findings[0].Severity != SeverityError {
		t.Fatalf("expected one PAN error, got %v", findings)
	}
}
