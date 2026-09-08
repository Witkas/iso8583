package iso8583

import "fmt"

// Severity classifies how serious a validation Finding is.
type Severity string

const (
	// Warning marks a value that is suspicious but still well-formed on the
	// wire — e.g. a PAN that fails the Luhn checksum.
	Warning Severity = "warning"
	// SeverityError marks a value that violates the field's expected shape —
	// e.g. a non-numeric amount.
	SeverityError Severity = "error"
)

// Finding is one issue discovered by Validate. Field is 0 for message-level
// findings (e.g. a malformed MTI).
type Finding struct {
	Field    int      `json:"field"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

func (f Finding) String() string {
	if f.Field == 0 {
		return fmt.Sprintf("[%s] MTI: %s", f.Severity, f.Message)
	}
	return fmt.Sprintf("[%s] field %d: %s", f.Severity, f.Field, f.Message)
}

// Validate runs a set of common sanity checks over a parsed message and returns
// any findings, most-serious concerns first. It is advisory: a message can be
// perfectly valid on the wire (and packable) yet still draw warnings, because
// these checks encode business-level conventions (Luhn, numeric amounts) rather
// than the wire format itself. An empty result means nothing suspicious was
// found among the checks implemented.
//
// Checks are keyed to the conventional ISO 8583 field numbers: field 2 (PAN,
// Luhn), fields 4/5/6 (amounts, numeric), fields 49/50/51 (currency codes,
// 3-digit numeric). Dialects that repurpose these numbers may see spurious
// findings; validation is deliberately conservative and never blocks packing.
func Validate(m *Message) []Finding {
	var errs, warns []Finding

	if len(m.MTI) != 4 || !allDigits(m.MTI) {
		errs = append(errs, Finding{0, SeverityError, fmt.Sprintf("MTI %q is not 4 numeric digits", m.MTI)})
	}

	if pan, ok := m.Fields[2]; ok {
		switch {
		case !allDigits(pan.Value):
			errs = append(errs, Finding{2, SeverityError, "PAN contains non-digit characters"})
		case !luhnValid(pan.Value):
			warns = append(warns, Finding{2, Warning, "PAN fails the Luhn checksum"})
		}
	}

	for _, n := range []int{4, 5, 6} {
		if f, ok := m.Fields[n]; ok && f.Value != "" && !allDigits(f.Value) {
			errs = append(errs, Finding{n, SeverityError, fmt.Sprintf("amount %q is not numeric", f.Value)})
		}
	}

	for _, n := range []int{49, 50, 51} {
		if f, ok := m.Fields[n]; ok {
			if len(f.Value) != 3 || !allDigits(f.Value) {
				errs = append(errs, Finding{n, SeverityError, fmt.Sprintf("currency code %q is not a 3-digit number", f.Value)})
			}
		}
	}

	return append(errs, warns...)
}

// allDigits reports whether s is non-empty and all ASCII digits.
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// luhnValid runs the Luhn (mod-10) checksum used by payment card numbers. It
// assumes s is all digits (callers check first) and returns false for empty
// input.
func luhnValid(s string) bool {
	if s == "" {
		return false
	}
	sum := 0
	double := false
	// Walk right to left: every second digit from the rightmost is doubled.
	for i := len(s) - 1; i >= 0; i-- {
		d := int(s[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}
